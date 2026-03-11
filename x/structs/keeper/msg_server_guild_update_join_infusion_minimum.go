package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateJoinInfusionMinimum(goCtx context.Context, msg *types.MsgGuildUpdateJoinInfusionMinimum) (*types.MsgGuildUpdateResponse, error) {
    emptyResponse := &types.MsgGuildUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)


    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_update_join_infusion_minimum")
    }

    guild := cc.GetGuild(msg.GuildId)
    if guild.CheckGuild() != nil {
            return emptyResponse, types.NewObjectNotFoundError("guild", msg.GuildId)
    }

    permissionErr := guild.CanUpdateJoinConstraintsBy(player)
    if permissionErr != nil {
        return emptyResponse, permissionErr
    }

    if (msg.JoinInfusionMinimum != guild.GetGuild().JoinInfusionMinimum) {
        guild.SetJoinInfusionMinimum(msg.JoinInfusionMinimum)
    }

	cc.CommitAll()
	return &types.MsgGuildUpdateResponse{}, nil
}
