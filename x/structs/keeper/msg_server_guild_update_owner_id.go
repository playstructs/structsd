package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateOwnerId(goCtx context.Context, msg *types.MsgGuildUpdateOwnerId) (*types.MsgGuildUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Guild update requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)

    guild, guildFound := k.GetGuild(ctx, msg.Id)
    if (!guildFound) {
            return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrGuildNotFound, "Guild wasn't found. Can't update that which does not exist", msg.Id)
    }

    guildObjectId           := GetObjectID(types.ObjectType_guild, msg.GuildId)
    guildObjectPermissionId := GetObjectPermissionIDBytes(guildObjectId, player.Id)
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)

    if (!k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.Permission(types.GuildPermissionUpdate))) {
        return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrGuildUpdate, "Calling player (%d) has no permissions to update guild", player.Id)
    }

    // Make sure the address calling this has Associate permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionManageGuild))) {
        return &types.MsgGuildApproveRegisterResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageGuild, "Calling address (%s) has no Guild Management permissions ", msg.Creator)
    }

    if (guild.Owner != msg.Owner) {
        _, guildOwnerFound := k.GetPlayer(ctx, msg.Owner)
        if (!guildOwnerFound) {
            return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrGuildUpdate, "Guild could not change to new owner (%d) because they weren't found", msg.Owner)
        }
        guild.SetOwner(msg.Owner)
        k.SetGuild(ctx, guild)
    }

	return &types.MsgGuildUpdateResponse{}, nil
}
