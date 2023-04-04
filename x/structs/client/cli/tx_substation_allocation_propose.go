package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"structs/x/structs/types"
)

var _ = strconv.Itoa(0)

func CmdSubstationAllocationPropose() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "substation-allocation-propose [id] [source-type] [source-id] [power]",
		Short: "Broadcast message substation-allocation-propose",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argId, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}
			argSourceType := args[1]
			argSourceId, err := cast.ToUint64E(args[2])
			if err != nil {
				return err
			}
			argPower := args[3]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSubstationAllocationPropose(
				clientCtx.GetFromAddress().String(),
				argId,
				argSourceType,
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
