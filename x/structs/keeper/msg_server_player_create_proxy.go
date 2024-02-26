package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
        return &types.MsgPlayerCreateProxyResponse{}, sdkerrors.Wrapf(types.ErrGuildNotFound, "Referenced Guild (%d) not found", guild.Id)
    }

    // Check on the Guild Join Type
    // The Guild can either openly allow GuildJoinType_Proxy (or any more open join type)
    // Or, if the Guild acceptance is locked down then we'll look to player permissions

    guildObjectId           := GetObjectIDBytes(types.ObjectType_guild, guild.Id)
    guildObjectPermissionId := GetObjectPermissionIDBytes(guildObjectId, proxyPlayer.Id)
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)

    // Check to make sure the player has permissions on the guild
    if (!k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.Permission(types.GuildPermissionRegisterPlayer))) {
        return &types.MsgPlayerCreateProxyResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%d) has no Player Registration permissions ", proxyPlayer.Id)
    }

    // Make sure the address calling this has Associate permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionManageGuild))) {
        return &types.MsgGuildApproveRegisterResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageGuild, "Calling address (%s) has no Guild Management permissions ", msg.Creator)
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

    // Give this user (aka the guild) the ability to update their substation
    k.PlayerPermissionAdd(ctx, player.Id, proxyPlayer.Id, types.PlayerPermissionSubstation)


	return &types.MsgPlayerCreateProxyResponse{}, nil
}
