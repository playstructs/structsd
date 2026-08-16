package keeper

import (
	"context"
	"strconv"
	"structs/x/structs/types"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) StructBuildComplete(goCtx context.Context, msg *types.MsgStructBuildComplete) (*types.MsgStructStatusResponse, error) {
    emptyResponse := &types.MsgStructStatusResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    callingPlayer, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	// load struct
	structure := cc.GetStruct(msg.StructId)

	// Check to see if the caller has permissions to proceed
    permissionError := structure.GetOwner().CanBuildHashedBy(callingPlayer)
    if (permissionError != nil) {
       return emptyResponse, permissionError
    }


	if !structure.LoadStruct() {
		return emptyResponse, types.NewObjectNotFoundError("struct", msg.StructId)
	}

	// A cancelled build stays unbuilt and slot-resident until the sweep, so
	// without this it would satisfy the check below and could be completed —
	// setting Built and re-adding the owner's load against a struct already
	// queued for deletion.
	if structure.IsDestroyed() {
		return emptyResponse, types.NewStructStateError(msg.StructId, "destroyed", "building", "build_complete")
	}

	if structure.IsBuilt() {
		return emptyResponse, types.NewStructStateError(msg.StructId, "built", "building", "build_complete")
	}

	if structure.GetOwner().IsOffline() {
		return emptyResponse, types.NewPlayerPowerError(structure.GetOwnerId(), "offline")
	}

	// Remove the BuildDraw load
	structure.GetOwner().StructsLoadDecrement(structure.GetStructType().BuildDraw)

	if !structure.GetOwner().CanSupportLoadAddition(structure.GetStructType().PassiveDraw) {
		return emptyResponse, types.NewPlayerPowerError(structure.GetOwnerId(), "capacity_exceeded").WithCapacity(structure.GetStructType().PassiveDraw, structure.GetOwner().GetAvailableCapacity())
	}

	// Check the Proof
	buildStartBlockString := strconv.FormatUint(structure.GetBlockStartBuild(), 10)
	hashInput := structure.GetStructId() + "BUILD" + buildStartBlockString + "NONCE" + msg.Nonce

	blockHeight := uint64(ctx.BlockHeight())
	blockStart := structure.GetBlockStartBuild()
	if blockHeight < blockStart {
		return emptyResponse, sdkerrors.Wrapf(types.ErrInvalidParameters, "block height %d precedes start block %d", blockHeight, blockStart)
	}
	currentAge := blockHeight - blockStart

    valid, achievedDifficulty := types.HashBuildAndCheckDifficulty(hashInput, msg.Proof, currentAge, structure.GetStructType().BuildDifficulty)
	if !valid {
		return emptyResponse, types.NewWorkFailureError("build", structure.GetStructId(), hashInput)
	}

	structure.StatusAddBuilt()
	structure.GoOnline()

    _ = ctx.EventManager().EmitTypedEvent(&types.EventHashSuccess{&types.EventHashSuccessDetail{CallerAddress: msg.Creator, Category: "build", Difficulty: achievedDifficulty, ObjectId: msg.StructId }})

	cc.CommitAll()
	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
