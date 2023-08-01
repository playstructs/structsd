package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"structs/x/structs/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
		Use:   "allocation-create [source-type] [source-id] [power] [controller]",
		Short: "Broadcast message allocation-create",
		Args:  cobra.RangeArgs(3, 4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {


            argSourceType := types.ObjectType_enum[args[0]]
            if (!types.IsValidAllocationConnectionType(argSourceType)) {
                return sdkerrors.Wrapf(types.ErrAllocationSourceType, "source type (%s) not valid for allocation", argSourceType.String() )
            }

			argSourceId, err := cast.ToUint64E(args[1])
			if err != nil {
				return err
			}

			argPower, err := cast.ToUint64E(args[2])
			if err != nil {
				return err
			}

            argController := args[3]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgAllocationCreate(
				clientCtx.GetFromAddress().String(),
				argController,
				argSourceType,
				argSourceId,
				argPower,
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
