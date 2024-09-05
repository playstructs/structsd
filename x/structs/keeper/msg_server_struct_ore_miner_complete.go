package keeper

import (
	"context"
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructOreMinerComplete(goCtx context.Context, msg *types.MsgStructOreMinerComplete) (*types.MsgStructOreMinerStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	structure := k.GetStructCacheFromId(ctx, msg.StructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructOreMinerStatusResponse{}, permissionError
    }

    // Is the Struct & Owner online?
    readinessError := structure.ReadinessCheck()
    if (readinessError != nil) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructOreMinerStatusResponse{}, readinessError
    }


    playerCharge := k.GetPlayerCharge(ctx, structure.GetOwnerId())
    if (playerCharge < structure.GetStructType().GetOreMiningCharge()) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructOreMinerStatusResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d for this attack, but player (%s) only had %d", structure.GetTypeId() , structure.GetStructType().GetOreMiningCharge(), structure.GetOwnerId(), playerCharge)
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
       return &types.MsgStructOreMinerStatusResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "Work failure for input (%s) when trying to mine on Struct %s", hashInput, structure.StructId)
    }

    // Got this far, let's reward the player with some Ore
    structure.OreMinePlanet()

    k.DischargePlayer(ctx, structure.GetOwnerId())

    structure.Commit()
    structure.GetOwner().Commit()
    structure.GetPlanet().Commit()

	return &types.MsgStructOreMinerStatusResponse{Struct: structure.GetStruct()}, nil
}
