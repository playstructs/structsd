package keeper

import (
	"context"


	sdk "github.com/cosmos/cosmos-sdk/types"
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
        return &types.MsgStructGeneratorStatusResponse{}, types.NewPlayerRequiredError(msg.Creator, "struct_generator_infuse")
    }
    callingPlayerId := GetObjectID(types.ObjectType_player, callingPlayerIndex)
    callingPlayer, _ := k.GetPlayer(ctx, callingPlayerId)

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Assets permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets)) {
        return &types.MsgStructGeneratorStatusResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionAssets), "assets")
    }

    structStatusAttributeId := GetStructAttributeIDByObjectId(types.StructAttributeType_status, msg.StructId)

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructGeneratorStatusResponse{}, types.NewObjectNotFoundError("struct", msg.StructId)
    }

    // Is the Struct online?
    if (k.StructAttributeFlagHasOneOf(ctx, structStatusAttributeId, uint64(types.StructStateOnline))) {
        k.DischargePlayer(ctx, callingPlayerId)
        return &types.MsgStructGeneratorStatusResponse{}, types.NewStructStateError(msg.StructId, "offline", "online", "generator_infuse")
    }

    // Load Struct Type
    structType, structTypeFound := k.GetStructType(ctx, structure.Type)
    if (!structTypeFound) {
        return &types.MsgStructGeneratorStatusResponse{}, types.NewObjectNotFoundError("struct_type", "").WithIndex(structure.Type)
    }


    if (structType.PowerGeneration == types.TechPowerGeneration_noPowerGeneration) {
        k.DischargePlayer(ctx, callingPlayerId)
        return &types.MsgStructGeneratorStatusResponse{}, types.NewStructCapabilityError(msg.StructId, "generation")
    }

    // FIX FIX FIX  FIX
    // TODO
    // Change all of these to do a more deeper check
    // Check for Fleet location, etc.
    // repeat everywhere
    // FIX FIX FIX  FIX
    planet, planetFound := k.GetPlanet(ctx, structure.LocationId)
    if (!planetFound) {
        return &types.MsgStructGeneratorStatusResponse{}, types.NewObjectNotFoundError("planet", structure.LocationId)
    }

    if (planet.Status == types.PlanetStatus_complete) {
        k.DischargePlayer(ctx, callingPlayerId)
        return &types.MsgStructGeneratorStatusResponse{}, types.NewPlanetStateError(structure.LocationId, "complete", "generator_infuse")
    }

    infusionAmount, parseError := sdk.ParseCoinsNormalized(msg.InfuseAmount)
    if (parseError != nil ){
        return &types.MsgStructGeneratorStatusResponse{}, types.NewFuelInfuseError(msg.StructId, msg.InfuseAmount, "invalid_amount")
    }

    if len(infusionAmount) < 1 {
        return &types.MsgStructGeneratorStatusResponse{}, types.NewFuelInfuseError(msg.StructId, msg.InfuseAmount, "invalid_amount")
    }

    if (infusionAmount[0].Denom == "ualpha") {
        // All good
    } else if (infusionAmount[0].Denom == "alpha") {
        alphaUnitConversionInt := math.NewIntFromUint64(uint64(1000000))
        infusionAmount[0].Amount = infusionAmount[0].Amount.Mul(alphaUnitConversionInt)
        infusionAmount[0].Denom  = "ualpha"
    } else {
        return &types.MsgStructGeneratorStatusResponse{}, types.NewFuelInfuseError(msg.StructId, msg.InfuseAmount, "invalid_denom").WithDenom(infusionAmount[0].Denom)
    }

    // Transfer the refined Alpha from the player
    playerAcc, _ := sdk.AccAddressFromBech32(callingPlayer.PrimaryAddress)
    sendError := k.bankKeeper.SendCoinsFromAccountToModule(ctx, playerAcc, types.ModuleName, infusionAmount)

    if (sendError != nil){
        k.DischargePlayer(ctx, callingPlayerId)
        return &types.MsgStructGeneratorStatusResponse{}, types.NewFuelInfuseError(msg.StructId, msg.InfuseAmount, "transfer_failed").WithDetails(sendError.Error())
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
