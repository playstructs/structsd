package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

func (k msgServer) StructAttack(goCtx context.Context, msg *types.MsgStructAttack) (*types.MsgStructAttackResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	structure := cc.GetStruct(msg.OperatingStructId)

	k.logger.Info("Attack Action", "structId", msg.OperatingStructId)
	// Check to see if the caller has permissions to proceed
	permissionError := structure.CanBePlayedBy(msg.Creator)
	if permissionError != nil {
		return &types.MsgStructAttackResponse{}, permissionError
	}

	if structure.GetOwner().IsHalted() {
		return &types.MsgStructAttackResponse{}, types.NewPlayerHaltedError(structure.GetOwnerId(), "struct_attack").WithStruct(msg.OperatingStructId)
	}

	// Is the Struct & Owner online?
	readinessError := structure.ReadinessCheck()
	if readinessError != nil {
		return &types.MsgStructAttackResponse{}, readinessError
	}

	if !structure.IsCommandable() {
		return &types.MsgStructAttackResponse{}, types.NewFleetCommandError(structure.GetFleet().GetFleetId(), "command_offline").WithStructId(structure.GetStructId())
	}

	weaponSystem, weaponSystemExists := types.TechWeaponSystem_enum[msg.WeaponSystem]
	if !weaponSystemExists {
		return &types.MsgStructAttackResponse{}, types.NewParameterValidationError("weapon_system", 0, "invalid")
	}

	weaponSystemError := structure.GetStructType().VerifyWeaponSystem(weaponSystem)
	if weaponSystemError != nil {
		return &types.MsgStructAttackResponse{}, weaponSystemError
	}

	playerCharge := k.GetPlayerCharge(ctx, structure.GetOwnerId())
	if playerCharge < structure.GetStructType().GetWeaponCharge(weaponSystem) {
		return &types.MsgStructAttackResponse{}, types.NewInsufficientChargeError(structure.GetOwnerId(), structure.GetStructType().GetWeaponCharge(weaponSystem), playerCharge, "attack").WithStructType(structure.GetTypeId())
	}

	// Jump out of Stealth Mode for the attack
	structure.StatusRemoveHidden()

	var eventAttackDetail *types.EventAttackDetail
	eventAttackDetail = structure.GetEventAttackDetail()
	eventAttackDetail.SetBaseDetails(structure.GetOwnerId(), structure.GetStructId(), structure.GetTypeId(), structure.GetLocationType(), structure.GetLocationId(), structure.GetOperatingAmbit(), structure.GetSlot(), weaponSystem, structure.GetStructType().GetWeaponControl(weaponSystem), structure.GetStructType().GetWeapon(weaponSystem))

	structure.ManualLoadEventAttackDetail(eventAttackDetail)

	var targetWasPlanetary bool
	var targetWasOnPlanet *PlanetCache

	if uint64(len(msg.TargetStructId)) != structure.GetStructType().GetWeaponTargets(weaponSystem) {
		return &types.MsgStructAttackResponse{}, types.NewCombatTargetingError(structure.GetStructId(), "", msg.WeaponSystem, "incomplete_targeting")
	}

	// Begin taking shots. Most weapons only use a single shot but some perform multiple.
	for shot := uint64(0); shot < (structure.GetStructType().GetWeaponTargets(weaponSystem)); shot++ {
		k.logger.Info("Attack Action", "structId", msg.OperatingStructId, "shot", shot, "shots", structure.GetStructType().GetWeaponTargets(weaponSystem), "target", msg.TargetStructId[shot])

		targetStructure := cc.GetStruct(msg.TargetStructId[shot])
		if !targetStructure.LoadStruct() {
			return &types.MsgStructAttackResponse{}, types.NewObjectNotFoundError("struct", msg.TargetStructId[shot])
		}

		targetStructure.ManualLoadEventAttackDetail(eventAttackDetail)
		eventAttackDetail.SetTargetPlayerId(targetStructure.GetOwnerId())

		eventAttackShotDetail := targetStructure.GetEventAttackShotDetail()
		structure.ManualLoadEventAttackShotDetail(eventAttackShotDetail)
		structure.GetEventAttackShotDetail().SetTargetDetails(targetStructure.GetStructId(), targetStructure.GetTypeId(), targetStructure.GetLocationType(), targetStructure.GetLocationId(), targetStructure.GetOperatingAmbit(), targetStructure.GetSlot())

		// Initialize target health - will be updated by TakeAttackDamage if hit
		currentTargetHealth := targetStructure.GetHealth()
		eventAttackShotDetail.SetTargetHealthBefore(currentTargetHealth)
		eventAttackShotDetail.SetTargetHealthAfter(currentTargetHealth)

		/* Can the attacker attack? */
		// Check that the Structs are within attacking range of each other
		// This includes both a weapon<->ambit check, and a fleet<->planet
		targetingError := structure.CanAttack(targetStructure, weaponSystem)
		if targetingError != nil {
			return &types.MsgStructAttackResponse{}, targetingError
		}

		k.logger.Info("Struct Targetable", "target", msg.TargetStructId[shot])

		if targetStructure.CanEvade(structure, weaponSystem) {
			k.logger.Info("Struct Evaded", "target", msg.TargetStructId[shot])
			structure.GetEventAttackDetail().AppendShot(targetStructure.FlushEventAttackShotDetail())
			continue
		}

		attackBlocked := false

		// Check to make sure the attack is either counterable, blockable, or both. Otherwise skip this section
		k.logger.Info("Struct Attacker Status", "structId", structure.GetStructId(), "blockable", (structure.GetStructType().GetWeaponBlockable(weaponSystem)), "counterable", (structure.GetStructType().GetWeaponCounterable(weaponSystem)))
		if (structure.GetStructType().GetWeaponBlockable(weaponSystem)) || (structure.GetStructType().GetWeaponCounterable(weaponSystem)) {

			// Check the Defenders
			defenderPlayer := targetStructure.GetOwner()
			defenders := targetStructure.GetDefenders()
			for _, defender := range defenders {
				k.logger.Info("Defender at Location", "defender", defender.GetStructId(), "locationId", defender.GetLocationId())

				// Use CC for deduplication - if this struct was already loaded, reuse that instance
				defender = cc.GetStruct(defender.GetStructId())

				defender.Defender = true
				defender.ManualLoadOwner(defenderPlayer)
				defender.ManualLoadEventAttackDetail(eventAttackDetail)
				defender.ManualLoadEventAttackShotDetail(eventAttackShotDetail)

				defenderReadinessError := defender.ReadinessCheck()
				if defenderReadinessError == nil {
					k.logger.Info("Defender seems ready to defend")
					if !attackBlocked && (structure.GetStructType().GetWeaponBlockable(weaponSystem)) {
						k.logger.Info("Defender to attempt a block!")
						attackBlocked = defender.AttemptBlock(structure, weaponSystem, targetStructure)
					}
				}

				if structure.GetStructType().GetWeaponCounterable(weaponSystem) {
					k.logger.Info("Defender trying to counter!.. ")
					counterErrors := defender.CanCounterAttack(structure)
					if counterErrors == nil {
						k.logger.Info("Defender counter-attacking!")
						structure.TakeCounterAttackDamage(defender)
					}
				}
			}
		}

		// Fun story, I'd actually forgotten this code block after writing all the other function
		// Turns out, my Struct wasn't attacking because I forgot the part of Attack that attacks.
		if !attackBlocked && structure.IsOnline() {
			k.logger.Info("Moving forward with the attack", "target", msg.TargetStructId[shot])
			targetStructure.TakeAttackDamage(structure, weaponSystem)
		} else {
			k.logger.Info("Attack against target was blocked", "target", msg.TargetStructId[shot])
		}

		if structure.GetStructType().GetWeaponCounterable(weaponSystem) {
			k.logger.Info("Target trying to Counter now!")
			counterErrors := targetStructure.CanCounterAttack(structure)
			if counterErrors == nil {
				k.logger.Info("Target Countering!")
				structure.TakeCounterAttackDamage(targetStructure)
			}
		}

		structure.GetEventAttackDetail().AppendShot(targetStructure.FlushEventAttackShotDetail())

		if targetStructure.GetStructType().GetCategory() == types.ObjectType_planet {
			targetWasPlanetary = true
			targetWasOnPlanet = targetStructure.GetPlanet()
		}
	}

	// Recoil Damage
	structure.TakeRecoilDamage(weaponSystem)

	// Check for Planetary Damage, namely Defense Cannons
	if targetWasPlanetary {
		targetWasOnPlanet.AttemptDefenseCannon(structure)
	}

	// Set attacker's final health after all damage sources have resolved
	eventAttackDetail.SetAttackerHealthAfter(structure.GetHealth())

	_ = ctx.EventManager().EmitTypedEvent(&types.EventAttack{EventAttackDetail: eventAttackDetail})

	k.DischargePlayer(ctx, structure.GetOwnerId())

	if ctx.ExecMode() == sdk.ExecModeCheck {
		//ctx.GasMeter().RefundGas(ctx.GasMeter().GasConsumed(), "Walkin it back")
		ctx.GasMeter().ConsumeGas(uint64(200000), "Messin' with the estimator")
	}
	k.logger.Info("Attack Transaction Gas", "gasMeter", ctx.GasMeter().String(), "execMode", ctx.ExecMode())

	return &types.MsgStructAttackResponse{}, nil
}
