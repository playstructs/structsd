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

func CmdGuildUpdate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guild-update [guild id]",
		Short: "Broadcast message guild-update",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argGuildId, err := cast.ToUint64E(args[0])
            if err != nil {
                return err
            }

			argOwnerId, err := cmd.Flags().GetUint64("owner-id")
            if err != nil {
                return err
            }

			argEndpoint, err := cmd.Flags().GetString("endpoint")
            if err != nil {
                return err
            }

			argEntrySubstationId, err := cmd.Flags().GetUint64("entry-substation-id")
            if err != nil {
                return err
            }

			argGuildJoinType, err := cmd.Flags().GetUint64("guild-join-type")
            if err != nil {
                return err
            }

			argInfusionJoinMinimum, err := cmd.Flags().GetUint64("infusion-join-minimum")
            if err != nil {
                return err
            }

            // If no parameters have been provided than no reason to continue
            // TODO: Return an error

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgGuildUpdate(
				clientCtx.GetFromAddress().String(),
				argGuildId,
				argEndpoint,
				argEntrySubstationId,
				argOwnerId,
				argGuildJoinType,
				argInfusionJoinMinimum,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	cmd.Flags().Uint64P("owner-id", "O", 0, "Change the Owner of the Guild")
	cmd.Flags().Uint64P("entry-substation-id", "S", 0, "Change the Substation new Guild members automatically join")
	cmd.Flags().Uint64P("guild-join-type", "T", 0, "Change how new Players are able to join the Guild")
    cmd.Flags().Uint64P("infusion-join-minimum", "M", 0, "Change the minimum about of Alpha that must be infused in the reactor for a player to join")

    cmd.Flags().StringP("endpoint", "E", "", "Change the external Guild API endpoint")


	return cmd
}
