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
	"time"

    "crypto/sha256"
    "encoding/hex"
)

var _ = strconv.Itoa(0)


func CmdStructMineCompute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "struct-mine-compute [struct ID]",
		Short: "Do the work to extract an Ore from the planet",
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

            fmt.Printf("Loaded Struct (%d) for mining process \n", performingStructure.Id)

            if (performingStructure.MiningSystemStatus != "ACTIVE") {
                fmt.Printf("Struct (%d) has no Active mining system \n", performingStructure.Id)
                return nil
            }

            currentBlockResponse, _ := queryClient.GetBlockHeight(context.Background(), &types.QueryBlockHeight{})
            currentBlock := currentBlockResponse.BlockHeight
            fmt.Printf("Mining process activated on %d, current block is %d \n", performingStructure.ActiveMiningSystemBlock, currentBlock)
            currentAge := currentBlock - performingStructure.ActiveMiningSystemBlock
            currentDifficulty := types.CalculateActionDifficulty(float64(currentAge))
            fmt.Printf("Mining difficulty is %d \n", currentDifficulty)


            structIdString                  := strconv.FormatUint(performingStructure.Id, 10)
            activeMiningSystemBlockString   := strconv.FormatUint(performingStructure.ActiveMiningSystemBlock , 10)
            fmt.Println("Starting Mining...")

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
                    currentAge = currentBlock - performingStructure.ActiveMiningSystemBlock
                    newDifficulty = types.CalculateActionDifficulty(float64(currentAge))

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

                newInput := structIdString + "MINE" + activeMiningSystemBlockString + "NONCE" + strconv.Itoa(i)
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
    cmd.Flags().IntP("difficulty_target_start", "D", 0, "Do not start the compute process until difficulty reaches this level (1-64)")

	return cmd
}
