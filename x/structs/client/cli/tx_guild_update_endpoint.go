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

func CmdGuildUpdateEndpoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guild-update-endpoint [guild id] [endpoint]",
		Short: "Update the Endpoint of a Guild",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argGuildId, err := cast.ToUint64E(args[0])
            if err != nil {
                return err
            }

			argEndpoint := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgGuildUpdateEndpoint(
				clientCtx.GetFromAddress().String(),
				argGuildId,
				argEndpoint,
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
