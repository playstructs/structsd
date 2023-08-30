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


func CmdStructRefineActivate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "struct-refine-activate [Struct ID]",
		Short: "Activate the refining process for a Struct",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			argStructId, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgStructRefineActivate(
				clientCtx.GetFromAddress().String(),
                argStructId,
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
