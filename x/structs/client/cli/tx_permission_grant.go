package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	//"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"structs/x/structs/types"
)

var _ = strconv.Itoa(0)

func CmdPermissionGrant() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permission-grant [objectId] [playerId] [permissions]",
		Short: "Grant permission on an Object to a Player",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			argObjectId      := args[0]

			argPlayerId      := args[1]

			argPermissions   := args[2]


			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgPermissionGrant(
				clientCtx.GetFromAddress().String(),
				argObjectId,
				argPlayerId,
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
