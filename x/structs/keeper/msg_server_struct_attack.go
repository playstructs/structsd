package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"

	"structs/x/structs/types"
)


func (k msgServer) StructAttack(goCtx context.Context, msg *types.MsgStructAttack) (*types.MsgStructAttackResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    structure := k.GetStructCacheFromId(ctx, msg.OperatingStructId)

    k.logger.Info("Attack Action", "structId", msg.OperatingStructId)
    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructAttackResponse{}, permissionError
    }

    if structure.GetOwner().IsHalted() {
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrPlayerHalted, "Struct (%s) cannot perform actions while Player (%s) is Halted", msg.OperatingStructId, structure.GetOwnerId())
    }

    // Is the Struct & Owner online?
    readinessError := structure.ReadinessCheck()
    if (readinessError != nil) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructAttackResponse{}, readinessError
    }

    if !structure.IsCommandable() {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Commanding a Fleet Struct (%s) requires a Command Struct be Online", structure.GetStructId())
    }

    playerCharge := k.GetPlayerCharge(ctx, structure.GetOwnerId())
    if (playerCharge < structure.GetStructType().GetWeaponCharge(types.TechWeaponSystem_enum[msg.WeaponSystem])) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d for this attack, but player (%s) only had %d", structure.GetTypeId() , structure.GetStructType().GetWeaponCharge(types.TechWeaponSystem_enum[msg.WeaponSystem]), structure.GetOwnerId(), playerCharge)
    }

    // Jump out of Stealth Mode for the attack
    structure.StatusRemoveHidden()

    var eventAttackDetail *types.EventAttackDetail
    eventAttackDetail = structure.GetEventAttackDetail()
    eventAttackDetail.SetBaseDetails(structure.GetOwnerId(), structure.GetStructId(), structure.GetTypeId(), structure.GetLocationType(), structure.GetLocationId(), structure.GetOperatingAmbit(), structure.GetSlot(), types.TechWeaponSystem_enum[msg.WeaponSystem], structure.GetStructType().GetWeaponControl(types.TechWeaponSystem_enum[msg.WeaponSystem]), structure.GetStructType().GetWeapon(types.TechWeaponSystem_enum[msg.WeaponSystem]))

    structure.ManualLoadEventAttackDetail(eventAttackDetail)

    var targetWasPlanetary bool
    var targetWasOnPlanet *PlanetCache

    if uint64(len(msg.TargetStructId)) != structure.GetStructType().GetWeaponTargets(types.TechWeaponSystem_enum[msg.WeaponSystem]) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrStructAction, "Attack Targeting Incomplete")
    }

    // Begin taking shots. Most weapons only use a single shot but some perform multiple.
    for shot := uint64(0); shot < (structure.GetStructType().GetWeaponTargets(types.TechWeaponSystem_enum[msg.WeaponSystem])); shot++ {
        k.logger.Info("Attack Action", "structId", msg.OperatingStructId, "shot", shot, "shots", structure.GetStructType().GetWeaponTargets(types.TechWeaponSystem_enum[msg.WeaponSystem]), "target", msg.TargetStructId[shot] )
        // Load the Target Struct cache object
        targetStructure := k.GetStructCacheFromId(ctx, msg.TargetStructId[shot])

        targetStructure.ManualLoadEventAttackDetail(eventAttackDetail)
        eventAttackDetail.SetTargetPlayerId(targetStructure.GetOwnerId())

        eventAttackShotDetail := targetStructure.GetEventAttackShotDetail()
        structure.ManualLoadEventAttackShotDetail(eventAttackShotDetail)
        structure.GetEventAttackShotDetail().SetTargetDetails(targetStructure.GetStructId(), targetStructure.GetTypeId(), targetStructure.GetLocationType(), targetStructure.GetLocationId(), targetStructure.GetOperatingAmbit(), targetStructure.GetSlot())

        /* Can the attacker attack? */
        // Check that the Structs are within attacking range of each other
        // This includes both a weapon<->ambit check, and a fleet<->planet
        targetingError := structure.CanAttack(&targetStructure, types.TechWeaponSystem_enum[msg.WeaponSystem])
        if (targetingError != nil) {
            k.DischargePlayer(ctx, structure.GetOwnerId())
            return &types.MsgStructAttackResponse{}, targetingError
        }

        k.logger.Info("Struct Targetable", "target", msg.TargetStructId[shot])

        if (targetStructure.CanEvade(&structure, types.TechWeaponSystem_enum[msg.WeaponSystem])) {
            k.logger.Info("Struct Evaded", "target", msg.TargetStructId[shot])
            structure.GetEventAttackDetail().AppendShot(targetStructure.FlushEventAttackShotDetail())
            continue
        }

        attackBlocked := false

        // Check to make sure the attack is either counterable, blockable, or both. Otherwise skip this section
        k.logger.Info("Struct Attacker Status", "structId", structure.GetStructId(), "blockable", (structure.GetStructType().GetWeaponBlockable(types.TechWeaponSystem_enum[msg.WeaponSystem])), "counterable",(structure.GetStructType().GetWeaponCounterable(types.TechWeaponSystem_enum[msg.WeaponSystem])))
        if ((structure.GetStructType().GetWeaponBlockable(types.TechWeaponSystem_enum[msg.WeaponSystem])) || (structure.GetStructType().GetWeaponCounterable(types.TechWeaponSystem_enum[msg.WeaponSystem]))) {

            // Check the Defenders
            defenderPlayer := targetStructure.GetOwner()
            defenders := targetStructure.GetDefenders()
            for _, defender := range defenders {
                k.logger.Info("Defender at Location", "defender", defender.GetStructId(), "locationId", defender.GetLocationId())

                defender.Defender = true
                defender.ManualLoadOwner(defenderPlayer)
                defender.ManualLoadEventAttackDetail(eventAttackDetail)
                defender.ManualLoadEventAttackShotDetail(eventAttackShotDetail)

                defenderReadinessError := defender.ReadinessCheck()
                if (defenderReadinessError == nil) {
                    k.logger.Info("Defender seems ready to defend")
                    if (!attackBlocked && (structure.GetStructType().GetWeaponBlockable(types.TechWeaponSystem_enum[msg.WeaponSystem]))) {
                        k.logger.Info("Defender to attempt a block!")
                        attackBlocked = defender.AttemptBlock(&structure, types.TechWeaponSystem_enum[msg.WeaponSystem], &targetStructure)
                    }

                }

                if (structure.GetStructType().GetWeaponCounterable(types.TechWeaponSystem_enum[msg.WeaponSystem])) {
                    k.logger.Info("Defender trying to counter!.. ")
                    counterErrors := defender.CanCounterAttack(&structure)
                    if (counterErrors == nil) {
                        k.logger.Info("Defender counter-attacking!")
                        structure.TakeCounterAttackDamage(defender)
                    }
                }

                defender.Commit()
            }
        }

        // Fun story, I'd actually forgotten this code block after writing all the other function
        // Turns out, my Struct wasn't attacking because I forgot the part of Attack that attacks.
        if (!attackBlocked && structure.IsOnline()) {
            k.logger.Info("Moving forward with the attack", "target", msg.TargetStructId[shot])
            targetStructure.TakeAttackDamage(&structure, types.TechWeaponSystem_enum[msg.WeaponSystem])
        } else {
            k.logger.Info("Attack against target was blocked", "target", msg.TargetStructId[shot])
        }


        if (structure.GetStructType().GetWeaponCounterable(types.TechWeaponSystem_enum[msg.WeaponSystem])) {
            k.logger.Info("Target trying to Counter now!")
            counterErrors := targetStructure.CanCounterAttack(&structure)
            if (counterErrors == nil) {
                k.logger.Info("Target Countering!")
                structure.TakeCounterAttackDamage(&targetStructure)
            }
        }

        structure.GetEventAttackDetail().AppendShot(targetStructure.FlushEventAttackShotDetail())

        if (targetStructure.GetStructType().GetCategory() == types.ObjectType_planet) {
            targetWasPlanetary = true
            targetWasOnPlanet = targetStructure.GetPlanet()
        }

        // Possibly over committing if the same target is hit multiple times.
        targetStructure.Commit()
    }

    // Recoil Damage
    structure.TakeRecoilDamage(types.TechWeaponSystem_enum[msg.WeaponSystem])

    // Check for Planetary Damage, namely Defense Cannons
    if (targetWasPlanetary) {
        targetWasOnPlanet.AttemptDefenseCannon(&structure)
    }

    _ = ctx.EventManager().EmitTypedEvent(&types.EventAttack{EventAttackDetail: eventAttackDetail})

    structure.Commit()

    k.DischargePlayer(ctx, structure.GetOwnerId())

    if (ctx.ExecMode() == sdk.ExecModeCheck) {
        //ctx.GasMeter().RefundGas(ctx.GasMeter().GasConsumed(), "Walkin it back")
        ctx.GasMeter().ConsumeGas(uint64(200000), "Messin' with the estimator")
    }
    k.logger.Info("Attack Transaction Gas", "gasMeter", ctx.GasMeter().String(), "execMode", ctx.ExecMode())

	return &types.MsgStructAttackResponse{}, nil
}
