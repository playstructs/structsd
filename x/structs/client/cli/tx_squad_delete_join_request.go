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


func CmdSquadDeleteJoinRequest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "squad-delete-join-request [squad id] [player id]",
		Short: "Delete a squad join request",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

            /* Setup the Context Objects */
            clientCtx, err := client.GetClientTxContext(cmd)
            if err != nil {
                return err
            }

            queryClient := types.NewQueryClient(clientCtx)


            /* Parse the arguments */
			argSquadId, err := cast.ToUint64E(args[0])
            if err != nil {
                return err
            }

            /* Parse the arguments */
			argPlayerId, err := cast.ToUint64E(args[1])
            if err != nil {
                return err
            }

            /* Build, Validate, Broadcast */
			msg := types.NewMsgSquadDeleteJoinRequest(
				clientCtx.GetFromAddress().String(),
				argSquadId,
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
