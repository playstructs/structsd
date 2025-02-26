package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) AgreementCapacityDecrease(goCtx context.Context, msg *types.MsgAgreementCapacityDecrease) (*types.MsgAgreementResponse, error) {
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

    // Decrease capacity
        // Decrease provider load
        // which increases duration
    agreement.CapacityDecrease(msg.CapacityDecrease)

    agreement.Commit()

	return &types.MsgAgreementResponse{}, nil
}
