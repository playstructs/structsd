package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	//"fmt"
)

func (k msgServer) StructActivate(goCtx context.Context, msg *types.MsgStructActivate) (*types.MsgStructStatusResponse, error) {
    emptyResponse := &types.MsgStructStatusResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
    cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }

    // load struct
    structure := cc.GetStruct(msg.StructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(callingPlayer)
    if (permissionError != nil) {
        return emptyResponse, permissionError
    }

    // Check Activation Readiness
        // Check Struct is Built
        // Check Struct is Offline
        // Check Player is Online
        // Check Player capacity
    readinessError := structure.ActivationReadinessCheck()
    if (readinessError != nil) {
        return emptyResponse, readinessError
    }

    if structure.GetOwner().GetCharge() < structure.GetStructType().ActivateCharge {
        return emptyResponse, types.NewInsufficientChargeError(structure.GetOwnerId(), structure.GetStructType().ActivateCharge, structure.GetOwner().GetCharge(), "activate").WithStructType(structure.GetTypeId()).WithStructId(msg.StructId)
    }

    // MsgStructActivate is registered in app/ante/maps.go::ChargeMessages, which
    // means the ante chain reserves the player's per-block charge slot for this
    // transaction. The handler must therefore Discharge() to actually consume
    // it; otherwise the player's `lastAction` is never updated and the ante's
    // charge floor check (StructsDecorator) silently disagrees with the ante's
    // per-tx dedup (ThrottleDecorator) about whether the slot was used.
    structure.GetOwner().Discharge()

    structure.GoOnline()

	cc.CommitAll()
	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
