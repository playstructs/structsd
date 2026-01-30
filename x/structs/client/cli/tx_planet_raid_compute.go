package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"structs/x/structs/types"

	"github.com/spf13/cobra"

	//sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"context"
	"fmt"
	"time"

	"crypto/sha256"
	"encoding/hex"
)

var _ = strconv.Itoa(0)

func CmdPlanetRaidCompute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "planet-raid-compute [fleet id]",
		Short: "Do the work to raid a planet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			argFleetId := args[0]

			var argProof string
			var argNonce string

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			difficultyTargetStart, _ := cmd.Flags().GetInt("difficulty_target_start")

			queryClient := types.NewQueryClient(clientCtx)

			// Load the Fleet
			performing_fleet_params := &types.QueryGetFleetRequest{
				Id: argFleetId,
			}

			performing_fleet_res, performing_fleet_err := queryClient.Fleet(context.Background(), performing_fleet_params)
			if performing_fleet_err != nil {
				return performing_fleet_err
			}

			var performingFleet types.Fleet
			performingFleet = performing_fleet_res.Fleet

			planet_params := &types.QueryGetPlanetRequest{
				Id: performingFleet.LocationId,
			}

			Planet_res, _ := queryClient.Planet(context.Background(), planet_params)
			planet := Planet_res.Planet
			planetAttributes := Planet_res.PlanetAttributes

			if planetAttributes.BlockStartRaid == 0 {
				fmt.Printf("Planet (%s) has no Active raid \n", performingFleet.LocationId)
				return nil
			}

			currentBlockResponse, _ := queryClient.GetBlockHeight(context.Background(), &types.QueryBlockHeight{})
			currentBlock := currentBlockResponse.BlockHeight
			fmt.Printf("Raid process against planet %s activated on %d, current block is %d \n", planet.Id, planetAttributes.BlockStartRaid, currentBlock)
			currentAge := currentBlock - planetAttributes.BlockStartRaid
			currentDifficulty := types.CalculateDifficulty(float64(currentAge), planetAttributes.PlanetaryShield) //structType.BuildDifficulty)
			fmt.Printf("Raid difficulty is %d \n", currentDifficulty)

			activeRaidBlockString := strconv.FormatUint(planetAttributes.BlockStartRaid, 10)
			fmt.Println("Starting Raiding...")

			var newDifficulty int
			var i int = 0
			for {
				if i > 0 { // the condition stops matching
					break // break out of the loop
				}

			COMPUTE:
				i = i + 1

				if (i % 20000) == 0 {
					currentBlockResponse, _ = queryClient.GetBlockHeight(context.Background(), &types.QueryBlockHeight{})
					currentBlock = currentBlockResponse.BlockHeight
					currentAge = currentBlock - planetAttributes.BlockStartRaid
					newDifficulty = types.CalculateDifficulty(float64(currentAge), planetAttributes.PlanetaryShield)

					if currentDifficulty != newDifficulty {
						currentDifficulty = newDifficulty
						fmt.Printf("Difficulty Change: %d \n", currentDifficulty)
					}

					if difficultyTargetStart > 0 {
						if difficultyTargetStart > currentDifficulty {
							time.Sleep(5 * time.Minute)
							goto COMPUTE
						}
					}

				}
				newHash := sha256.New()

				newInput := performingFleet.Id + "@" + planet.Id + "RAID" + activeRaidBlockString + "NONCE" + strconv.Itoa(i)
				newHash.Write([]byte(newInput))
				newHashOutput := hex.EncodeToString(newHash.Sum(nil))

				if valid, _ := types.HashBuildAndCheckDifficulty(newInput, newHashOutput, currentAge, planetAttributes.PlanetaryShield); !valid {
					goto COMPUTE
				}

				fmt.Println("")
				fmt.Println("Raid Complete!")
				fmt.Println(newInput)
				argNonce = strconv.Itoa(i)
				fmt.Println(newHashOutput)
				argProof = newHashOutput
			}

			msg := &types.MsgPlanetRaidComplete{
				Creator: clientCtx.GetFromAddress().String(),
				FleetId: argFleetId,
				Proof:   argProof,
				Nonce:   argNonce,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().IntP("difficulty_target_start", "D", 0, "Do not start the compute process until difficulty reaches this level (1-64)")

	return cmd
}
