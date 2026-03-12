package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

const maxIDLength = 256

func (k msgServer) PermissionGuildRankSet(goCtx context.Context, msg *types.MsgPermissionGuildRankSet) (*types.MsgPermissionResponse, error) {
	emptyResponse := &types.MsgPermissionResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	k.AddressEmitActivity(ctx, msg.Creator)

	if msg.ObjectId == "" {
		return emptyResponse, types.NewParameterValidationError("object_id", 0, "required")
	}
	if len(msg.ObjectId) > maxIDLength {
		return emptyResponse, types.NewParameterValidationError("object_id", 0, "exceeds_max_length")
	}
	if msg.GuildId == "" {
		return emptyResponse, types.NewParameterValidationError("guild_id", 0, "required")
	}
	if len(msg.GuildId) > maxIDLength {
		return emptyResponse, types.NewParameterValidationError("guild_id", 0, "exceeds_max_length")
	}
	if msg.Permission == 0 {
		return emptyResponse, types.NewParameterValidationError("permission", 0, "below_minimum").WithRange(1, 0)
	}

	player, err := cc.GetPlayerByAddress(msg.Creator)
	if err != nil {
		return emptyResponse, err
	}

	permissionedObject := cc.GetPermissionedObject(msg.ObjectId)
	if permissionedObject == nil {
		return emptyResponse, types.NewPermissionError("player", player.GetPlayerId(), "object", msg.ObjectId, msg.Permission, "permission_guild_rank_set")
	}

	guild := cc.GetGuild(msg.GuildId)
	if guild.CheckGuild() != nil {
		return emptyResponse, types.NewObjectNotFoundError("guild", msg.GuildId)
	}

	permissionErr := cc.PermissionCheck(permissionedObject, player, types.PermAdmin)
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}

	cc.SetPermissionsGuildRank(permissionedObject, guild, types.Permission(msg.Permission), msg.HighestRank)
	cc.CommitAll()
	return &types.MsgPermissionResponse{}, nil
}
