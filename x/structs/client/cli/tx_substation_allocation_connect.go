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

func CmdSubstationAllocationConnect() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "substation-allocation-connect [allocation-id] [destination-substation-id]",
		Short: "Broadcast message substation-allocation-connect",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argAllocationId, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}

			argDestinationSubstationId, err := cast.ToUint64E(args[1])
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSubstationAllocationConnect(
				clientCtx.GetFromAddress().String(),
				argAllocationId,
				argDestinationSubstationId,
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