package keeper

import (
	"context"
    //"strconv"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) StructBuildCancel(goCtx context.Context, msg *types.MsgStructBuildCancel) (*types.MsgStructStatusResponse, error) {
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

    if !structure.LoadStruct(){
        return &types.MsgStructStatusResponse{}, types.NewObjectNotFoundError("struct", msg.StructId)
    }

    if structure.IsBuilt() {
        structure.GetOwner().Discharge()
        return &types.MsgStructStatusResponse{}, types.NewStructStateError(msg.StructId, "built", "building", "build_cancel")
    }

    structure.DestroyAndCommit()

	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
