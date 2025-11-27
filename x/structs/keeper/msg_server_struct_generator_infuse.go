package keeper

import (
	"context"


	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"

	"cosmossdk.io/math"
)

func (k msgServer) StructGeneratorInfuse(goCtx context.Context, msg *types.MsgStructGeneratorInfuse) (*types.MsgStructGeneratorStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (callingPlayerIndex == 0) {
        return &types.MsgStructGeneratorStatusResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build actions requires Player account but none associated with %s", msg.Creator)
    }
    callingPlayerId := GetObjectID(types.ObjectType_player, callingPlayerIndex)
    callingPlayer, _ := k.GetPlayer(ctx, callingPlayerId)

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Assets permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets)) {
        return &types.MsgStructGeneratorStatusResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no assets permissions ", msg.Creator)
    }

    structStatusAttributeId := GetStructAttributeIDByObjectId(types.StructAttributeType_status, msg.StructId)

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructGeneratorStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct (%s) not found", msg.StructId)
    }

    // Is the Struct online?
    if (k.StructAttributeFlagHasOneOf(ctx, structStatusAttributeId, uint64(types.StructStateOnline))) {
        k.DischargePlayer(ctx, callingPlayerId)
        return &types.MsgStructGeneratorStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) is offline. Activate it", msg.StructId)
    }

    // Load Struct Type
    structType, structTypeFound := k.GetStructType(ctx, structure.Type)
    if (!structTypeFound) {
        return &types.MsgStructGeneratorStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct Type (%d) was not found. Building a Struct with schematics might be tough", structure.Type)
    }


    if (structType.PowerGeneration == types.TechPowerGeneration_noPowerGeneration) {
        k.DischargePlayer(ctx, callingPlayerId)
        return &types.MsgStructGeneratorStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) has no generation systems", msg.StructId)
    }

    // FIX FIX FIX  FIX
    // TODO
    // Change all of these to do a more deeper check
    // Check for Fleet location, etc.
    // repeat everywhere
    // FIX FIX FIX  FIX
    planet, planetFound := k.GetPlanet(ctx, structure.LocationId)
    if (!planetFound) {
        return &types.MsgStructGeneratorStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Planet (%s) was not found, which is actually a pretty big problem. Please tell an adult", structure.LocationId)
    }

    if (planet.Status == types.PlanetStatus_complete) {
        k.DischargePlayer(ctx, callingPlayerId)
        return &types.MsgStructGeneratorStatusResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "Planet (%s) is already complete. Move on bud, no work to be done here", structure.LocationId)
    }

    infusionAmount, parseError := sdk.ParseCoinsNormalized(msg.InfuseAmount)
    if (parseError != nil ){
        return &types.MsgStructGeneratorStatusResponse{}, sdkerrors.Wrapf(types.ErrStructInfuse, "Infuse amount (%s) is invalid", msg.InfuseAmount)
    }

    if (infusionAmount[0].Denom == "ualpha") {
        // All good
    } else if (infusionAmount[0].Denom == "alpha") {
        alphaUnitConversionInt := math.NewIntFromUint64(uint64(1000000))
        infusionAmount[0].Amount = infusionAmount[0].Amount.Mul(alphaUnitConversionInt)
        infusionAmount[0].Denom  = "ualpha"
    } else {
        return &types.MsgStructGeneratorStatusResponse{}, sdkerrors.Wrapf(types.ErrStructInfuse, "Infuse amount (%s) is invalid, %s is not a fuel", msg.InfuseAmount, infusionAmount[0].Denom)
    }

    // Transfer the refined Alpha from the player
    playerAcc, _ := sdk.AccAddressFromBech32(callingPlayer.PrimaryAddress)
    sendError := k.bankKeeper.SendCoinsFromAccountToModule(ctx, playerAcc, types.ModuleName, infusionAmount)

    if (sendError != nil){
        k.DischargePlayer(ctx, callingPlayerId)
        return &types.MsgStructGeneratorStatusResponse{}, sdkerrors.Wrapf(types.ErrStructInfuse, "Infuse failed %s", sendError )
    }
    k.bankKeeper.BurnCoins(ctx, types.ModuleName, infusionAmount)

    infusion := k.GetInfusionCache(ctx, types.ObjectType_struct, structure.Id, callingPlayer.PrimaryAddress)

    infusion.SetRatio(structType.GeneratingRate)
    infusion.SetCommission(math.LegacyZeroDec())
    infusion.AddFuel(infusionAmount[0].Amount.Uint64())
    infusion.Commit()

    _ = ctx.EventManager().EmitTypedEvent(&types.EventAlphaInfuse{&types.EventAlphaInfuseDetail{PlayerId: callingPlayer.Id, PrimaryAddress: callingPlayer.PrimaryAddress, Amount: infusionAmount[0].Amount.Uint64()}})

	return &types.MsgStructGeneratorStatusResponse{}, nil
}
