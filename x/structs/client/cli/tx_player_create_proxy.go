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

func CmdPlayerCreateProxy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "player-create-proxy [guild-id] [substation-id] [address] [proof]",
		Short: "Broadcast message player-create-proxy",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argGuildId, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}
			argSubstationId, err := cast.ToUint64E(args[1])
			if err != nil {
				return err
			}
			argAddress := args[2]
			argProof := args[3]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgPlayerCreateProxy(
				clientCtx.GetFromAddress().String(),
				argGuildId,
				argSubstationId,
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
