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


func CmdStructRefineCompute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "struct-refine-compute [struct ID] [Height Estimate]",
		Short: "Do the work to convert Alpha Ore into Alpha Matter",
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

            fmt.Printf("Loaded Struct (%d) for mining process \n", performingStructure.Id)

            if (performingStructure.RefiningSystemStatus != "ACTIVE") {
                fmt.Printf("Struct (%d) has no Active refining system \n", performingStructure.Id)
                return nil
            }

            currentBlock := argHeightEst
            fmt.Printf("Refining process activated on %d, current block is %d \n", performingStructure.ActiveRefiningSystemBlock, currentBlock)
            currentAge := currentBlock - performingStructure.ActiveMiningSystemBlock
            currentDifficulty := types.CalculateActionDifficulty(float64(currentAge))
            fmt.Printf("Refining difficult is %d \n", currentDifficulty)


            structIdString                  := strconv.FormatUint(performingStructure.Id, 10)
            activeRefiningSystemBlockString := strconv.FormatUint(performingStructure.ActiveRefiningSystemBlock , 10)
            fmt.Println("Starting Refining...")

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

                newInput := structIdString + "REFINE" + activeRefiningSystemBlockString + "NONCE" + strconv.Itoa(i)
				newHash.Write([]byte(newInput))
				newHashOutput := hex.EncodeToString(newHash.Sum(nil))


				if (!types.HashBuildAndCheckActionDifficulty(newInput, newHashOutput, currentAge)) { goto COMPUTE }


				fmt.Println("")
				fmt.Println("Refining (hopefully) Complete!")
				fmt.Println(newInput)
				argNonce = strconv.Itoa(i)
				fmt.Println(newHashOutput)
				argProof = newHashOutput
			}


			msg := types.NewMsgStructRefine(
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
