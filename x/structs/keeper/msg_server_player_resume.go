package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	//"fmt"
)

func (k msgServer) PlayerResume(goCtx context.Context, msg *types.MsgPlayerResume) (*types.MsgPlayerResumeResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // load player
    player, playerLoadError := k.GetPlayerCacheFromId(ctx, msg.PlayerId)
    if (playerLoadError != nil) {
        return &types.MsgPlayerResumeResponse{}, playerLoadError
    }

    // Check to see if the caller has permissions to proceed
    permissionError := player.CanBeUpdatedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgPlayerResumeResponse{}, permissionError
    }

    if (player.GetCharge() < types.PlayerResumeCharge) {
        return &types.MsgPlayerResumeResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Resuming from Halt requires a charge of %d, but player (%s) only had %d", types.PlayerResumeCharge, msg.PlayerId , player.GetCharge())
    }

    player.Resume()
    player.Discharge()
    player.Commit()

	return &types.MsgPlayerResumeResponse{}, nil
}
