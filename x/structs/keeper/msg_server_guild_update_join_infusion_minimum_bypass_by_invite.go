package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateJoinInfusionMinimumBypassByInvite(goCtx context.Context, msg *types.MsgGuildUpdateJoinInfusionMinimumBypassByInvite) (*types.MsgGuildUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)


    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgGuildUpdateResponse{}, types.NewPlayerRequiredError(msg.Creator, "guild_update_join_bypass_invite")
    }

    guild := cc.GetGuild(msg.GuildId)
    if guild.CheckGuild() != nil {
            return &types.MsgGuildUpdateResponse{}, types.NewObjectNotFoundError("guild", msg.GuildId)
    }

    permissionErr := guild.CanUpdateJoinConstraintsBy(player)
    if permissionErr != nil {
        return &types.MsgGuildUpdateResponse{}, permissionErr
    }

    if (msg.GuildJoinBypassLevel != guild.GetGuild().JoinInfusionMinimumBypassByInvite) {
        guild.SetJoinInfusionMinimumBypassByInvite(msg.GuildJoinBypassLevel)
    }

	cc.CommitAll()
	return &types.MsgGuildUpdateResponse{}, nil
}
