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

func CmdPermissionRevokeOnObject() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permission-revoke-on-object [objectId] [playerId] [permission,permission2,...]",
		Short: "Revoke permission on an Object from a Player",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			argObjectId      := args[0]

			argPlayerId      := args[1]

			argPermissions   := args[2]
            var aggPermissions uint64

            splitPermissions := strings.Split(argPermissions, ",")
            for _, permission := range splitPermissions {
                aggPermissions = aggPermissions | uint64(types.Permission_enum[strings.ToLower(permission)])
            }

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgPermissionRevokeOnObject(
				clientCtx.GetFromAddress().String(),
				argObjectId,
				argPlayerId,
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
