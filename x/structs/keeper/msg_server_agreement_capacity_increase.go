package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) AgreementCapacityIncrease(goCtx context.Context, msg *types.MsgAgreementCapacityIncrease) (*types.MsgAgreementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)
    activePlayer, _ := k.GetPlayerCacheFromAddress(ctx, msg.Creator)

    agreement := k.GetAgreementCacheFromId(ctx, msg.AgreementId)

    permissionError := agreement.CanUpdate(&activePlayer)
    if (permissionError != nil) {
        return &types.MsgAgreementResponse{}, permissionError
    }

    // Checkpoint
    agreement.GetProvider().Checkpoint()

    agreement.CapacityIncrease(msg.CapacityIncrease)
    // increase capacity
        // increase provider load
        // which decreases duration

    //commit


	return &types.MsgAgreementResponse{}, nil
}
