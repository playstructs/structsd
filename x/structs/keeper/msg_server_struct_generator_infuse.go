package keeper

import (
	"context"

	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"
)

func (k msgServer) StructGeneratorInfuse(goCtx context.Context, msg *types.MsgStructGeneratorInfuse) (*types.MsgStructGeneratorStatusResponse, error) {
    emptyResponse := &types.MsgStructGeneratorStatusResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	callingPlayer, _ := cc.GetPlayerByAddress(msg.Creator)
	if callingPlayer.CheckPlayer() != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "struct_generator_infuse")
	}

	permissionErr := callingPlayer.CanInfuseTokensBy(callingPlayer)
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}

	structure := cc.GetStruct(msg.StructId)
	if structure.CheckStruct() != nil {
		return emptyResponse, types.NewObjectNotFoundError("struct", msg.StructId)
	}

	// Is the Struct online?
	if structure.IsOnline() {
		return emptyResponse, types.NewStructStateError(msg.StructId, "offline", "online", "generator_infuse")
	}

	if structure.GetStructType().PowerGeneration == types.TechPowerGeneration_noPowerGeneration {
		return emptyResponse, types.NewStructCapabilityError(msg.StructId, "generation")
	}

	if structure.GetPlanet().IsComplete() {
		return emptyResponse, types.NewPlanetStateError(structure.GetLocationId(), "complete", "generator_infuse")
	}

	infusionAmount, parseError := sdk.ParseCoinsNormalized(msg.InfuseAmount)
	if parseError != nil {
		return emptyResponse, types.NewFuelInfuseError(msg.StructId, msg.InfuseAmount, "invalid_amount")
	}

	if len(infusionAmount) < 1 {
		return emptyResponse, types.NewFuelInfuseError(msg.StructId, msg.InfuseAmount, "invalid_amount")
	}

	if infusionAmount[0].Denom == "ualpha" {
		// All good
	} else if infusionAmount[0].Denom == "alpha" {
		alphaUnitConversionInt := math.NewIntFromUint64(uint64(1000000))
		infusionAmount[0].Amount = infusionAmount[0].Amount.Mul(alphaUnitConversionInt)
		infusionAmount[0].Denom = "ualpha"
	} else {
		return emptyResponse, types.NewFuelInfuseError(msg.StructId, msg.InfuseAmount, "invalid_denom").WithDenom(infusionAmount[0].Denom)
	}

	// Transfer the refined Alpha from the player
	playerAcc, _ := sdk.AccAddressFromBech32(callingPlayer.GetPrimaryAddress())
	sendError := k.bankKeeper.SendCoinsFromAccountToModule(ctx, playerAcc, types.ModuleName, infusionAmount)

	if sendError != nil {
		return emptyResponse, types.NewFuelInfuseError(msg.StructId, msg.InfuseAmount, "transfer_failed").WithDetails(sendError.Error())
	}
	k.bankKeeper.BurnCoins(ctx, types.ModuleName, infusionAmount)

	infusion := cc.UpsertInfusion(types.ObjectType_struct, structure.GetStructId(), callingPlayer.GetPrimaryAddress(), callingPlayer.GetPlayerId())

	infusion.SetRatio(structure.GetStructType().GeneratingRate)
	infusion.SetCommission(math.LegacyZeroDec())
	infusion.AddFuel(infusionAmount[0].Amount.Uint64())

	_ = ctx.EventManager().EmitTypedEvent(&types.EventAlphaInfuse{&types.EventAlphaInfuseDetail{PlayerId: callingPlayer.GetPlayerId(), PrimaryAddress: callingPlayer.GetPrimaryAddress(), Amount: infusionAmount[0].Amount.Uint64()}})

	cc.CommitAll()
	return &types.MsgStructGeneratorStatusResponse{}, nil
}
