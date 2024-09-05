package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	//"fmt"
)

func (k msgServer) StructActivate(goCtx context.Context, msg *types.MsgStructActivate) (*types.MsgStructStatusResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // load struct
    structure := k.GetStructCacheFromId(ctx, msg.StructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    // Check Activation Readiness
        // Check Struct is Built
        // Check Struct is Offline
        // Check Player is Online
        // Check Player capacity
    readinessError := structure.ActivationReadinessCheck()
    if (readinessError != nil) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructStatusResponse{}, readinessError
    }


    playerCharge := k.GetPlayerCharge(ctx, structure.GetOwnerId())
    if (playerCharge < structure.GetStructType().GetActivateCharge()) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d for this mining operation, but player (%s) only had %d", structure.GetTypeId() , structure.GetStructType().GetOreMiningCharge(), structure.GetOwnerId(), playerCharge)
    }




    // go online
        // update player capacity
        // turn on systems

    // commit struct
    // commit struct player



    // Add to the players struct load
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, sudoPlayer.Id), structType.PassiveDraw)

    // Turn on the mining systems
    if (structType.PlanetaryMining != types.TechPlanetaryMining_noPlanetaryMining) {
        k.SetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, structure.Id), uint64(ctx.BlockHeight()))
    }

    // Turn on the refinery
    if (structType.PlanetaryRefinery != types.TechPlanetaryRefineries_noPlanetaryRefinery) {
        k.SetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, structure.Id), uint64(ctx.BlockHeight()))
    }

    // Set the struct status flag to include built
    k.SetStructAttributeFlagAdd(ctx, structStatusAttributeId, uint64(types.StructStateOnline))


	return &types.MsgStructStatusResponse{Struct: structure}, nil
}
