package keeper

import (
	"context"

	"structs/x/structs/types"

	//sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) GuildMembershipInvite(goCtx context.Context, msg *types.MsgGuildMembershipInvite) (*types.MsgGuildMembershipResponse, error) {
    emptyResponse := &types.MsgGuildMembershipResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return emptyResponse, err
    }

	if msg.GuildId == "" {
		msg.GuildId = callingPlayer.GetGuildId()
	}

    // TODO Confirm permissions are being handled properly within.
    guildMembershipApplication, guildMembershipApplicationError := cc.GetGuildMembershipApplicationCache(callingPlayer, types.GuildJoinType_invite, msg.GuildId, msg.PlayerId)
    if guildMembershipApplicationError != nil {
        return emptyResponse, guildMembershipApplicationError
    }

	/*
	 * We're either going to load up the substation provided as an
	 * override, or we're going to default to using the guild entry substation
	 */
	if msg.SubstationId != "" {
	    substationOverrideError := guildMembershipApplication.SetSubstationIdOverride(msg.SubstationId)
	    if substationOverrideError != nil {
	        return emptyResponse, substationOverrideError
	    }
	}

	cc.CommitAll()
	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication.GuildMembershipApplication}, nil
}
