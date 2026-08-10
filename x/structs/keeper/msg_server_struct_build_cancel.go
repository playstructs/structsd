package keeper

import (
	"context"
    //"strconv"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) StructBuildCancel(goCtx context.Context, msg *types.MsgStructBuildCancel) (*types.MsgStructStatusResponse, error) {
    emptyResponse := &types.MsgStructStatusResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetSigningPlayer(msg.Creator)
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

    // DestroyAndCommit is idempotent, so a repeat cancel is already harmless.
    // Reject it outright rather than reporting success for work that did not
    // happen.
    if structure.IsDestroyed() {
        return emptyResponse, types.NewStructStateError(msg.StructId, "destroyed", "building", "build_cancel")
    }

    if structure.IsBuilt() {
        return emptyResponse, types.NewStructStateError(msg.StructId, "built", "building", "build_cancel")
    }

    structure.DestroyAndCommit()

	cc.CommitAll()
	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
