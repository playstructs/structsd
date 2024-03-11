package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"structs/x/structs/types"

	"strings"

)

var _ = strconv.Itoa(0)

/*

  string creator        = 1;
  string controller     = 2;
  objectType sourceType = 3;
  uint64 sourceId       = 4;
  uint64 power          = 5;

*/

func CmdAllocationCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "allocation-create [source-id] [power]",
		Short: "Broadcast message allocation-create",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {


			argSourceId := args[0]

			argPower, err := cast.ToUint64E(args[1])
			if err != nil {
				return err
			}

            argController, err := cmd.Flags().GetString("controller")
            if err != nil {
                return err
            }

            flagType, err := cmd.Flags().GetString("type")
            if err != nil {
                return err
            }
            argType := types.AllocationType_enum[strings.ToLower(flagType)]


			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgAllocationCreate(
				clientCtx.GetFromAddress().String(),
				argController,
				argSourceId,
				argPower,
				argType,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

    flags.AddTxFlagsToCmd(cmd)

    cmd.Flags().StringP("controller", "C", "", "The address of the allocation owner at creation")
    cmd.Flags().StringP("type", "T", "static", "static, dynamic, or automated")


	return cmd
}
