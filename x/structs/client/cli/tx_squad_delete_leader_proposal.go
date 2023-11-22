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

func CmdSquadDeleteLeaderProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "squad-delete-leader-proposal [Squad Id]",
		Short: "Delete a Squad Leader Proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
            argSquadId, err := cast.ToUint64E(args[0])
            if err != nil {
                return err
            }


			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSquadDeleteLeaderProposal(
				clientCtx.GetFromAddress().String(),
				argSquadId,
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
