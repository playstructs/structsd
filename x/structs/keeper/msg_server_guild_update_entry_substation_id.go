package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateEntrySubstationId(goCtx context.Context, msg *types.MsgGuildUpdateEntrySubstationId) (*types.MsgGuildUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (playerIndex == 0) {
     return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Guild update requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayerFromIndex(ctx, playerIndex, false)

    guild, guildFound := k.GetGuild(ctx, msg.GuildId)
    if (!guildFound) {
         return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrGuildNotFound, "Guild (%s) wasn't found. Can't update that which does not exist", msg.GuildId)
    }

    guildObjectPermissionId := GetObjectPermissionIDBytes(msg.GuildId, player.Id)
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)

    if (!k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.PermissionUpdate)) {
        return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrGuildUpdate, "Calling player (%s) has no permissions to update guild", player.Id)
    }

    // Make sure the address calling this has Associate permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets)) {
        return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageGuild, "Calling address (%s) has no Guild Management permissions ", msg.Creator)
    }

    if (msg.EntrySubstationId != "") {

        substationObjectPermissionId    := GetObjectPermissionIDBytes(msg.EntrySubstationId, player.Id)

        // check that the calling player has substation permissions
        if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {
            return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%s) has no Substation Connect Player permissions ", player.Id)
        }
        guild.SetEntrySubstationId(msg.EntrySubstationId)
    }

    k.SetGuild(ctx, guild)


	return &types.MsgGuildUpdateResponse{}, nil
}
