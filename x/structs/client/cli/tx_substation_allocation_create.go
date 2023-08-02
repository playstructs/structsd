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

func CmdSubstationAllocationCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "substation-allocation-create [source-id] [power] [controller]",
		Short: "Broadcast message substation-allocation-create",
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

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var argController string
			if len(args) > 2 {
				argController = args[2]
			} else {
				argController = clientCtx.GetFromAddress().String()
			}

			msg := types.NewMsgSubstationAllocationCreate(
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
