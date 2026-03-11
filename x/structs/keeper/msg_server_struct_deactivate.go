package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	//"fmt"
)

func (k msgServer) StructDeactivate(goCtx context.Context, msg *types.MsgStructDeactivate) (*types.MsgStructStatusResponse, error) {
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

    if !structure.LoadStruct(){
        return emptyResponse, types.NewObjectNotFoundError("struct", msg.StructId)
    }

    if !structure.IsBuilt() {
        return emptyResponse, types.NewStructStateError(msg.StructId, "building", "built", "deactivate")
    }

    if structure.IsOffline() {
        return emptyResponse, types.NewStructStateError(msg.StructId, "offline", "online", "deactivate")
    }

    if structure.GetOwner().IsOffline(){
        return emptyResponse, types.NewPlayerPowerError(structure.GetOwnerId(), "offline")
    }


    structure.GoOffline()

	cc.CommitAll()
	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
