package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdatePfp(goCtx context.Context, msg *types.MsgGuildUpdatePfp) (*types.MsgGuildUpdateResponse, error) {
	emptyResponse := &types.MsgGuildUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetPlayerByAddress(msg.Creator)
	if err != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_update_pfp")
	}

	guild := cc.GetGuild(msg.GuildId)
	if guild.CheckGuild() != nil {
		return emptyResponse, types.NewObjectNotFoundError("guild", msg.GuildId)
	}

	permissionErr := cc.PermissionCheck(guild, player, types.PermUpdate)
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}

	if err := types.ValidatePfp(msg.Pfp); err != nil {
		return emptyResponse, err
	}

	guild.SetPfp(msg.Pfp)

	cc.CommitAll()
	return &types.MsgGuildUpdateResponse{}, nil
}
