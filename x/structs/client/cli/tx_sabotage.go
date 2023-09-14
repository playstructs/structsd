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

)

var _ = strconv.Itoa(0)


func CmdSabotage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sabotage [struct ID] [proof] [nonce]",
		Short: "Sabotage a Struct",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			argStructId, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}

		    argProof := args[1]
			argNonce := args[2]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

            // TODO could verify hash proof before sending

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

	return cmd
}
