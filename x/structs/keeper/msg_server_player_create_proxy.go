package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlayerCreateProxy(goCtx context.Context, msg *types.MsgPlayerCreateProxy) (*types.MsgPlayerCreateProxyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO Add a verification process to ensure that the proxy agent had the rights to do this
	// Basically, the player will need to provide some sort of signature that can then be verified here

	// Look up requesting account
	proxyPlayer := k.UpsertPlayer(ctx, msg.Creator)

	// look up destination guild
	guild, guildFound := k.GetGuild(ctx, proxyPlayer.GuildId)

    if (!guildFound) {
        // abort
    }

	// look up destination substation
	substation, substationFound := k.GetSubstation(ctx, guild.EntrySubstationId, true)

    if (!substationFound) {
        //abort
    }

	// create new player
    player := k.UpsertPlayer(ctx, msg.Address)

    // Add player to the guild
    player.SetGuild(guild.Id)

    // Connect player to the substation
    // Now let's get the player some power
    if (player.SubstationId == 0) {
        // Connect Player to Substation
        k.SubstationConnectPlayer(ctx, substation, player)
    }


	return &types.MsgPlayerCreateProxyResponse{}, nil
}
