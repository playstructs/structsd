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

func CmdGuildUpdateJoinType() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guild-update-join-type [guild id] [join type]",
		Short: "Update the Join Type of a Guild",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argGuildId, err := cast.ToUint64E(args[0])
            if err != nil {
                return err
            }

			argJoinType, err := cast.ToUint64E(args[1])
            if err != nil {
                return err
            }

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgGuildUpdateJoinType(
				clientCtx.GetFromAddress().String(),
				argGuildId,
				argJoinType,
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
