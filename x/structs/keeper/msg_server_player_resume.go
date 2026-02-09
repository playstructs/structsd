package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	//"fmt"
)

func (k msgServer) PlayerResume(goCtx context.Context, msg *types.MsgPlayerResume) (*types.MsgPlayerResumeResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)
    cc := k.NewCurrentContext(ctx)
    defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // load player
    player, playerLoadError := cc.GetPlayer(msg.PlayerId)
    if (playerLoadError != nil) {
        return &types.MsgPlayerResumeResponse{}, playerLoadError
    }

    // Check to see if the caller has permissions to proceed
    permissionError := player.CanBeUpdatedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgPlayerResumeResponse{}, permissionError
    }

    if (player.GetCharge() < types.PlayerResumeCharge) {
        return &types.MsgPlayerResumeResponse{}, types.NewInsufficientChargeError(msg.PlayerId, types.PlayerResumeCharge, player.GetCharge(), "resume")
    }

    player.Resume()
    player.Discharge()

	return &types.MsgPlayerResumeResponse{}, nil
}
