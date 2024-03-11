package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildJoinProxy(goCtx context.Context, msg *types.MsgGuildJoinProxy) (*types.MsgGuildJoinProxyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO Add a verification process to ensure that the proxy agent had the rights to do this
	// Basically, the player will need to provide some sort of signature that can then be verified here

	// Look up requesting account
	proxyPlayer := k.UpsertPlayer(ctx, msg.Creator, true)

	// look up destination guild
	guild, guildFound := k.GetGuild(ctx, proxyPlayer.GuildId)

    if (!guildFound) {
        return &types.MsgGuildJoinProxyResponse{}, sdkerrors.Wrapf(types.ErrGuildNotFound, "Referenced Guild (%d) not found", guild.Id)
    }

    guildObjectPermissionId := GetObjectPermissionIDBytes(guild.Id, proxyPlayer.Id)
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)

    // Check to make sure the player has permissions on the guild
    if (!k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.PermissionAssociations)) {
        return &types.MsgGuildJoinProxyResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%d) has no Player Registration permissions ", proxyPlayer.Id)
    }

    // Make sure the address calling this has Associate permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssociations)) {
        return &types.MsgGuildJoinProxyResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageGuild, "Calling address (%s) has no Guild Management permissions ", msg.Creator)
    }


    var substation types.Substation
    substationFound := false

	/* Look up destination substation
	 *
	 * We're going to try and load up the substation override first
	 * and if that doesn't exist, we'll go load up the regular
	 * guild entry substation.
	 *
	 * Proxy player needs permissions on the override but the default
	 * entry substation will always work.
	 */

	if (msg.SubstationId != "") {
	    substation, substationFound = k.GetSubstation(ctx, msg.SubstationId, true)
        if (!substationFound) {
            return &types.MsgGuildJoinProxyResponse{}, sdkerrors.Wrapf(types.ErrSubstationNotFound, "Provided Substation Override (%s) not found", msg.SubstationId)
        }

        // Since the Guild Entry Substation is being overridden, let's make
        // sure the ProxyPlayer actually have authority over this substation
        substationObjectPermissionId := GetObjectPermissionIDBytes(substation.Id, proxyPlayer.Id)
        if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {
            return &types.MsgGuildJoinProxyResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%d) has no Player Connect permissions on Substation (%s) used as override", proxyPlayer.Id, substation.Id)
        }
	}

    if (!substationFound) {
        substation, substationFound = k.GetSubstation(ctx, guild.EntrySubstationId, true)
        if (!substationFound) {
            return &types.MsgGuildJoinProxyResponse{}, sdkerrors.Wrapf(types.ErrSubstationNotFound, "Entry Substation (%s) for Guild (%s) not found", guild.EntrySubstationId, guild.Id)
        }
    }

	// create new player
    player := k.UpsertPlayer(ctx, msg.Address, true)
    // this doesn't make a lot of sense. Shouldn't really be creating users here since this is a create.
    // Maybe this needs to changed to guild_join_proxy and guild_join


    // Join Guild via Proxy only Works if not in a guild
    if (player.GuildId != "") {
        return &types.MsgGuildJoinProxyResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Cannot proxy join a player that is already in a guild")
    }

    // Add player to the guild
    player.GuildId = guild.Id

    // Connect player to the substation
    // Now let's get the player some power
    if (player.SubstationId == "") {
        // Connect Player to Substation
        k.SubstationConnectPlayer(ctx, substation, player)
    }

    // Give this user (aka the guild) the ability to update their substation
    playerObjectPermissionId := GetObjectPermissionIDBytes(player.Id, proxyPlayer.Id)
    k.PermissionAdd(ctx, playerObjectPermissionId, types.PermissionGrid)

	return &types.MsgGuildJoinProxyResponse{}, nil
}
