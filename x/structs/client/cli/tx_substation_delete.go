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

func CmdSubstationDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "substation-delete [substation-id] [migration-substation-id]",
		Short: "Broadcast message substation-delete",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argSubstationId, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}

			argMigrationSubstationId, _ := cast.ToUint64E(args[1])

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSubstationDelete(
				clientCtx.GetFromAddress().String(),
				argSubstationId,
				argMigrationSubstationId,
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
