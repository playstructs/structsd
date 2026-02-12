package keeper

import (
	"context"

	"structs/x/structs/types"

	//sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) GuildMembershipInvite(goCtx context.Context, msg *types.MsgGuildMembershipInvite) (*types.MsgGuildMembershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgGuildMembershipResponse{}, err
    }

    // Use cache permission methods
    callingPlayerPermissionError := callingPlayer.CanBeAdministratedBy(msg.Creator, types.PermissionAssociations)
    if callingPlayerPermissionError != nil {
        return &types.MsgGuildMembershipResponse{}, callingPlayerPermissionError
    }

	if msg.GuildId == "" {
		msg.GuildId = callingPlayer.GetGuildId()
	}

    guildMembershipApplication, guildMembershipApplicationError := cc.GetGuildMembershipApplicationCache(callingPlayer, types.GuildJoinType_invite, msg.GuildId, msg.PlayerId)
    if guildMembershipApplicationError != nil {
        return &types.MsgGuildMembershipResponse{}, guildMembershipApplicationError
    }

	/*
	 * We're either going to load up the substation provided as an
	 * override, or we're going to default to using the guild entry substation
	 */
	if msg.SubstationId != "" {
	    substationOverrideError := guildMembershipApplication.SetSubstationIdOverride(msg.SubstationId)
	    if substationOverrideError != nil {
	        return &types.MsgGuildMembershipResponse{}, substationOverrideError
	    }
	}

	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication.GuildMembershipApplication}, nil
}
