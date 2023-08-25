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

func CmdAddressApproveRegister() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "address-approve-register [address] [decision] [permissions]",
		Short: "Broadcast message address-approve-register",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argApproved, err := strconv.ParseBool(args[0])
			if err != nil {
				return err
			}
			argAddress      := args[1]
			argPermissions, err := cast.ToUint64E(args[2])
            if err != nil {
                return err
            }

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgAddressApproveRegister(
				clientCtx.GetFromAddress().String(),
				argApproved,
				argAddress,
				argPermissions,
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
