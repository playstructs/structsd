package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateJoinInfusionMinimum(goCtx context.Context, msg *types.MsgGuildUpdateJoinInfusionMinimum) (*types.MsgGuildUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (playerIndex == 0) {
        return &types.MsgGuildUpdateResponse{}, types.NewPlayerRequiredError(msg.Creator, "guild_update_join_infusion_minimum")
    }
    player, _ := k.GetPlayerFromIndex(ctx, playerIndex)

    guild, guildFound := k.GetGuild(ctx, msg.GuildId)
    if (!guildFound) {
            return &types.MsgGuildUpdateResponse{}, types.NewObjectNotFoundError("guild", msg.GuildId)
    }

    guildObjectPermissionId := GetObjectPermissionIDBytes(msg.GuildId, player.Id)
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)

    if (!k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.PermissionUpdate)) {
        return &types.MsgGuildUpdateResponse{}, types.NewPermissionError("player", player.Id, "guild", msg.GuildId, uint64(types.PermissionUpdate), "guild_update")
    }

    // Make sure the address calling this has Associate permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets)) {
        return &types.MsgGuildUpdateResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionAssets), "guild_management")
    }

    if (msg.JoinInfusionMinimum != guild.JoinInfusionMinimum) {
        guild.JoinInfusionMinimum = msg.JoinInfusionMinimum
        k.SetGuild(ctx, guild)
    }


	return &types.MsgGuildUpdateResponse{}, nil
}
