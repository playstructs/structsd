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

	callingPlayer, playerErr := cc.GetSigningPlayer(msg.Creator)
	if playerErr != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "struct_generator_infuse")
	}
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

	// A destroyed struct is offline, so the check below would reject it anyway.
	// Say so explicitly, because this handler moves the player's coins and must
	// never send them to a generator that is queued for deletion.
	if structure.IsDestroyed() {
		return emptyResponse, types.NewStructStateError(msg.StructId, "destroyed", "active", "generator_infuse")
	}

	// Is the Struct online?
	if !structure.IsOnline() {
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

	/* Exactly one coin, and only then look at what it is.
	 *
	 * msg.InfuseAmount is a free-form string and ParseCoinsNormalized happily
	 * accepts a comma-separated list, which it then sorts by denom. Checking
	 * infusionAmount[0] and moving the whole slice therefore validated one coin
	 * and burned all of them: "1ualpha,1000000uguild.1-2" sorts ualpha first,
	 * passes, and destroys the guild tokens. Only the first coin was ever
	 * credited as fuel, so the rest were burned for nothing.
	 *
	 * That crossed an authorization boundary as well as an accounting one. The
	 * only permission this handler asks for is PermTokenInfuse, and it spends the
	 * player's primary account - so a delegate scoped to infusion alone could
	 * destroy asset classes that PermGuildTokenBurn exists to protect.
	 */
	if len(infusionAmount) != 1 {
		return emptyResponse, types.NewFuelInfuseError(msg.StructId, msg.InfuseAmount, "invalid_amount")
	}

	var fuelAmount math.Int
	switch infusionAmount[0].Denom {
	case "ualpha":
		fuelAmount = infusionAmount[0].Amount
	case "alpha":
		fuelAmount = infusionAmount[0].Amount.Mul(math.NewIntFromUint64(uint64(1000000)))
	default:
		return emptyResponse, types.NewFuelInfuseError(msg.StructId, msg.InfuseAmount, "invalid_denom").WithDenom(infusionAmount[0].Denom)
	}

	// Rebuilt rather than reused: what moves has to be what was checked, not the
	// parsed value the message asked for.
	fuelCoins := sdk.NewCoins(sdk.NewCoin("ualpha", fuelAmount))

	// Transfer the refined Alpha from the player
	playerAcc, _ := sdk.AccAddressFromBech32(callingPlayer.GetPrimaryAddress())
	sendError := k.bankKeeper.SendCoinsFromAccountToModule(ctx, playerAcc, types.ModuleName, fuelCoins)

	if sendError != nil {
		return emptyResponse, types.NewFuelInfuseError(msg.StructId, msg.InfuseAmount, "transfer_failed").WithDetails(sendError.Error())
	}

	// Propagated: the coins are in the module account by now, so a silent burn
	// failure would leave them there with nothing crediting them back.
	if burnError := k.bankKeeper.BurnCoins(ctx, types.ModuleName, fuelCoins); burnError != nil {
		return emptyResponse, types.NewFuelInfuseError(msg.StructId, msg.InfuseAmount, "burn_failed").WithDetails(burnError.Error())
	}

	infusion := cc.UpsertInfusion(types.ObjectType_struct, structure.GetStructId(), callingPlayer.GetPrimaryAddress(), callingPlayer.GetPlayerId())

	infusion.SetRatio(structure.GetStructType().GeneratingRate)
	infusion.SetCommission(math.LegacyZeroDec())
	infusion.AddFuel(fuelAmount.Uint64())

	_ = ctx.EventManager().EmitTypedEvent(&types.EventAlphaInfuse{&types.EventAlphaInfuseDetail{PlayerId: callingPlayer.GetPlayerId(), PrimaryAddress: callingPlayer.GetPrimaryAddress(), Amount: fuelAmount.Uint64()}})

	cc.CommitAll()
	return &types.MsgStructGeneratorStatusResponse{}, nil
}
