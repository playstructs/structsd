package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	//"fmt"
)

func (k msgServer) StructActivate(goCtx context.Context, msg *types.MsgStructActivate) (*types.MsgStructStatusResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)
    cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // load struct
    structure := cc.GetStruct(msg.StructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    // Check Activation Readiness
        // Check Struct is Built
        // Check Struct is Offline
        // Check Player is Online
        // Check Player capacity
    readinessError := structure.ActivationReadinessCheck()
    if (readinessError != nil) {
        return &types.MsgStructStatusResponse{}, readinessError
    }

    if structure.GetOwner().GetCharge() < structure.GetStructType().ActivateCharge {
        return &types.MsgStructStatusResponse{}, types.NewInsufficientChargeError(structure.GetOwnerId(), structure.GetStructType().ActivateCharge, structure.GetOwner().GetCharge(), "activate").WithStructType(structure.GetTypeId()).WithStructId(msg.StructId)
    }

    structure.GoOnline()

	cc.CommitAll()
	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
