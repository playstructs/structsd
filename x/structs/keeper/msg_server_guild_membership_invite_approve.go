package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildMembershipInviteApprove(goCtx context.Context, msg *types.MsgGuildMembershipInviteApprove) (*types.MsgGuildMembershipResponse, error) {
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

    if (msg.PlayerId == "") {
        msg.PlayerId = callingPlayer.GetPlayerId()
    }

	if msg.GuildId == "" {
		msg.GuildId = callingPlayer.GetGuildId()
	}

    guildMembershipApplication, guildMembershipApplicationError := cc.GetGuildMembershipApplicationCache(callingPlayer, types.GuildJoinType_invite, msg.GuildId, msg.PlayerId)
    if guildMembershipApplicationError != nil {
        return emptyResponse, guildMembershipApplicationError
    }

    guildMembershipApplicationError = guildMembershipApplication.VerifyInviteAsPlayer()
    if guildMembershipApplicationError != nil {
        return emptyResponse, guildMembershipApplicationError
    }

	if msg.SubstationId != "" {
	    substationOverrideError := guildMembershipApplication.SetSubstationIdOverride(msg.SubstationId)
	    if substationOverrideError != nil {
	        return emptyResponse, substationOverrideError
	    }
	}

    guildMembershipApplicationError = guildMembershipApplication.ApproveInvite()

	cc.CommitAll()
	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication.GuildMembershipApplication}, nil
}
