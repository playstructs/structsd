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

/**
*
    message MsgSquadCreate {
      string creator            = 1;
      uint64 guildId            = 2; -- could be the players guild by default
      uint64 leader             = 3; -- player by default
      uint64 squadJoinType      = 4; -- guild minimum by default
      uint64 entrySubstationId  = 5;
    }
*/

func CmdSquadCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "squad-create [entry substation id]",
		Short: "Broadcast message squad-create",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			argEntrySubstationId, err := cast.ToUint64E(args[0])
            if err != nil {
                return err
            }

            /*
             * Load the Player object of the calling address
             * Followed by Guild object from the Player object
             *
             * Used for defaults of Guild ID, Leader
             */
            addressLookupRequest := &types.QueryGetAddressRequest{
                Address: clientCtx.GetFromAddress().String(),
            }

            addressResults, AddressLookupErr := queryClient.Address(context.Background(), addressLookupRequest)
            if AddressLookupErr != nil {
                return AddressLookupErr
            }

            playerLookupRequest = &types.QueryGetPlayerRequest{
                Id: addressResults.PlayerId,
            }

            playerResults, playerLookupErr := queryClient.Player(context.Background(), playerLookupRequest)
            if playerLookupErr != nil {
                return playerLookupErr
            }

            guildLookupRequest = &types.QueryGetGuildRequest{
                Id: playerResults.GuildId,
            }

            guildResults, guildLookupErr := queryClient.Guild(context.Background(), guildLookupRequest)
            if guildLookupErr != nil {
                return guildLookupErr
            }



            /* Set the remaining Parameters */
		    argGuildId, err := cmd.Flags().GetUint64("guild-id")
            if err != nil {
                return err
            }

            if (argGuildId == 0) {
                argGuildId = playerResults.GuildId
            }

            argLeader, err := cmd.Flags().GetUint64("leader")
            if err != nil {
                return err
            }

            if (argLeader == 0) {
                argLeader = playerResults.Id
            }

            argSquadJoinType, err := cmd.Flags().GetUint64("join-type")
            if err != nil {
                return err
            }

            if (argSquadJoinType == types.SquadJoinType_Invalid) {
                argSquadJoinType = guildResults.SquadJoinTypeMinimum
            }

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSquadCreate(
				clientCtx.GetFromAddress().String(),
				argGuildId,
				argLeader,
				argSquadJoinType,
				argEntrySubstationId,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

    cmd.Flags().Uint64P("guild-id", "G", 0, "Set the Guild ID of the new Squad")
    cmd.Flags().Uint64P("leader", "L", 0, "Set the Leader of the Squad")
    cmd.Flags().Uint64P("join-type", "J", types.SquadJoinType_Invalid, "Set the Join Type of the new Squad")

	return cmd
}
