package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/spf13/cast"
	"structs/x/structs/types"
)

var _ = strconv.Itoa(0)

func CmdSubstationCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "substation-create [player-connection-allocation] [owner]",
		Short: "Broadcast message substation-create",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {


			argPlayerConnectionAllocation, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}


            var argOwner uint64
            if (len(args) > 1) {
                argOwner, err = cast.ToUint64E(args[1])
                if err != nil {
                    return err
                }

            }

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSubstationCreate(
				clientCtx.GetFromAddress().String(),
				argOwner,
				argPlayerConnectionAllocation,
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
