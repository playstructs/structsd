package cli

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/spf13/cobra"

	"structs/x/structs/types"
)

/* Signs a founder's consent to have their guild founded by somebody else.
 *
 * This broadcasts nothing. It is the founder's half of a charter: the pool that
 * will do the mining needs the signature *before* it starts, because a charter is
 * a race and a winning nonce cannot wait for a round trip to the founder. The
 * output goes to the solver over any channel at all, then rides along in their
 * MsgGuildCreate.
 *
 * Every field of the guild is bound into the signature, so what comes out is
 * consent to found one exact guild and not a bearer token for any guild. Change
 * your mind about the substation and you sign again.
 */
func CmdGuildCharterConsent() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guild-charter-consent [reactor ID]",
		Short: "Sign consent for another player to found your guild",
		Long: `Sign consent for another player to found your guild.

Produces the founder-consent fields for guild-create-compute. Nothing is
broadcast and no fees are paid. Hand the output to whoever is mining.

The signature is bound to the reactor, entry substation, endpoint and the
current puzzle anchor. If any of those change -- including the anchor, which
moves whenever any guild on the chain is founded with a proof -- sign again.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			argReactorId := args[0]
			argEndpoint, _ := cmd.Flags().GetString("endpoint")
			argEntrySubstationId, _ := cmd.Flags().GetString("entry-substation-id")

			queryClient := types.NewQueryClient(clientCtx)

			consentingAddress := clientCtx.GetFromAddress().String()
			addressResults, addressErr := queryClient.Address(context.Background(), &types.QueryGetAddressRequest{
				Address: consentingAddress,
			})
			if addressErr != nil {
				return addressErr
			}
			if addressResults.PlayerId == "" {
				return fmt.Errorf("address %s is not registered to a player", consentingAddress)
			}

			// The anchor may be pinned so a founder can pre-sign for a puzzle
			// they are watching rather than the one the node happens to report.
			anchor, _ := cmd.Flags().GetUint64("anchor")
			if anchor == 0 {
				charter, charterErr := queryClient.GuildCharter(context.Background(), &types.QueryGuildCharter{})
				if charterErr != nil {
					return charterErr
				}
				anchor = charter.Anchor
			}

			consentInput := types.GuildCharterConsentInput(addressResults.PlayerId, argReactorId, argEntrySubstationId, argEndpoint, anchor)

			signature, pubKey, signErr := clientCtx.Keyring.Sign(clientCtx.FromName, []byte(consentInput), signingtypes.SignMode_SIGN_MODE_DIRECT)
			if signErr != nil {
				return signErr
			}

			consent := struct {
				FounderPlayerId   string `json:"founder_player_id"`
				Address           string `json:"address"`
				ProofPubKey       string `json:"proof_pub_key"`
				ProofSignature    string `json:"proof_signature"`
				ReactorId         string `json:"reactor_id"`
				EntrySubstationId string `json:"entry_substation_id"`
				Endpoint          string `json:"endpoint"`
				Anchor            uint64 `json:"anchor"`
			}{
				FounderPlayerId:   addressResults.PlayerId,
				Address:           consentingAddress,
				ProofPubKey:       hex.EncodeToString(pubKey.Bytes()),
				ProofSignature:    hex.EncodeToString(signature),
				ReactorId:         argReactorId,
				EntrySubstationId: argEntrySubstationId,
				Endpoint:          argEndpoint,
				Anchor:            anchor,
			}

			output, marshalErr := json.MarshalIndent(consent, "", "  ")
			if marshalErr != nil {
				return marshalErr
			}

			return clientCtx.PrintString(string(output) + "\n")
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().String("endpoint", "", "Guild endpoint URL")
	cmd.Flags().String("entry-substation-id", "", "Substation new members are connected to")
	cmd.Flags().Uint64("anchor", 0, "Puzzle anchor to bind to (default: the current one)")

	return cmd
}
