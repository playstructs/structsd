package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/spf13/cobra"
	"structs/x/structs/types"

	"strings"
)

var _ = strconv.Itoa(0)

func CmdSubstationPlayerMigrate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "substation-player-migrate [substation-id] [player-id,player-id2,...]",
		Short: "Migrate a list of players from one substation to another",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argSubstationId := args[0]

			argPlayerIds := args[1]

            var aggPlayerIds []string

            splitPlayerIds := strings.Split(argPlayerIds, ",")
            for _, playerId := range splitPlayerIds {
                aggPlayerIds = append(aggPlayerIds, playerId)
            }

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSubstationPlayerMigrate(
				clientCtx.GetFromAddress().String(),
				argSubstationId,
				aggPlayerIds,
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
