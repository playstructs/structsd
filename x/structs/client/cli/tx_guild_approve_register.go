package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/spf13/cobra"
	"structs/x/structs/types"
)

var _ = strconv.Itoa(0)

func CmdGuildApproveRegister() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guild-approve-register [guildId] [playerId] [decision]",
		Short: "Broadcast message guild-approve-register",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

            argGuildId := args[0]

			argPlayerId := args[1]


			argApproved, err := strconv.ParseBool(args[2])
			if err != nil {
				return err
			}



			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgGuildApproveRegister(
				clientCtx.GetFromAddress().String(),
				argApproved,
				argGuildId,
				argPlayerId,
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
