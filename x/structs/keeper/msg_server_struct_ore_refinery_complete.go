package keeper

import (
	"context"
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructOreRefineryComplete(goCtx context.Context, msg *types.MsgStructOreRefineryComplete) (*types.MsgStructOreRefineryStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)


	structure := k.GetStructCacheFromId(ctx, msg.StructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructOreRefineryStatusResponse{}, permissionError
    }

    // Is the Struct & Owner online?
    readinessError := structure.ReadinessCheck()
    if (readinessError != nil) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructOreRefineryStatusResponse{}, readinessError
    }


    playerCharge := k.GetPlayerCharge(ctx, structure.GetOwnerId())
    if (playerCharge < structure.GetStructType().GetOreRefiningCharge()) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructOreRefineryStatusResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d for this refinement, but player (%s) only had %d", structure.GetTypeId() , structure.GetStructType().GetOreRefiningCharge(), structure.GetOwnerId(), playerCharge)
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
       return &types.MsgStructOreRefineryStatusResponse{}, sdkerrors.Wrapf(types.ErrStructRefine, "Work failure for input (%s) when trying to refine on Struct %s", hashInput, structure.StructId)
    }

    structure.OreRefine()

    k.DischargePlayer(ctx, structure.GetOwnerId())

    structure.Commit()
    structure.GetOwner().Commit()
    structure.GetPlanet().Commit()

    _ = ctx.EventManager().EmitTypedEvent(&types.EventAlphaRefine{&types.EventAlphaRefineDetail{PlayerId: structure.GetOwnerId(), PrimaryAddress: structure.GetOwner().GetPrimaryAddress(), Amount: 1}})

	return &types.MsgStructOreRefineryStatusResponse{Struct: structure.GetStruct()}, nil
}
