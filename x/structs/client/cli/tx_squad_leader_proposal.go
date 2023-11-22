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

/**
*
    message MsgSquadCreate {
      string creator            = 1;
      uint64 guildId            = 2; -- could be the players guild by default
      uint64 leader             = 3; -- player by default
      uint64 squadJoinType      = 4; -- guild minimum by default
      uint64 entrySubstationId  = 5;
    }
*/

func CmdSquadLeaderProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "squad-leader-proposal [squad id] [leader]",
		Short: "Propose a new squad leader",
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
			msg := types.NewMsgSquadLeaderProposal(
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
