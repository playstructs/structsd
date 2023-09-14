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




func CmdGuildUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guild-update [guild id] [owner id] [endpoint] [entry substation id] [guild join type] [infusion join minimum]",
		Short: "Broadcast message guild-update",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argGuildId, err := cast.ToUint64E(args[0])
            if err != nil {
                return err
            }

			argOwnerId, err := cast.ToUint64E(args[1])
            if err != nil {
                return err
            }

			argEndpoint := args[2]

			argEntrySubstationId, err := cast.ToUint64E(args[3])
            if err != nil {
                return err
            }

			argGuildJoinType, err := cast.ToUint64E(args[4])
            if err != nil {
                return err
            }

			argInfusionJoinMinimum, err := cast.ToUint64E(args[5])
            if err != nil {
                return err
            }

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgGuildUpdate(
				clientCtx.GetFromAddress().String(),
				argGuildId,
				argEndpoint,
				argEntrySubstationId,
				argOwnerId,
				argGuildJoinType,
				argInfusionJoinMinimum,
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
