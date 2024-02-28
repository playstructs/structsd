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

    if (!substationFound) {
        // Error substation not found for guild
    }

    if (!substation.HasPlayerCapacity()) {
        // error substation does not have capacity for another player to join
    }


    if (guild.JoinInfusionMinimum > 0) {

    // check the guild infusion minimum
        // does the player not meet it?
            // if BypassByRequest is allowed
                k.GuildSetRegisterRequest(ctx, guild, player)
            // else
                // error, doesn't meet infusion minimum. No BypassRequest allowed
        // else
            // add the user to the guild


    }




    switch guild.GuildJoinType {

        case types.GuildJoinType_Open:
            // If the player is already connected to a substation then leave them
            // Maybe add an option to force migration later
            if (player.SubstationId == "") {
                if (substationFound) {
                    // Check if the substation has room
                    if substation.HasPlayerCapacity() {
                        // Connect Player to Substation
                        k.SubstationConnectPlayer(ctx, substation, player)
                    } else {
                        // error that the substation cannot accept a new player
                        }
                } else {
                    // TODO Throw Error : No entry substation found for public guild
                    return &types.MsgPlayerCreateResponse{}, nil
                }
            }

            // Add player to the guild
            player.SetGuild(guild.Id)
            k.SetPlayer(ctx, player)

        case types.GuildJoinType_InfusionMinimum:
            // TODO Throw error : join via delegation
            return &types.MsgPlayerCreateResponse{}, nil

        case types.GuildJoinType_Request:
            k.GuildSetRegisterRequest(ctx, guild, player)

        case types.GuildJoinType_Invite:
            // TODO Throw error : Join via invite only
            return &types.MsgPlayerCreateResponse{}, nil

        default:
            // TODO Throw error : Guild config error
            // What type of join rule is even set if we got to here?
            return &types.MsgPlayerCreateResponse{}, nil
    }



	return &types.MsgPlayerCreateResponse{}, nil
}
