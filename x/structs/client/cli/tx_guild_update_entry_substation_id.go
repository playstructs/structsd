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

func CmdGuildUpdateEntrySubstationId() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guild-update-entry-substation-id [guild id] [entry substation id]",
		Short: "Update the Entry Substation ID of a Guild",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argGuildId := args[0]

            argEntrySubstationId := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgGuildUpdateEntrySubstationId(
				clientCtx.GetFromAddress().String(),
				argGuildId,
				argEntrySubstationId,
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
