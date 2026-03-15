package keeper

import (
	"context"
	"strconv"

	"structs/x/structs/types"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) StructOreRefineryComplete(goCtx context.Context, msg *types.MsgStructOreRefineryComplete) (*types.MsgStructOreRefineryStatusResponse, error) {
    emptyResponse := &types.MsgStructOreRefineryStatusResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }

	structure := cc.GetStruct(msg.StructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBeHashedBy(callingPlayer)
    if (permissionError != nil) {
        return emptyResponse, permissionError
    }


    // Is the Struct & Owner online?
    readinessError := structure.ReadinessCheck()
    if (readinessError != nil) {
        return emptyResponse, readinessError
    }

    refiningReadinessError := structure.CanOreRefine()
    if (refiningReadinessError != nil) {
        return emptyResponse, refiningReadinessError
    }

    activeOreRefiningSystemBlockString := strconv.FormatUint(structure.GetBlockStartOreRefine() , 10)
    hashInput := structure.StructId + "REFINE" + activeOreRefiningSystemBlockString + "NONCE" + msg.Nonce

    blockHeight := uint64(ctx.BlockHeight())
    blockStart := structure.GetBlockStartOreRefine()
    if blockHeight < blockStart {
        return emptyResponse, sdkerrors.Wrapf(types.ErrInvalidParameters, "block height %d precedes start block %d", blockHeight, blockStart)
    }
    currentAge := blockHeight - blockStart
    valid, achievedDifficulty := types.HashBuildAndCheckDifficulty(hashInput, msg.Proof, currentAge, structure.GetStructType().OreRefiningDifficulty)
    if !valid {
       return emptyResponse, types.NewWorkFailureError("refine", structure.StructId, hashInput)
    }

    if err := structure.OreRefine(); err != nil {
        return emptyResponse, err
    }

    _ = ctx.EventManager().EmitTypedEvent(&types.EventAlphaRefine{&types.EventAlphaRefineDetail{PlayerId: structure.GetOwnerId(), PrimaryAddress: structure.GetOwner().GetPrimaryAddress(), Amount: 1}})
    _ = ctx.EventManager().EmitTypedEvent(&types.EventHashSuccess{&types.EventHashSuccessDetail{CallerAddress: msg.Creator, Category: "refine", Difficulty: achievedDifficulty, ObjectId: msg.StructId }})

	cc.CommitAll()
	return &types.MsgStructOreRefineryStatusResponse{Struct: structure.GetStruct()}, nil
}
