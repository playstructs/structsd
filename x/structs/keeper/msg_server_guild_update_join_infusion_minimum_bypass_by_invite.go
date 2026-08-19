package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateJoinInfusionMinimumBypassByInvite(goCtx context.Context, msg *types.MsgGuildUpdateJoinInfusionMinimumBypassByInvite) (*types.MsgGuildUpdateResponse, error) {
    emptyResponse := &types.MsgGuildUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)


    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    player, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
        return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_update_join_bypass_invite")
    }

    guild := cc.GetGuild(msg.GuildId)
    if guild.CheckGuild() != nil {
            return emptyResponse, types.NewObjectNotFoundError("guild", msg.GuildId)
    }

    permissionErr := guild.CanUpdateJoinConstraintsBy(player)
    if permissionErr != nil {
        return emptyResponse, permissionErr
    }

    // Validated ahead of the equality guard: a level that matches what is
    // already stored is still refused, so a corrupted record cannot be
    // rewritten to itself and read as an accepted value.
    if !msg.GuildJoinBypassLevel.IsValid() {
        return emptyResponse, errorsmod.Wrapf(types.ErrInvalidGuildJoinBypassLevel, "level (%d) on guild (%s)", int32(msg.GuildJoinBypassLevel), msg.GuildId)
    }

    if (msg.GuildJoinBypassLevel != guild.GetGuild().JoinInfusionMinimumBypassByInvite) {
        if setErr := guild.SetJoinInfusionMinimumBypassByInvite(msg.GuildJoinBypassLevel); setErr != nil {
            return emptyResponse, setErr
        }
    }

	cc.CommitAll()
	return &types.MsgGuildUpdateResponse{}, nil
}
