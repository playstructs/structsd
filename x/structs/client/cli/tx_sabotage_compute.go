package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/spf13/cobra"
	"structs/x/structs/types"

	//sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"context"
	"fmt"
	"time"

    "crypto/sha256"
    "encoding/hex"
)

var _ = strconv.Itoa(0)


func CmdSabotageCompute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sabotage-compute [struct ID]",
		Short: "Sabotage a Struct (with computation)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			argStructId := args[0]

			var argProof string
			var argNonce string

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

            difficultyTargetStart, _ := cmd.Flags().GetInt("difficulty_target_start")

			queryClient := types.NewQueryClient(clientCtx)


			// Load the Struct
			performing_structure_params := &types.QueryGetStructRequest{
				Id: argStructId,
			}


			performing_structure_res, performing_structure_err := queryClient.Struct(context.Background(), performing_structure_params)
			if performing_structure_err != nil {
				return performing_structure_err
			}

			var performingStructure types.Struct
			performingStructure = performing_structure_res.Struct

            fmt.Printf("Loaded Struct (%s) as target for sabotage \n", performingStructure.Id)


            currentBlockResponse, _ := queryClient.GetBlockHeight(context.Background(), &types.QueryBlockHeight{})
            currentBlock := currentBlockResponse.BlockHeight
            fmt.Printf("Sabotage process activated on %d, current block is %d \n", performingStructure.Id, currentBlock)

            var currentAge uint64
            var currentDifficulty int
            var blockString string
            switch performingStructure.Type {
                case "Mining Rig":
                    currentAge         = currentBlock - performingStructure.ActiveMiningSystemBlock
                    currentDifficulty  = types.CalculateActionDifficultySabotage(float64(currentAge), types.DifficultySabotageRangeMine)
                    blockString        = strconv.FormatUint(performingStructure.ActiveMiningSystemBlock , 10)
                case "Refinery":
                    currentAge         = currentBlock - performingStructure.ActiveRefiningSystemBlock
                    currentDifficulty  = types.CalculateActionDifficultySabotage(float64(currentAge), types.DifficultySabotageRangeRefine)
                    blockString        = strconv.FormatUint(performingStructure.ActiveRefiningSystemBlock , 10)
                case "Small Generator":
                    currentAge         = currentBlock - performingStructure.BuildStartBlock
                    currentDifficulty  = types.CalculateActionDifficultySabotage(float64(currentAge), types.DifficultySabotageRangePower)
                    blockString        = strconv.FormatUint(performingStructure.BuildStartBlock , 10)
            }


            fmt.Printf("Sabotage difficulty is %d \n", currentDifficulty)

            fmt.Println("Starting sabotage mission...")

            var newDifficulty int
			var i int = 0
			for  {
				if i > 0  {      // the condition stops matching
                	break        // break out of the loop
        		}

COMPUTE:
				i = i + 1

                if (i % 20000) == 0 {
                    currentBlockResponse, _ = queryClient.GetBlockHeight(context.Background(), &types.QueryBlockHeight{})
                    currentBlock = currentBlockResponse.BlockHeight

                    switch performingStructure.Type {
                        case "Mining Rig":
                            currentAge         = currentBlock - performingStructure.ActiveMiningSystemBlock
                            currentDifficulty  = types.CalculateActionDifficultySabotage(float64(currentAge), types.DifficultySabotageRangeMine)
                        case "Refinery":
                            currentAge         = currentBlock - performingStructure.ActiveRefiningSystemBlock
                            currentDifficulty  = types.CalculateActionDifficultySabotage(float64(currentAge), types.DifficultySabotageRangeRefine)
                        case "Small Generator":
                            currentAge         = currentBlock - performingStructure.BuildStartBlock
                            currentDifficulty  = types.CalculateActionDifficultySabotage(float64(currentAge), types.DifficultySabotageRangePower)
                    }

                    if currentDifficulty != newDifficulty {
                        currentDifficulty = newDifficulty
                        fmt.Printf("Difficulty Change: %d \n", currentDifficulty)
                    }

                    if (difficultyTargetStart > 0 ) {
                        if (difficultyTargetStart > currentDifficulty) {
                            time.Sleep(5 * time.Minute)
                            goto COMPUTE
                        }
                    }

                }

				newHash := sha256.New()

                newInput := performingStructure.Id + "SABOTAGE" + blockString + "NONCE" + strconv.Itoa(i)
				newHash.Write([]byte(newInput))
				newHashOutput := hex.EncodeToString(newHash.Sum(nil))

				if (!types.HashBuildAndCheckActionDifficulty(newInput, newHashOutput, currentAge)) { goto COMPUTE }

				fmt.Println("")
				fmt.Println("Saboteur!")
				fmt.Println(newInput)
				argNonce = strconv.Itoa(i)
				fmt.Println(newHashOutput)
				argProof = newHashOutput
			}



			msg := types.NewMsgSabotage(
				clientCtx.GetFromAddress().String(),
                argStructId,
                argProof,
                argNonce,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().IntP("difficulty_target_start", "D", 0, "Do not start the compute process until difficulty reaches this level (1-64)")

	return cmd
}
