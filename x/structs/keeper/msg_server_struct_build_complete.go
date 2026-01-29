package keeper

import (
	"context"
	"strconv"
	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) StructBuildComplete(goCtx context.Context, msg *types.MsgStructBuildComplete) (*types.MsgStructStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	// load struct
	structure := k.GetStructCacheFromId(ctx, msg.StructId)

	// Check to see if the caller has permissions to proceed
	/*
	   callerID, isOwner, permissionError := structure.CanBeHashedBy(msg.Creator)
	   if (permissionError != nil) {
	       return &types.MsgStructStatusResponse{}, permissionError
	   }
	*/

	if !structure.LoadStruct() {
		return &types.MsgStructStatusResponse{}, types.NewObjectNotFoundError("struct", msg.StructId)
	}

	if structure.GetOwner().IsHalted() {
		return &types.MsgStructStatusResponse{}, types.NewPlayerHaltedError(structure.GetOwnerId(), "struct_build_complete").WithStruct(msg.StructId)
	}

	if structure.IsBuilt() {
		structure.GetOwner().Discharge()
		structure.GetOwner().Commit()
		return &types.MsgStructStatusResponse{}, types.NewStructStateError(msg.StructId, "built", "building", "build_complete")
	}

	// Check Player Charge
	/*
	   if (structure.GetOwner().GetCharge() < structure.GetStructType().ActivateCharge) {
	       err := types.NewInsufficientChargeError(structure.GetOwnerId(), structure.GetStructType().ActivateCharge, structure.GetOwner().GetCharge(), "struct_build_complete").WithStructType(structure.GetStructType().Id)
	       structure.GetOwner().Discharge()
	       structure.GetOwner().Commit()
	       return &types.MsgStructStatusResponse{}, err
	   }
	*/

	if structure.GetOwner().IsOffline() {
		return &types.MsgStructStatusResponse{}, types.NewPlayerPowerError(structure.GetOwnerId(), "offline")
	}

	// Remove the BuildDraw load
	structure.GetOwner().StructsLoadDecrement(structure.GetStructType().BuildDraw)

	if !structure.GetOwner().CanSupportLoadAddition(structure.GetStructType().PassiveDraw) {
		structure.GetOwner().StructsLoadIncrement(structure.GetStructType().BuildDraw)
		//structure.GetOwner().Discharge()
		structure.GetOwner().Commit()
		return &types.MsgStructStatusResponse{}, types.NewPlayerPowerError(structure.GetOwnerId(), "capacity_exceeded").WithCapacity(structure.GetStructType().PassiveDraw, structure.GetOwner().GetAvailableCapacity())
	}

	// Check the Proof
	buildStartBlockString := strconv.FormatUint(structure.GetBlockStartBuild(), 10)
	hashInput := structure.GetStructId() + "BUILD" + buildStartBlockString + "NONCE" + msg.Nonce

	currentAge := uint64(ctx.BlockHeight()) - structure.GetBlockStartBuild()

    valid, achievedDifficulty := types.HashBuildAndCheckDifficulty(hashInput, msg.Proof, currentAge, structure.GetStructType().BuildDifficulty)
	if !valid {
		structure.GetOwner().StructsLoadIncrement(structure.GetStructType().BuildDraw)
		//structure.GetOwner().Discharge()
		//structure.GetOwner().Halt()
		structure.GetOwner().Commit()
		return &types.MsgStructStatusResponse{}, types.NewWorkFailureError("build", structure.GetStructId(), hashInput)
	}

	structure.StatusAddBuilt()
	structure.GoOnline()
	structure.Commit()

    _ = ctx.EventManager().EmitTypedEvent(&types.EventHashSuccess{&types.EventHashSuccessDetail{CallerAddress: msg.Creator, Category: "build", Difficulty: achievedDifficulty, ObjectId: msg.StructId }})


	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
