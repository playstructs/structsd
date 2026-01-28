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

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)


	structure := k.GetStructCacheFromId(ctx, msg.StructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBeHashedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructOreRefineryStatusResponse{}, permissionError
    }

    if structure.GetOwner().IsHalted() {
        return &types.MsgStructOreRefineryStatusResponse{}, types.NewPlayerHaltedError(structure.GetOwnerId(), "ore_refine_complete").WithStruct(msg.StructId)
    }

    // Is the Struct & Owner online?
    readinessError := structure.ReadinessCheck()
    if (readinessError != nil) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructOreRefineryStatusResponse{}, readinessError
    }

    refiningReadinessError := structure.CanOreRefine()
    if (refiningReadinessError != nil) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructOreRefineryStatusResponse{}, refiningReadinessError
    }

    activeOreRefiningSystemBlockString := strconv.FormatUint(structure.GetBlockStartOreRefine() , 10)
    hashInput := structure.StructId + "REFINE" + activeOreRefiningSystemBlockString + "NONCE" + msg.Nonce

    currentAge := uint64(ctx.BlockHeight()) - structure.GetBlockStartOreRefine()
    if (!types.HashBuildAndCheckDifficulty(hashInput, msg.Proof, currentAge, structure.GetStructType().GetOreRefiningDifficulty())) {
       //structure.GetOwner().Halt()
       return &types.MsgStructOreRefineryStatusResponse{}, types.NewWorkFailureError("refine", structure.StructId, hashInput)
    }

    structure.OreRefine()

    structure.Commit()

    _ = ctx.EventManager().EmitTypedEvent(&types.EventAlphaRefine{&types.EventAlphaRefineDetail{PlayerId: structure.GetOwnerId(), PrimaryAddress: structure.GetOwner().GetPrimaryAddress(), Amount: 1}})

	return &types.MsgStructOreRefineryStatusResponse{Struct: structure.GetStruct()}, nil
}
