package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/spf13/cobra"
	"structs/x/structs/types"

)

var _ = strconv.Itoa(0)

func CmdPlayerCreateProxy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "player-create-proxy [Address]",
		Short: "Broadcast message player-create-proxy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
		    argAddress := args[0]

            argSubstationId, err := cmd.Flags().GetUint64("substation-id")
            if err != nil {
                return err
            }


			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgPlayerCreateProxy(
				clientCtx.GetFromAddress().String(),
				argAddress,
				argSubstationId,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().Uint64P("substation-id", "S", 0, "Override the Guild Substation with this Substation ID during proxy player creation")

	return cmd
}
