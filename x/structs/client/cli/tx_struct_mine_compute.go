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


func CmdStructMineCompute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "struct-ore-mine-compute [struct ID]",
		Short: "Do the work to extract an Ore from the planet",
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

            fmt.Printf("Loaded Struct (%s) for mining process \n", performingStructure.Id)


            struct_attribute_block_start_params := &types.QueryGetStructAttributeRequest{
                StructId: argStructId,
                AttributeType: "blockStartOreMine",
            }

            mineStartBlock_res, _ := queryClient.StructAttribute(context.Background(), struct_attribute_block_start_params)
            mineStartBlock := mineStartBlock_res.Attribute

            if (mineStartBlock == 0) {
                fmt.Printf("Struct (%s) has no Active mining system \n", performingStructure.Id)
                return nil
            }


            struct_type_params := &types.QueryGetStructTypeRequest{
                Id: performingStructure.Type,
            }

            structType_res, _ := queryClient.StructType(context.Background(), struct_type_params)
            structType := structType_res.StructType


            currentBlockResponse, _ := queryClient.GetBlockHeight(context.Background(), &types.QueryBlockHeight{})
            currentBlock := currentBlockResponse.BlockHeight
            fmt.Printf("Mining process activated on %d, current block is %d \n", mineStartBlock, currentBlock)
            currentAge := currentBlock - mineStartBlock
            currentDifficulty := types.CalculateDifficulty(float64(currentAge), structType.BuildDifficulty)
            fmt.Printf("Mining difficulty is %d \n", currentDifficulty)


            activeMiningSystemBlockString   := strconv.FormatUint(mineStartBlock , 10)
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
                    currentAge = currentBlock - mineStartBlock
                    newDifficulty = types.CalculateDifficulty(float64(currentAge), structType.OreMiningDifficulty)

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

                newInput := performingStructure.Id + "MINE" + activeMiningSystemBlockString + "NONCE" + strconv.Itoa(i)
				newHash.Write([]byte(newInput))
				newHashOutput := hex.EncodeToString(newHash.Sum(nil))

				if (!types.HashBuildAndCheckDifficulty(newInput, newHashOutput, currentAge, structType.OreMiningDifficulty)) { goto COMPUTE }

				fmt.Println("")
				fmt.Println("Mining Complete!")
				fmt.Println(newInput)
				argNonce = strconv.Itoa(i)
				fmt.Println(newHashOutput)
				argProof = newHashOutput
			}



			msg := &types.MsgStructOreMinerComplete{
                Creator:  clientCtx.GetFromAddress().String(),
                StructId: argStructId,
                Proof: argProof,
                Nonce: argNonce,
            }


			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
    cmd.Flags().IntP("difficulty_target_start", "D", 0, "Do not start the compute process until difficulty reaches this level (1-64)")

	return cmd
}
