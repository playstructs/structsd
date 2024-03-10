package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	//"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"structs/x/structs/types"

	"strings"
)

var _ = strconv.Itoa(0)

func CmdPermissionGrantOnAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permission-grant-on-address [address] [permission,permission2,...]",
		Short: "Grant permission on an Object to a Player",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			argAddress      := args[0]

			argPermissions   := args[1]
            var aggPermissions uint64

            splitPermissions := strings.Split(argPermissions, ",")
            for _, permission := range splitPermissions {
                aggPermissions = aggPermissions | uint64(types.Permission_enum[strings.ToLower(permission)])
            }

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgPermissionGrantOnAddress(
				clientCtx.GetFromAddress().String(),
				argAddress,
				aggPermissions,
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
