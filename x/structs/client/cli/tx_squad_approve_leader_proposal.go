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

func CmdSquadApproveLeaderProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "squad-approve-leader-proposal [Squad Id] [Player Id] [Decision]",
		Short: "Approve of Deny a Squad Leader Prosal",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
            argSquadId, err := cast.ToUint64E(args[0])
            if err != nil {
                return err
            }

			argPlayerId, err := cast.ToUint64E(args[1])
			if err != nil {
				return err
			}

			argApproved, err := strconv.ParseBool(args[2])
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSquadApproveLeaderProposal(
				clientCtx.GetFromAddress().String(),
				argSquadId,
				argPlayerId,
				argApproved,
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
