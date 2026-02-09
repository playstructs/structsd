package keeper

import (
	"context"

	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) GuildUpdateOwnerId(goCtx context.Context, msg *types.MsgGuildUpdateOwnerId) (*types.MsgGuildUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetPlayerByAddress(msg.Creator)
	if err != nil {
		return &types.MsgGuildUpdateResponse{}, types.NewPlayerRequiredError(msg.Creator, "guild_update_owner")
	}

	guild, guildFound := k.GetGuild(ctx, msg.GuildId)
	if !guildFound {
		return &types.MsgGuildUpdateResponse{}, types.NewObjectNotFoundError("guild", msg.GuildId)
	}

	guildObjectPermissionId := GetObjectPermissionIDBytes(msg.GuildId, player.PlayerId)
	addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

	if !k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.PermissionUpdate) {
		return &types.MsgGuildUpdateResponse{}, types.NewPermissionError("player", player.PlayerId, "guild", msg.GuildId, uint64(types.PermissionUpdate), "guild_update")
	}

	// Make sure the address calling this has Associate permissions
	if !k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets) {
		return &types.MsgGuildUpdateResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionAssets), "guild_management")
	}

	if guild.Owner != msg.Owner {
		_, err = cc.GetPlayer(msg.Owner)
		if err != nil {
			return &types.MsgGuildUpdateResponse{}, types.NewObjectNotFoundError("player", msg.Owner)
		}
		guild.SetOwner(msg.Owner)
		k.SetGuild(ctx, guild)
	}

	return &types.MsgGuildUpdateResponse{}, nil
}
