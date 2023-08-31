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


func CmdStructBuildCompute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "struct-build-compute [struct ID] [height estimate]",
		Short: "Do the work to finish a Struct build",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			argStructId, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}

			argHeightEst, err := cast.ToUint64E(args[1])
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

            fmt.Printf("Loaded Struct (%d) for building process \n", performingStructure.Id)

            if (performingStructure.Status != "BUILDING") {
                fmt.Printf("Struct (%d) is already built \n", performingStructure.Id)
                return nil
            }

            currentBlock := argHeightEst
            fmt.Printf("Build process activated on %d, current block is %d \n", performingStructure.BuildStartBlock, currentBlock)
            currentAge := currentBlock - performingStructure.BuildStartBlock
            currentDifficulty := types.CalculateActionDifficulty(float64(currentAge))
            fmt.Printf("Building difficult is %d \n", currentDifficulty)


            structIdString                  := strconv.FormatUint(performingStructure.Id, 10)
            buildStartBlockString           := strconv.FormatUint(performingStructure.BuildStartBlock , 10)
            fmt.Println("Starting Building...")

			var i int = 0
			for  {
				if i > 0  {      // the condition stops matching
                	break        // break out of the loop
        		}

COMPUTE:
				i = i + 1

				// dumb progress bar
				if (i % 3) > 0 {
					fmt.Print("\b")
				} else {
					fmt.Print("..")
				}

				newHash := sha256.New()

				/* hashInput                       := structIdString + "MINE" + activeMiningSystemBlockString + "NONCE" + msg.Nonce */

                newInput := structIdString + "BUILD" + buildStartBlockString + "NONCE" + strconv.Itoa(i)
				newHash.Write([]byte(newInput))
				newHashOutput := hex.EncodeToString(newHash.Sum(nil))


				if (!types.HashBuildAndCheckBuildDifficulty(newInput, newHashOutput, currentAge)) { goto COMPUTE }


				fmt.Println("")
				fmt.Println("Building Complete!")
				fmt.Println(newInput)
				argNonce = strconv.Itoa(i)
				fmt.Println(newHashOutput)
				argProof = newHashOutput
			}



			msg := types.NewMsgStructBuildComplete(
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
