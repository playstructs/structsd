package keeper

import (
	"context"
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) StructOreRefineryComplete(goCtx context.Context, msg *types.MsgStructOreRefineryComplete) (*types.MsgStructOreRefineryStatusResponse, error) {
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
        return &types.MsgStructOreRefineryStatusResponse{}, permissionError
    }
    */

    // Is the Struct & Owner online?
    readinessError := structure.ReadinessCheck()
    if (readinessError != nil) {
        return &types.MsgStructOreRefineryStatusResponse{}, readinessError
    }

    refiningReadinessError := structure.CanOreRefine()
    if (refiningReadinessError != nil) {
        return &types.MsgStructOreRefineryStatusResponse{}, refiningReadinessError
    }

    activeOreRefiningSystemBlockString := strconv.FormatUint(structure.GetBlockStartOreRefine() , 10)
    hashInput := structure.StructId + "REFINE" + activeOreRefiningSystemBlockString + "NONCE" + msg.Nonce

    currentAge := uint64(ctx.BlockHeight()) - structure.GetBlockStartOreRefine()
    valid, achievedDifficulty := types.HashBuildAndCheckDifficulty(hashInput, msg.Proof, currentAge, structure.GetStructType().OreRefiningDifficulty)
    if !valid {
       return &types.MsgStructOreRefineryStatusResponse{}, types.NewWorkFailureError("refine", structure.StructId, hashInput)
    }

    structure.OreRefine()

    _ = ctx.EventManager().EmitTypedEvent(&types.EventAlphaRefine{&types.EventAlphaRefineDetail{PlayerId: structure.GetOwnerId(), PrimaryAddress: structure.GetOwner().GetPrimaryAddress(), Amount: 1}})
    _ = ctx.EventManager().EmitTypedEvent(&types.EventHashSuccess{&types.EventHashSuccessDetail{CallerAddress: msg.Creator, Category: "refine", Difficulty: achievedDifficulty, ObjectId: msg.StructId }})

	return &types.MsgStructOreRefineryStatusResponse{Struct: structure.GetStruct()}, nil
}
