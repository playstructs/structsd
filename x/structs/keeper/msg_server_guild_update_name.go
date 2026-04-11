package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateName(goCtx context.Context, msg *types.MsgGuildUpdateName) (*types.MsgGuildUpdateResponse, error) {
	emptyResponse := &types.MsgGuildUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetPlayerByAddress(msg.Creator)
	if err != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_update_name")
	}

	guild := cc.GetGuild(msg.GuildId)
	if guild.CheckGuild() != nil {
		return emptyResponse, types.NewObjectNotFoundError("guild", msg.GuildId)
	}

	permissionErr := cc.PermissionCheck(guild, player, types.PermUpdate)
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}

	if err := types.ValidateEntityName(msg.Name); err != nil {
		return emptyResponse, err
	}

	existingGuildId, taken := k.GetGuildIdByName(ctx, msg.Name)
	if taken && existingGuildId != msg.GuildId {
		return emptyResponse, types.ErrGuildNameTaken
	}

	oldName := guild.GetName()
	if oldName != "" {
		k.RemoveGuildNameIndex(ctx, oldName)
	}

	guild.SetName(msg.Name)
	k.SetGuildNameIndex(ctx, msg.Name, msg.GuildId)

	cc.CommitAll()
	return &types.MsgGuildUpdateResponse{}, nil
}
