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

func CmdPlayerCreateProxy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "player-create-proxy [address] [proof]",
		Short: "Broadcast message player-create-proxy",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argAddress := args[0]
			argProof := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgPlayerCreateProxy(
				clientCtx.GetFromAddress().String(),
				argAddress,
				argProof,
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
