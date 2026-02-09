package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	//"fmt"
)

func (k msgServer) StructDeactivate(goCtx context.Context, msg *types.MsgStructDeactivate) (*types.MsgStructStatusResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)
    cc := k.NewCurrentContext(ctx)
    defer cc.CommitAll()

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

    if structure.GetOwner().IsHalted() {
        return &types.MsgStructStatusResponse{}, types.NewPlayerHaltedError(structure.GetOwnerId(), "struct_deactivate").WithStruct(msg.StructId)
    }

    if !structure.LoadStruct(){
        return &types.MsgStructStatusResponse{}, types.NewObjectNotFoundError("struct", msg.StructId)
    }

    if !structure.IsBuilt() {
        structure.GetOwner().Discharge()
        return &types.MsgStructStatusResponse{}, types.NewStructStateError(msg.StructId, "building", "built", "deactivate")
    }

    if structure.IsOffline() {
        structure.GetOwner().Discharge()
        return &types.MsgStructStatusResponse{}, types.NewStructStateError(msg.StructId, "offline", "online", "deactivate")
    }

    if structure.GetOwner().IsOffline(){
        return &types.MsgStructStatusResponse{}, types.NewPlayerPowerError(structure.GetOwnerId(), "offline")
    }


    structure.GoOffline()

	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
