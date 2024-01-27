package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"structs/x/structs/types"
	"github.com/spf13/cast"
)

var _ = strconv.Itoa(0)

func CmdGuildUpdateJoinInfusionMinimum() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guild-update-join-infusion-minimum [guild id] [infusion join minimum]",
		Short: "Update the Join Infusion Minimum of a Guild",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argGuildId, err := cast.ToUint64E(args[0])
            if err != nil {
                return err
            }

			argJoinInfusionMinimum, err := cast.ToUint64E(args[1])
            if err != nil {
                return err
            }

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgGuildUpdateJoinInfusionMinimum(
				clientCtx.GetFromAddress().String(),
				argGuildId,
				argJoinInfusionMinimum,
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
