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

func CmdAddressRegister() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "address-register [player-id] [address-type] [address] [proof]",
		Short: "Broadcast message address-register",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argPlayerId, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}
			argAddressType := args[1]
			argAddress := args[2]
			argProof := args[3]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgAddressRegister(
				clientCtx.GetFromAddress().String(),
				argPlayerId,
				argAddressType,
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
