package keeper

import (
	"context"
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) StructOreMinerComplete(goCtx context.Context, msg *types.MsgStructOreMinerComplete) (*types.MsgStructOreMinerStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	structure := k.GetStructCacheFromId(ctx, msg.StructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBeHashedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructOreMinerStatusResponse{}, permissionError
    }

    if structure.GetOwner().IsHalted() {
        return &types.MsgStructOreMinerStatusResponse{}, types.NewPlayerHaltedError(structure.GetOwnerId(), "ore_mine_complete").WithStruct(msg.StructId)
    }

    // Is the Struct & Owner online?
    readinessError := structure.ReadinessCheck()
    if (readinessError != nil) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructOreMinerStatusResponse{}, readinessError
    }

    miningReadinessError := structure.CanOreMinePlanet()
    if (miningReadinessError != nil) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructOreMinerStatusResponse{}, miningReadinessError
    }

    activeOreMiningSystemBlockString    := strconv.FormatUint(structure.GetBlockStartOreMine() , 10)
    hashInput                           := msg.StructId + "MINE" + activeOreMiningSystemBlockString + "NONCE" + msg.Nonce

    currentAge := uint64(ctx.BlockHeight()) - structure.GetBlockStartOreMine()
    if (!types.HashBuildAndCheckDifficulty(hashInput, msg.Proof, currentAge, structure.GetStructType().GetOreMiningDifficulty())) {
       //structure.GetOwner().Halt()
       return &types.MsgStructOreMinerStatusResponse{}, types.NewWorkFailureError("mine", structure.StructId, hashInput)
    }

    // Got this far, let's reward the player with some Ore
    structure.OreMinePlanet()
    structure.Commit()

    _ = ctx.EventManager().EmitTypedEvent(&types.EventOreMine{&types.EventOreMineDetail{PlayerId: structure.GetOwnerId(), PrimaryAddress: structure.GetOwner().GetPrimaryAddress(), Amount: 1}})

	return &types.MsgStructOreMinerStatusResponse{Struct: structure.GetStruct()}, nil
}
