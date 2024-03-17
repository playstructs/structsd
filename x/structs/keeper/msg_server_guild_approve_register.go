package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildApproveRegister(goCtx context.Context, msg *types.MsgGuildApproveRegister) (*types.MsgGuildApproveRegisterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator), false)


    if (!playerFound) {
        return &types.MsgGuildApproveRegisterResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Could not perform guild action with non-player address (%s)", msg.Creator)
    }

    guild, guildFound := k.GetGuild(ctx, msg.GuildId)
    if (!guildFound) {
        return &types.MsgGuildApproveRegisterResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Referenced Guild (%s) not found", guild.Id)
    }


    guildObjectPermissionId := GetObjectPermissionIDBytes(msg.GuildId, player.Id)
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)

    // Check to make sure the player has permissions on the guild
    if (!k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.PermissionAssociations)) {
        return &types.MsgGuildApproveRegisterResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%s) has no Player Registration permissions ", player.Id)
    }

    // Make sure the address calling this has Associate permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssociations)) {
        return &types.MsgGuildApproveRegisterResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageGuild, "Calling address (%s) has no Guild Management permissions ", msg.Creator)
    }


    registeringPlayer, registeringPlayerFound := k.GetPlayer(ctx, msg.PlayerId, true)


    if (registeringPlayerFound) {
        if (msg.Approve) {
            k.GuildApproveRegisterRequest(ctx, guild, registeringPlayer)
        } else {
            k.GuildDenyRegisterRequest(ctx, guild, registeringPlayer)
        }
    }

	return &types.MsgGuildApproveRegisterResponse{}, nil
}
