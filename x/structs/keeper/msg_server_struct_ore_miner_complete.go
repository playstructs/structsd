package keeper

import (
	"context"
	"strconv"

	//"fmt"

	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) StructOreMinerComplete(goCtx context.Context, msg *types.MsgStructOreMinerComplete) (*types.MsgStructOreMinerStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	structure := cc.GetStruct(msg.StructId)

	// Check to see if the caller has permissions to proceed
	/*
	   callerID, isOwner, permissionError := structure.CanBeHashedBy(msg.Creator)
	   if (permissionError != nil) {
	       return &types.MsgStructOreMinerStatusResponse{}, permissionError
	   }
	*/

	// Is the Struct & Owner online?
	readinessError := structure.ReadinessCheck()
	if readinessError != nil {
		return &types.MsgStructOreMinerStatusResponse{}, readinessError
	}

	miningReadinessError := structure.CanOreMinePlanet()
	if miningReadinessError != nil {
		return &types.MsgStructOreMinerStatusResponse{}, miningReadinessError
	}

	activeOreMiningSystemBlockString := strconv.FormatUint(structure.GetBlockStartOreMine(), 10)
	hashInput := msg.StructId + "MINE" + activeOreMiningSystemBlockString + "NONCE" + msg.Nonce

	currentAge := uint64(ctx.BlockHeight()) - structure.GetBlockStartOreMine()

	valid, achievedDifficulty := types.HashBuildAndCheckDifficulty(hashInput, msg.Proof, currentAge, structure.GetStructType().OreMiningDifficulty);
	if !valid {
		return &types.MsgStructOreMinerStatusResponse{}, types.NewWorkFailureError("mine", structure.StructId, hashInput)
	}

	// Got this far, let's reward the player with some Ore
	structure.OreMinePlanet()

	_ = ctx.EventManager().EmitTypedEvent(&types.EventOreMine{&types.EventOreMineDetail{PlayerId: structure.GetOwnerId(), PrimaryAddress: structure.GetOwner().GetPrimaryAddress(), Amount: 1}})
    _ = ctx.EventManager().EmitTypedEvent(&types.EventHashSuccess{&types.EventHashSuccessDetail{CallerAddress: msg.Creator, Category: "mine", Difficulty: achievedDifficulty, ObjectId: msg.StructId }})

	return &types.MsgStructOreMinerStatusResponse{Struct: structure.GetStruct()}, nil
}
