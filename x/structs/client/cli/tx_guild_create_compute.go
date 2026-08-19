package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"structs/x/structs/types"
)

// How many nonces to try between wall-clock reads. Small enough that a refresh
// is never late by anything a human would notice, large enough that the clock
// read is not on the hashing path.
const charterClockCheckNonces = 4096

/* charterConsent is the output of guild-charter-consent, read back here.
 *
 * Carrying the guild's shape as well as the signature is what lets this command
 * refuse a mismatch up front. The alternative is discovering after weeks of
 * mining that the founder signed for a different substation, and the chain's
 * error for that is an indistinguishable bad signature.
 */
type charterConsent struct {
	FounderPlayerId   string `json:"founder_player_id"`
	Address           string `json:"address"`
	ProofPubKey       string `json:"proof_pub_key"`
	ProofSignature    string `json:"proof_signature"`
	ReactorId         string `json:"reactor_id"`
	EntrySubstationId string `json:"entry_substation_id"`
	Endpoint          string `json:"endpoint"`
	Anchor            uint64 `json:"anchor"`
}

/* Mines the global charter puzzle and broadcasts the guild that solves it.
 *
 * One puzzle for the whole chain, reset by every guild founded on a proof, so
 * this is a race against everyone else mining. Two consequences shape the loop.
 * The anchor is part of the preimage, so when somebody else wins, every nonce
 * ground so far is worthless and the search restarts from the new anchor -- this
 * is not an error, it is the mechanism. And difficulty falls as the puzzle ages,
 * so it is re-read as we go, and --difficulty-target-start lets a would-be
 * founder sleep until the puzzle has decayed to something their hardware can
 * finish.
 */
func CmdGuildCreateCompute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guild-create-compute [reactor ID]",
		Short: "Do the work to found a Guild",
		Long: `Do the work to found a Guild.

Mines the chain-global charter puzzle and broadcasts MsgGuildCreate with the
solution. The puzzle gets easier the longer it goes unsolved and resets to its
hardest every time a guild is founded with a proof, so expect to be interrupted
and to restart: that is the race working as intended.

By default the guild is founded for you. Pass --consent-file to found one for
another player from the consent they signed with guild-charter-consent; you are
credited as its charter solver either way.

Founding a guild you are not already in makes you leave your current guild. A
guild owner cannot found a second guild -- transfer the first away, or found for
somebody else with a consent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			argReactorId := args[0]
			argEndpoint, _ := cmd.Flags().GetString("endpoint")
			argEntrySubstationId, _ := cmd.Flags().GetString("entry-substation-id")
			difficultyTargetStart, _ := cmd.Flags().GetInt("difficulty_target_start")
			pollInterval, _ := cmd.Flags().GetDuration("poll-interval")
			refreshInterval, _ := cmd.Flags().GetDuration("refresh-interval")

			queryClient := types.NewQueryClient(clientCtx)

			solverAddress := clientCtx.GetFromAddress().String()
			solverResults, solverErr := queryClient.Address(context.Background(), &types.QueryGetAddressRequest{
				Address: solverAddress,
			})
			if solverErr != nil {
				return solverErr
			}
			if solverResults.PlayerId == "" {
				return fmt.Errorf("address %s is not registered to a player", solverAddress)
			}
			solverPlayerId := solverResults.PlayerId

			founderPlayerId := solverPlayerId
			var consent charterConsent

			consentFile, _ := cmd.Flags().GetString("consent-file")
			if consentFile != "" {
				consentBytes, readErr := os.ReadFile(consentFile)
				if readErr != nil {
					return readErr
				}
				if unmarshalErr := json.Unmarshal(consentBytes, &consent); unmarshalErr != nil {
					return unmarshalErr
				}

				/* Refuse a consent that does not describe the guild being asked
				 * for. The founder signed over all of it, so a disagreement here
				 * is a proof that will be rejected however long it takes to find.
				 */
				if consent.ReactorId != argReactorId {
					return fmt.Errorf("consent is for reactor %s, not %s", consent.ReactorId, argReactorId)
				}
				if consent.EntrySubstationId != argEntrySubstationId {
					return fmt.Errorf("consent is for entry substation %q, not %q", consent.EntrySubstationId, argEntrySubstationId)
				}
				if consent.Endpoint != argEndpoint {
					return fmt.Errorf("consent is for endpoint %q, not %q", consent.Endpoint, argEndpoint)
				}
				if consent.FounderPlayerId == "" || consent.ProofPubKey == "" || consent.ProofSignature == "" {
					return fmt.Errorf("consent file is missing the founder, pubkey or signature")
				}

				founderPlayerId = consent.FounderPlayerId
				fmt.Printf("Founding for player %s on their consent \n", founderPlayerId)
			}

			/* Re-reads the puzzle, sleeping while it is still harder than this
			 * founder is willing to spend cycles on. Difficulty falls with age,
			 * so waiting is a real strategy rather than idling.
			 */
			readCharter := func() (*types.QueryGuildCharterResponse, error) {
				for {
					state, stateErr := queryClient.GuildCharter(context.Background(), &types.QueryGuildCharter{})
					if stateErr != nil {
						return nil, stateErr
					}

					if difficultyTargetStart <= 0 || int(state.Difficulty) <= difficultyTargetStart {
						return state, nil
					}

					fmt.Printf("Difficulty is %d, waiting for %d \n", state.Difficulty, difficultyTargetStart)
					time.Sleep(pollInterval)
				}
			}

			charter, charterErr := readCharter()
			if charterErr != nil {
				return charterErr
			}

			if consentFile != "" && consent.Anchor != charter.Anchor {
				return fmt.Errorf("consent is bound to anchor %d but the current puzzle is anchored at %d; ask the founder to sign again", consent.Anchor, charter.Anchor)
			}

			fmt.Printf("Charter puzzle anchored at %d, age %d, difficulty %d \n", charter.Anchor, charter.Age, charter.Difficulty)
			fmt.Println("Starting Charter work...")

			var proof string
			var nonce string

			for nonce == "" {
				anchor := charter.Anchor
				attempt := uint64(0)
				lastRead := time.Now()

				for {
					/* Refreshing is rate-limited by the clock rather than by a
					 * nonce count. A puzzle that decays over weeks does not need
					 * watching more than occasionally, and a nonce interval alone
					 * would mean hundreds of queries a second on fast hardware.
					 * The counter is only here to keep the clock read off the
					 * hashing path.
					 */
					if (attempt%charterClockCheckNonces) == 0 && time.Since(lastRead) >= refreshInterval {
						lastRead = time.Now()

						refreshed, refreshErr := readCharter()
						if refreshErr != nil {
							return refreshErr
						}

						if int(refreshed.Difficulty) != int(charter.Difficulty) {
							fmt.Printf("Difficulty Change: %d \n", refreshed.Difficulty)
						}
						charter = refreshed

						/* Somebody else founded a guild. Every nonce ground so
						 * far was bound to the old anchor and is dead, and so is
						 * a consent signed against it.
						 */
						if charter.Anchor != anchor {
							fmt.Printf("Puzzle reset: a guild was founded, new anchor is %d \n", charter.Anchor)
							if consentFile != "" {
								return fmt.Errorf("the founder's consent was bound to anchor %d and is now stale; ask them to sign against %d", anchor, charter.Anchor)
							}
							break
						}
					}

					candidateNonce := strconv.FormatUint(attempt, 10)
					hashInput := types.GuildCharterWorkInput(clientCtx.ChainID, solverPlayerId, founderPlayerId, anchor, candidateNonce)
					hashOutput := types.HashBuild(hashInput)

					if valid, _ := types.HashBuildAndCheckDifficulty(hashInput, hashOutput, charter.Age, charter.DifficultyRange); valid {
						fmt.Println("")
						fmt.Println("Charter work Complete!")
						fmt.Println(hashInput)
						fmt.Println(hashOutput)
						proof = hashOutput
						nonce = candidateNonce
						break
					}

					attempt++
				}
			}

			msg := &types.MsgGuildCreate{
				Creator:           solverAddress,
				ReactorId:         argReactorId,
				Endpoint:          argEndpoint,
				EntrySubstationId: argEntrySubstationId,
				Proof:             proof,
				Nonce:             nonce,
			}

			if consentFile != "" {
				msg.FounderPlayerId = consent.FounderPlayerId
				msg.Address = consent.Address
				msg.ProofPubKey = consent.ProofPubKey
				msg.ProofSignature = consent.ProofSignature
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().String("endpoint", "", "Guild endpoint URL")
	cmd.Flags().String("entry-substation-id", "", "Substation new members are connected to")
	cmd.Flags().String("consent-file", "", "JSON consent from guild-charter-consent, to found a guild for another player")
	cmd.Flags().IntP("difficulty_target_start", "D", 0, "Do not spend cycles until the puzzle has decayed to this difficulty or easier (1-64)")
	cmd.Flags().Duration("poll-interval", 5*time.Minute, "How long to sleep between checks while waiting for difficulty_target_start")
	cmd.Flags().Duration("refresh-interval", 15*time.Second, "How often to re-read the puzzle while mining, to notice a reset or a difficulty drop")

	return cmd
}
