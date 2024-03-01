package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlayerCreate(goCtx context.Context, msg *types.MsgPlayerCreate) (*types.MsgPlayerCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Look up requesting account
	player := k.UpsertPlayer(ctx, msg.Creator, true)

	// look up destination guild
	guild, guildFound := k.GetGuild(ctx, msg.GuildId)

    if (!guildFound) {
        // abort
    }

	// look up destination substation
	substation, substationFound := k.GetSubstation(ctx, guild.EntrySubstationId, true)

    if (guild.JoinInfusionMinimum > 0) {

        // Check to see the delegation of the player
        if (false) {

        } else if (guild.JoinInfusionMinimumBypassByRequest != types.GuildJoinBypassLevel_closed) {
            k.GuildSetRegisterRequest(ctx, guild, player)
            return &types.MsgPlayerCreateResponse{}, nil
        } else {
            return &types.MsgPlayerCreateResponse{}, nil
            // return error
                // does not meet the delegation minimums and requests are closed
        }

    } else {
        // If the player is already connected to a substation then leave them
        // Maybe add an option to force migration later
        if (player.SubstationId == "") {
            if (!substationFound) {
                // TODO Throw Error : No entry substation found for public guild
                return &types.MsgPlayerCreateResponse{}, nil
            }

            player.GuildId = guild.Id
            k.SubstationConnectPlayer(ctx, substation, player)
        }
    }

	return &types.MsgPlayerCreateResponse{}, nil
}
