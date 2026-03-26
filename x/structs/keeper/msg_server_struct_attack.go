package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

func (k msgServer) StructAttack(goCtx context.Context, msg *types.MsgStructAttack) (*types.MsgStructAttackResponse, error) {
    emptyResponse := &types.MsgStructAttackResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }

	structure := cc.GetStruct(msg.OperatingStructId)

	k.logger.Info("Attack Action", "structId", msg.OperatingStructId)
	// Check to see if the caller has permissions to proceed
	permissionError := structure.CanBePlayedBy(callingPlayer)
	if permissionError != nil {
		return emptyResponse, permissionError
	}

	// Is the Struct & Owner online?
	readinessError := structure.ReadinessCheck()
	if readinessError != nil {
		return emptyResponse, readinessError
	}

	if !structure.IsCommandable() {
		return emptyResponse, types.NewFleetCommandError(structure.GetFleet().GetFleetId(), "command_offline").WithStructId(structure.GetStructId())
	}

	weaponSystem, weaponSystemExists := types.TechWeaponSystem_enum[msg.WeaponSystem]
	if !weaponSystemExists {
		return emptyResponse, types.NewParameterValidationError("weapon_system", 0, "invalid")
	}

	weaponSystemError := structure.GetStructType().VerifyWeaponSystem(weaponSystem)
	if weaponSystemError != nil {
		return emptyResponse, weaponSystemError
	}

	if structure.GetOwner().GetCharge() < structure.GetStructType().GetWeaponCharge(weaponSystem) {
		return emptyResponse, types.NewInsufficientChargeError(structure.GetOwnerId(), structure.GetStructType().GetWeaponCharge(weaponSystem), structure.GetOwner().GetCharge(), "attack").WithStructType(structure.GetTypeId())
	}

	// Jump out of Stealth Mode for the attack
	structure.StatusRemoveHidden()

	if uint64(len(msg.TargetStructId)) != structure.GetStructType().GetWeaponTargets(weaponSystem) {
		return emptyResponse, types.NewCombatTargetingError(structure.GetStructId(), "", msg.WeaponSystem, "incomplete_targeting")
	}

	ac := NewAttackContext(cc, structure, weaponSystem)

	// One iteration per weapon target (multi-target weapons); per-target projectile count is GetWeaponShots.
	for targetIdx := uint64(0); targetIdx < structure.GetStructType().GetWeaponTargets(weaponSystem); targetIdx++ {
		k.logger.Info("Attack Action", "structId", msg.OperatingStructId, "targetIdx", targetIdx, "weaponTargets", structure.GetStructType().GetWeaponTargets(weaponSystem), "target", msg.TargetStructId[targetIdx])

		targetStructure := cc.GetStruct(msg.TargetStructId[targetIdx])
		if !targetStructure.LoadStruct() {
			return emptyResponse, types.NewObjectNotFoundError("struct", msg.TargetStructId[targetIdx])
		}

		ac.BeginShot(targetStructure)

		targetingError := ac.ValidateTarget()
		if targetingError != nil {
			return emptyResponse, targetingError
		}

		k.logger.Info("Struct Targetable", "target", msg.TargetStructId[targetIdx])

		evaded := ac.ResolveEvasion()

		ac.ResolveDefenders(evaded)
		if !evaded {
		    ac.ResolveAttackDamage()
		}
		ac.ResolveTargetCounter()
		ac.EndShot()

		// If the attacker was destroyed during this target volley (e.g. by counter-attacks),
		// stop processing further targets immediately.
		if structure.IsDestroyed() {
			k.logger.Info("Attacker destroyed during combat, ending attack early", "structId", msg.OperatingStructId)
			break
		}
	}

	ac.ResolveRecoil()
	ac.ResolvePlanetaryDefense()
	ac.Finalize(ctx)

	structure.GetOwner().Discharge()

	if ctx.ExecMode() == sdk.ExecModeCheck {
		//ctx.GasMeter().RefundGas(ctx.GasMeter().GasConsumed(), "Walkin it back")
		ctx.GasMeter().ConsumeGas(uint64(200000), "Messin' with the estimator")
	}
	k.logger.Info("Attack Transaction Gas", "gasMeter", ctx.GasMeter().String(), "execMode", ctx.ExecMode())

	cc.CommitAll()
	return &types.MsgStructAttackResponse{}, nil
}
