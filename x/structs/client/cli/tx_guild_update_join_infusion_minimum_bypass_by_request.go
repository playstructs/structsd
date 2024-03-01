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

func CmdGuildUpdateJoinInfusionMinimumBypassByRequest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guild-update-join-infusion-minimum-bypass-by-request [guild id] [Bypass Level]",
		Short: "Update how Requests are allowed to bypass the Infusion Minimum",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argGuildId := args[0]

			argGuildJoinBypassLevel, err := cast.ToUint64E(args[1])
            if err != nil {
                return err
            }

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgGuildUpdateJoinInfusionMinimumBypassByRequest(
				clientCtx.GetFromAddress().String(),
				argGuildId,
				types.GuildJoinBypassLevel(argGuildJoinBypassLevel),
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
