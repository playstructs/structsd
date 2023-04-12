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

func CmdSubstationAllocationActivate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "substation-allocation-activate [allocation-id] [decision]",
		Short: "Broadcast message substation-allocation-activate",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argAllocationId, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}

            argDecision, err := cast.ToBoolE(args[1])
            if err != nil {
                return err
            }

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSubstationAllocationActivate(
				clientCtx.GetFromAddress().String(),
				argAllocationId,
				argDecision,
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
