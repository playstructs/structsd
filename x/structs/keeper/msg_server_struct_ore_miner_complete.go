package keeper

import (
	"context"
	"strconv"

	"structs/x/structs/types"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) StructOreMinerComplete(goCtx context.Context, msg *types.MsgStructOreMinerComplete) (*types.MsgStructOreMinerStatusResponse, error) {
    emptyResponse := &types.MsgStructOreMinerStatusResponse{}
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
	if readinessError != nil {
		return emptyResponse, readinessError
	}

	miningReadinessError := structure.CanOreMinePlanet()
	if miningReadinessError != nil {
		return emptyResponse, miningReadinessError
	}

	activeOreMiningSystemBlockString := strconv.FormatUint(structure.GetBlockStartOreMine(), 10)
	hashInput := msg.StructId + "MINE" + activeOreMiningSystemBlockString + "NONCE" + msg.Nonce

	blockHeight := uint64(ctx.BlockHeight())
	blockStart := structure.GetBlockStartOreMine()
	if blockHeight < blockStart {
		return emptyResponse, sdkerrors.Wrapf(types.ErrInvalidParameters, "block height %d precedes start block %d", blockHeight, blockStart)
	}
	currentAge := blockHeight - blockStart

	valid, achievedDifficulty := types.HashBuildAndCheckDifficulty(hashInput, msg.Proof, currentAge, structure.GetStructType().OreMiningDifficulty);
	if !valid {
		return emptyResponse, types.NewWorkFailureError("mine", structure.StructId, hashInput)
	}

	// Got this far, let's reward the player with some Ore
	structure.OreMinePlanet()

	_ = ctx.EventManager().EmitTypedEvent(&types.EventOreMine{&types.EventOreMineDetail{PlayerId: structure.GetOwnerId(), PrimaryAddress: structure.GetOwner().GetPrimaryAddress(), Amount: 1}})
    _ = ctx.EventManager().EmitTypedEvent(&types.EventHashSuccess{&types.EventHashSuccessDetail{CallerAddress: msg.Creator, Category: "mine", Difficulty: achievedDifficulty, ObjectId: msg.StructId }})

	cc.CommitAll()
	return &types.MsgStructOreMinerStatusResponse{Struct: structure.GetStruct()}, nil
}
