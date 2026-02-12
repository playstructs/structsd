package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildMembershipRequestRevoke(goCtx context.Context, msg *types.MsgGuildMembershipRequestRevoke) (*types.MsgGuildMembershipResponse, error) {
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

    if (msg.PlayerId == "") {
        msg.PlayerId = callingPlayer.GetPlayerId()
    }

	if msg.GuildId == "" {
		msg.GuildId = callingPlayer.GetGuildId()
	}

    guildMembershipApplication, guildMembershipApplicationError := cc.GetGuildMembershipApplicationCache(callingPlayer, types.GuildJoinType_request, msg.GuildId, msg.PlayerId)
    if guildMembershipApplicationError != nil {
        return &types.MsgGuildMembershipResponse{}, guildMembershipApplicationError
    }

    guildMembershipApplicationError = guildMembershipApplication.VerifyRequestAsPlayer()
    if guildMembershipApplicationError != nil {
        return &types.MsgGuildMembershipResponse{}, guildMembershipApplicationError
    }

    guildMembershipApplicationError = guildMembershipApplication.RevokeRequest()

	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication.GuildMembershipApplication}, nil
}
