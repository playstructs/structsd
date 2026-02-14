package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) AgreementCapacityDecrease(goCtx context.Context, msg *types.MsgAgreementCapacityDecrease) (*types.MsgAgreementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)
    activePlayer, _ := cc.GetPlayerByAddress(msg.Creator)

    agreement := cc.GetAgreement(msg.AgreementId)

    permissionError := agreement.CanUpdate(activePlayer)
    if (permissionError != nil) {
        return &types.MsgAgreementResponse{}, permissionError
    }

    // Checkpoint
    agreement.GetProvider().Checkpoint()

    // Decrease capacity
        // Decrease provider load
        // which increases duration
    agreement.CapacityDecrease(msg.CapacityDecrease)

	cc.CommitAll()
	return &types.MsgAgreementResponse{}, nil
}
