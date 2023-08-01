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

func CmdReactorAllocationCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reactor-allocation-create [source-id] [power] [controller]",
		Short: "Broadcast message reactor-allocation-create",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			argSourceId, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}

			argPower, err := cast.ToUint64E(args[1])
			if err != nil {
				return err
			}

            argController := args[2]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgReactorAllocationCreate(
				clientCtx.GetFromAddress().String(),
				argController,
				argSourceId,
				argPower,
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
