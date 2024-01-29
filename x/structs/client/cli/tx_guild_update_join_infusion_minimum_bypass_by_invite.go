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

func CmdGuildUpdateJoinInfusionMinimumBypassByInvite() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guild-update-join-infusion-minimum-bypass-by-invite [guild id] [Bypass Level]",
		Short: "Update how Invitations are allowed to bypass the Infusion Minimum",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argGuildId, err := cast.ToUint64E(args[0])
            if err != nil {
                return err
            }

			argGuildJoinBypassLevel, err := cast.ToUint64E(args[1])
            if err != nil {
                return err
            }

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgGuildUpdateJoinInfusionMinimumBypassByInvite(
				clientCtx.GetFromAddress().String(),
				argGuildId,
				argGuildJoinBypassLevel,
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
