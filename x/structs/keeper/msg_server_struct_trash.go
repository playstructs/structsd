package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) StructTrash(goCtx context.Context, msg *types.MsgStructTrash) (*types.MsgStructStatusResponse, error) {
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

    if structure.IsDestroyed() {
        return emptyResponse, types.NewStructStateError(msg.StructId, "destroyed", "active", "trash")
    }

    // Trashing a struct costs the same charge as building it. Confirm the owner
    // can pay before destroying anything.
    owner := structure.GetOwner()
    buildCharge := structure.GetStructType().BuildCharge
    if (owner.GetCharge() < buildCharge) {
        return emptyResponse, types.NewInsufficientChargeError(owner.GetPlayerId(), buildCharge, owner.GetCharge(), "trash").WithStructType(structure.GetTypeId())
    }

    structure.DestroyAndCommit()
    owner.Discharge()

	cc.CommitAll()
	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
