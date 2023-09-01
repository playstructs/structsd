package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"structs/x/structs/types"

	//sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"context"
	"fmt"

    "crypto/sha256"
    "encoding/hex"
)

var _ = strconv.Itoa(0)


func CmdSabotage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sabotage [struct ID]",
		Short: "Sabotage a Struct",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			argStructId, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}

			var argProof string
			var argNonce string

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}


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

            fmt.Printf("Loaded Struct (%d) as target for sabotage \n", performingStructure.Id)


            currentBlockResponse, _ := queryClient.GetBlockHeight(context.Background(), &types.QueryBlockHeight{})
            currentBlock := currentBlockResponse.BlockHeight
            fmt.Printf("Sabotage process activated on %d, current block is %d \n", performingStructure.ActiveMiningSystemBlock, currentBlock)

            var currentAge uint64
            var currentDifficulty int
            var blockString string
            switch performingStructure.Type {
                case "Mining Rig":
                    currentAge         = currentBlock - performingStructure.ActiveMiningSystemBlock
                    currentDifficulty  = types.CalculateActionDifficultySabotage(float64(currentAge), types.DifficultySabotageRangeMine)
                    blockString        = strconv.FormatUint(performingStructure.ActiveMiningSystemBlock , 10)
                case "Refinery":
                    currentAge         = currentBlock - performingStructure.ActiveMiningSystemBlock
                    currentDifficulty  = types.CalculateActionDifficultySabotage(float64(currentAge), types.DifficultySabotageRangeRefine)
                    blockString        = strconv.FormatUint(performingStructure.ActiveRefiningSystemBlock , 10)
                case "Small Generator":
                    currentAge         = currentBlock - performingStructure.ActiveMiningSystemBlock
                    currentDifficulty  = types.CalculateActionDifficultySabotage(float64(currentAge), types.DifficultySabotageRangePower)
                    blockString        = strconv.FormatUint(performingStructure.BuildStartBlock , 10)
            }


            fmt.Printf("Sabotage difficulty is %d \n", currentDifficulty)
            structIdString := strconv.FormatUint(performingStructure.Id, 10)

            fmt.Println("Starting Mining...")

            var newDifficulty int
			var i int = 0
			for  {
				if i > 0  {      // the condition stops matching
                	break        // break out of the loop
        		}

COMPUTE:
				i = i + 1

                if (i % 20000) > 0 {
                    currentBlockResponse, _ = queryClient.GetBlockHeight(context.Background(), &types.QueryBlockHeight{})
                    currentBlock = currentBlockResponse.BlockHeight

                    switch performingStructure.Type {
                        case "Mining Rig":
                            currentAge         = currentBlock - performingStructure.ActiveMiningSystemBlock
                            currentDifficulty  = types.CalculateActionDifficultySabotage(float64(currentAge), types.DifficultySabotageRangeMine)
                        case "Refinery":
                            currentAge         = currentBlock - performingStructure.ActiveMiningSystemBlock
                            currentDifficulty  = types.CalculateActionDifficultySabotage(float64(currentAge), types.DifficultySabotageRangeRefine)
                        case "Small Generator":
                            currentAge         = currentBlock - performingStructure.ActiveMiningSystemBlock
                            currentDifficulty  = types.CalculateActionDifficultySabotage(float64(currentAge), types.DifficultySabotageRangePower)
                    }

                    if currentDifficulty != newDifficulty {
                        currentDifficulty = newDifficulty
                        fmt.Printf("Difficulty Change: %d \n", currentDifficulty)
                    }

                }

				newHash := sha256.New()

                newInput := structIdString + "SABOTAGE" + blockString + "NONCE" + strconv.Itoa(i)
				newHash.Write([]byte(newInput))
				newHashOutput := hex.EncodeToString(newHash.Sum(nil))

				if (!types.HashBuildAndCheckActionDifficulty(newInput, newHashOutput, currentAge)) { goto COMPUTE }

				fmt.Println("")
				fmt.Println("Mining Complete!")
				fmt.Println(newInput)
				argNonce = strconv.Itoa(i)
				fmt.Println(newHashOutput)
				argProof = newHashOutput
			}



			msg := types.NewMsgStructMine(
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

	return cmd
}