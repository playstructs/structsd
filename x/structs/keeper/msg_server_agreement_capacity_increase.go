package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) AgreementCapacityIncrease(goCtx context.Context, msg *types.MsgAgreementCapacityIncrease) (*types.MsgAgreementResponse, error) {
    emptyResponse := &types.MsgAgreementResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)
    activePlayer, lookupErr := cc.GetSigningPlayer(msg.Creator)
    if lookupErr != nil {
        return emptyResponse, lookupErr
    }

    agreement := cc.GetAgreement(msg.AgreementId)

    permissionError := agreement.CanUpdate(activePlayer)
    if (permissionError != nil) {
        return emptyResponse, permissionError
    }

    // Checkpoint before the load changes, or the new load gets billed across the
    // span the old one was serving.
    if err := agreement.GetProvider().Checkpoint(); err != nil {
        return emptyResponse, err
    }

    // increase capacity
        // increase provider load
        // which decreases duration
    if err := agreement.CapacityIncrease(msg.CapacityIncrease); err != nil {
        return emptyResponse, err
    }

	cc.CommitAll()
	return &types.MsgAgreementResponse{}, nil
}
