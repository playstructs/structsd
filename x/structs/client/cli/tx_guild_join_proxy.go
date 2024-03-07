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

func CmdGuildJoinProxy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guild-join-proxy [Address]",
		Short: "Join a player to the guild via a proxy player",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
		    argAddress := args[0]

            argSubstationId, err := cmd.Flags().GetString("substation-id")
            if err != nil {
                return err
            }


			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgGuildJoinProxy(
				clientCtx.GetFromAddress().String(),
				argAddress,
				argSubstationId,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().StringP("substation-id", "S", "", "Override the Guild Substation with this Substation ID during proxy player creation")

	return cmd
}
