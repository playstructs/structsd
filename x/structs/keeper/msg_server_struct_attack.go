package keeper

import (
	"context"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"


    "github.com/nethruster/go-fraction"
	"structs/x/structs/types"
)


func (k msgServer) StructAttack(goCtx context.Context, msg *types.MsgStructAttack) (*types.MsgStructAttackResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := k.GetPlayerCacheFromAddress(ctx, msg.Creator)
    if (err != nil) {
        return &types.MsgStructAttackResponse{}, err
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionPlay)) {
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }


    structure := k.GetStructCacheFromId(ctx, msg.OperatingStructId)


    // Is the Struct online?
    if (!structure.IsOffline()) {
        k.DischargePlayer(ctx, structure.GetOwner())
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) is offline. Activate it", msg.OperatingStructId)
    }


    if (callingPlayer.PlayerId != structure.GetOwner()) {
        // Check permissions on Creator on Planet
        playerPermissionId := GetObjectPermissionIDBytes(structure.GetOwner(), callingPlayer.PlayerId)
        if (!k.PermissionHasOneOf(ctx, playerPermissionId, types.PermissionPlay)) {
            return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling account (%s) has no play permissions on target player (%s)", callingPlayer.PlayerId, structure.GetOwner())
        }
    }
    sudoPlayer, _ := k.GetPlayerCacheFromId(ctx, structure.GetOwner())
    if (sudoPlayer.IsOffline()){
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",sudoPlayer.PlayerId)
    }

    // Load Struct Type
    structTypeFound := structure.LoadType()
    if (!structTypeFound) {
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct Type (%d) was not found. Building a Struct with schematics might be tough", structure.GetTypeId())
    }

    var weapon              types.TechActiveWeaponry
    var weaponControl       types.TechWeaponControl
    var weaponCharge        uint64
    var weaponAmbits        uint64
    var weaponTargets       uint64
    var weaponShots         uint64
    var weaponDamage        uint64
    var weaponBlockable     bool
    var weaponCounterable   bool
    var weaponRecoilDamage  uint64
    var weaponShotSuccessRate fraction.Fraction
    var fractionErr error

    switch msg.WeaponSystem {

        case types.TechWeaponSystem_primaryWeapon:
          weapon                = structure.StructType.PrimaryWeapon
          weaponControl         = structure.StructType.PrimaryWeaponControl
          weaponCharge          = structure.StructType.PrimaryWeaponCharge
          weaponAmbits          = structure.StructType.PrimaryWeaponAmbits
          weaponTargets         = structure.StructType.PrimaryWeaponTargets
          weaponShots           = structure.StructType.PrimaryWeaponShots
          weaponDamage          = structure.StructType.PrimaryWeaponDamage
          weaponBlockable       = structure.StructType.PrimaryWeaponBlockable
          weaponCounterable     = structure.StructType.PrimaryWeaponCounterable
          weaponRecoilDamage    = structure.StructType.PrimaryWeaponRecoilDamage
          weaponShotSuccessRate, fractionErr = fraction.New(structure.StructType.PrimaryWeaponShotSuccessRateNumerator, structure.StructType.PrimaryWeaponShotSuccessRateDenominator)

        case types.TechWeaponSystem_secondaryWeapon:
          weapon                = structure.StructType.SecondaryWeapon
          weaponControl         = structure.StructType.SecondaryWeaponControl
          weaponCharge          = structure.StructType.SecondaryWeaponCharge
          weaponAmbits          = structure.StructType.SecondaryWeaponAmbits
          weaponTargets         = structure.StructType.SecondaryWeaponTargets
          weaponShots           = structure.StructType.SecondaryWeaponShots
          weaponDamage          = structure.StructType.SecondaryWeaponDamage
          weaponBlockable       = structure.StructType.SecondaryWeaponBlockable
          weaponCounterable     = structure.StructType.SecondaryWeaponCounterable
          weaponRecoilDamage    = structure.StructType.SecondaryWeaponRecoilDamage
          weaponShotSuccessRate, fractionErr = fraction.New(structure.StructType.SecondaryWeaponShotSuccessRateNumerator, structure.StructType.SecondaryWeaponShotSuccessRateDenominator)

        default:
            return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "No valid weapon system provided")
    }

    if (fractionErr != nil) {
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "We've got a success rate issue in the Struct Type (%d). Pls tell an adult.", structure.StructType.Id)
    }

    // Pacify the compiler
    _ = weapon
    _ = weaponControl
    _ = weaponCharge
    _ = weaponAmbits
    _ = weaponTargets
    _ = weaponShots
    _ = weaponDamage
    _ = weaponBlockable
    _ = weaponCounterable
    _ = weaponRecoilDamage
    _ = weaponShotSuccessRate


    // Check Sudo Player Charge
    playerCharge := k.GetPlayerCharge(ctx, structure.GetOwner())
    if (playerCharge < weaponCharge) {
        k.DischargePlayer(ctx, structure.GetOwner())
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d for this attack, but player (%s) only had %d", structure.GetTypeId() , weaponCharge, structure.GetOwner(), playerCharge)
    }


    // Load Defender?
    // Load Defender Location?
        // Load Planet?
        // Load Fleet

    // Load attacker location?


    /* Can the attacker attack? */
        /*
            - Is the Attacker Online? (Done ✅)
            - Is the Defender Destroyed?
            - Is the Defender within Range of the Attackers Position
            - Is the Defender within Range of the Attackers Weapon?
                - Including Stealth
            - Is the Defender has a Defensive Block (None MVP ✅)
            - Does the Planet have a Defensive Block (None MVP ✅)

        */



    // Is the defending struct within range of the attack type?
    // Struct and Defender are within the same battle range (fleet comparisons, planets, etc)

    /* Struct Range
        The ability of a Struct to target an enemy Struct depends on a number of factors.

        - Attacker location type (Planet or Fleet)
            - Where the attacker fleet location is
                - planet and position in the list
        - Defender location type (Planet or Fleet)
            - Where the defender fleet location is
                - planet and position in the list


        If the attacker is on a planet
            they can only fight a struct on a fleet
                and that fleet must be at the planet
                    and the fleet must be first in line of attack


        If the attacker is on a fleet
           the attacker can attack the planet or its fleet if they are
            1) next in line in the queue
            2) beside the fleet they're attacking

    */
    /*
    switch structure.LocationType {
        case types.ObjectType_fleet:

        case types.ObjectType_planet:
        default:
            err = sdkerrors.Wrapf(types.ErrStructAction, "Struct (%s) cannot attack from this location (%s) ", structure.Id, structure.LocationType)
    }

    // Attack in Range of Ambit




    attackBlocked = false
    // Check the Defenders
    defenders = k.GetStructDefenders(ctx, msg.TargetStructId)
    for defenders as defender

    */



    k.DischargePlayer(ctx, structure.GetOwner())

	return &types.MsgStructAttackResponse{}, nil
}
