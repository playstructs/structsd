package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateJoinInfusionMinimumBypassByRequest(goCtx context.Context, msg *types.MsgGuildUpdateJoinInfusionMinimumBypassByRequest) (*types.MsgGuildUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (playerIndex == 0) {
        return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Guild update requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayerFromIndex(ctx, playerIndex)

    guild, guildFound := k.GetGuild(ctx, msg.GuildId)
    if (!guildFound) {
            return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Guild (%s) wasn't found. Can't update that which does not exist", msg.GuildId)
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

    if (msg.GuildJoinBypassLevel != guild.JoinInfusionMinimumBypassByRequest) {
        guild.JoinInfusionMinimumBypassByRequest = msg.GuildJoinBypassLevel
        k.SetGuild(ctx, guild)
    }

	return &types.MsgGuildUpdateResponse{}, nil
}
