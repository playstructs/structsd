package keeper

import (
	"context"

    "fmt"

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

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructAttackResponse{}, permissionError
    }


    // Is the Struct & Owner online?
    readinessError := structure.ReadinessCheck()
    if (readinessError != nil) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructAttackResponse{}, readinessError
    }


    playerCharge := k.GetPlayerCharge(ctx, structure.GetOwnerId())
    if (playerCharge < structure.GetStructType().GetWeaponCharge(msg.WeaponSystem)) {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d for this attack, but player (%s) only had %d", structure.GetTypeId() , structure.GetStructType().GetWeaponCharge(msg.WeaponSystem), structure.GetOwnerId(), playerCharge)
    }

    for shot := uint64(0); shot < (structure.GetStructType().GetWeaponTargets(msg.WeaponSystem)); shot++ {
        // Load the Target Struct cache object
        targetStructure := k.GetStructCacheFromId(ctx, msg.TargetStructId[shot])

        /* Can the attacker attack? */

        // Check that the Structs are within attacking range of each other
        // This includes both a weapon<->ambit check, and a fleet<->planet
        targetingError := structure.CanAttack(&targetStructure, msg.WeaponSystem)
        if (targetingError != nil) {
            k.DischargePlayer(ctx, structure.GetOwnerId())
            return &types.MsgStructAttackResponse{}, targetingError
        }

        // TODO Event - Targeted

        if (targetStructure.CanEvade(&structure, msg.WeaponSystem)) {
            continue
        }

        attackBlocked := false

        // Check to make sure the attack is either counterable, blockable, or both. Otherwise skip this section
        if ((structure.GetStructType().GetWeaponBlockable(msg.WeaponSystem)) || (structure.GetStructType().GetWeaponCounterable(msg.WeaponSystem))) {

            // Check the Defenders
            defenderPlayer := targetStructure.GetOwner()
            defenders := targetStructure.GetDefenders()
            for _, defender := range defenders {
                fmt.Printf("Defender (%s) Protecting (%s) at Location (%s)", defender.DefendingStructId, defender.ProtectedStructId, defender.LocationId)

                defenderStructure := k.GetStructCacheFromId(ctx, defender.DefendingStructId)
                defenderStructure.ManualLoadOwner(defenderPlayer)

                defenderReadinessError := defenderStructure.ReadinessCheck()
                if (defenderReadinessError == nil) {
                    if (!attackBlocked && (structure.GetStructType().GetWeaponBlockable(msg.WeaponSystem))) {
                        attackBlocked = defenderStructure.AttemptBlock(&structure, msg.WeaponSystem, &targetStructure)
                    }

                }

                if (structure.GetStructType().GetWeaponCounterable(msg.WeaponSystem)) {
                    counterErrors := defenderStructure.CanCounterAttack(&structure)
                    if (counterErrors == nil) {
                        structure.TakeCounterAttackDamage(&defenderStructure)
                    }
                }

                defenderStructure.Commit()
            }
        }

        if (structure.GetStructType().GetWeaponCounterable(msg.WeaponSystem)) {
            counterErrors := targetStructure.CanCounterAttack(&structure)
            if (counterErrors == nil) {
                structure.TakeCounterAttackDamage(&targetStructure)
            }
        }

        targetStructure.Commit()

    }

    // Recoil Damage
    structure.TakeRecoilDamage(msg.WeaponSystem)
    structure.Commit()

    k.DischargePlayer(ctx, structure.GetOwnerId())

	return &types.MsgStructAttackResponse{}, nil
}
