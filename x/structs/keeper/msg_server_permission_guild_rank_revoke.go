package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PermissionGuildRankRevoke(goCtx context.Context, msg *types.MsgPermissionGuildRankRevoke) (*types.MsgPermissionResponse, error) {
	emptyResponse := &types.MsgPermissionResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	k.AddressEmitActivity(ctx, msg.Creator)

	if msg.ObjectId == "" {
		return emptyResponse, types.NewParameterValidationError("object_id", 0, "required")
	}

	if msg.GuildId == "" {
		return emptyResponse, types.NewParameterValidationError("guild_id", 0, "required")
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
		return emptyResponse, types.NewPermissionError("player", player.GetPlayerId(), "object", msg.ObjectId, msg.Permission, "permission_guild_rank_revoke")
	}

	guild := cc.GetGuild(msg.GuildId)
	if guild.CheckGuild() != nil {
		return emptyResponse, types.NewObjectNotFoundError("guild", msg.GuildId)
	}

	permissionErr := cc.PermissionCheck(permissionedObject, player, types.Permission(msg.Permission))
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}

	cc.RemovePermissionsGuildRank(permissionedObject, guild, types.Permission(msg.Permission))
	cc.CommitAll()
	return &types.MsgPermissionResponse{}, nil
}
