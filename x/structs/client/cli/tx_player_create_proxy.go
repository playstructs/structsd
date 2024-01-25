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

func CmdPlayerCreateProxy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "player-create-proxy [Address] [Substation ID]",
		Short: "Broadcast message player-create-proxy",
		Args:  cobra.RangeArgs(1,2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
		    argAddress := args[0]

            argSubstationOverride := false
            argSubstationId, err := cast.ToUint64E(args[1])
            if err != nil {
                return err
            }
            if (argSubstationId > 0) {
                argSubstationOverride = true
            }

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgPlayerCreateProxy(
				clientCtx.GetFromAddress().String(),
				argAddress,
				argSubstationOverride,
				argSubstationId,
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
