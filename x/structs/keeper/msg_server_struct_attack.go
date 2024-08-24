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

    callingPlayerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (callingPlayerIndex == 0) {
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct actions requires Player account but none associated with %s", msg.Creator)
    }
    callingPlayerId := GetObjectID(types.ObjectType_player, callingPlayerIndex)

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionPlay)) {
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structStatusAttributeId := GetStructAttributeIDByObjectId(types.StructAttributeType_status, msg.OperatingStructId)

    structure, structureFound := k.GetStruct(ctx, msg.OperatingStructId)
    if (!structureFound) {
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct (%s) not found", msg.OperatingStructId)
    }

    // Is the Struct online?
    if (k.StructAttributeFlagHasOneOf(ctx, structStatusAttributeId, uint64(types.StructStateOnline))) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) is offline. Activate it", msg.OperatingStructId)
    }

    if (callingPlayerId != structure.Owner) {
        // Check permissions on Creator on Planet
        playerPermissionId := GetObjectPermissionIDBytes(structure.Owner, callingPlayerId)
        if (!k.PermissionHasOneOf(ctx, playerPermissionId, types.PermissionPlay)) {
            return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling account (%s) has no play permissions on target player (%s)", callingPlayerId, structure.Owner)
        }
    }
    sudoPlayer, _ := k.GetPlayer(ctx, structure.Owner, true)
    if (!sudoPlayer.IsOnline()){
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",sudoPlayer.Id)
    }

    // Load Struct Type
    structType, structTypeFound := k.GetStructType(ctx, structure.Type)
    if (!structTypeFound) {
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct Type (%d) was not found. Building a Struct with schematics might be tough", structure.Type)
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
          weapon                = structType.PrimaryWeapon
          weaponControl         = structType.PrimaryWeaponControl
          weaponCharge          = structType.PrimaryWeaponCharge
          weaponAmbits          = structType.PrimaryWeaponAmbits
          weaponTargets         = structType.PrimaryWeaponTargets
          weaponShots           = structType.PrimaryWeaponShots
          weaponDamage          = structType.PrimaryWeaponDamage
          weaponBlockable       = structType.PrimaryWeaponBlockable
          weaponCounterable     = structType.PrimaryWeaponCounterable
          weaponRecoilDamage    = structType.PrimaryWeaponRecoilDamage
          weaponShotSuccessRate, fractionErr = fraction.New(structType.PrimaryWeaponShotSuccessRateNumerator, structType.PrimaryWeaponShotSuccessRateDenominator)

        case types.TechWeaponSystem_secondaryWeapon:
          weapon                = structType.SecondaryWeapon
          weaponControl         = structType.SecondaryWeaponControl
          weaponCharge          = structType.SecondaryWeaponCharge
          weaponAmbits          = structType.SecondaryWeaponAmbits
          weaponTargets         = structType.SecondaryWeaponTargets
          weaponShots           = structType.SecondaryWeaponShots
          weaponDamage          = structType.SecondaryWeaponDamage
          weaponBlockable       = structType.SecondaryWeaponBlockable
          weaponCounterable     = structType.SecondaryWeaponCounterable
          weaponRecoilDamage    = structType.SecondaryWeaponRecoilDamage
          weaponShotSuccessRate, fractionErr = fraction.New(structType.SecondaryWeaponShotSuccessRateNumerator, structType.SecondaryWeaponShotSuccessRateDenominator)

        default:
            return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "No valid weapon system provided")
    }

    if (fractionErr != nil) {
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "We've got a success rate issue in the Struct Type (%d). Pls tell an adult.", structType.Id)
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
    playerCharge := k.GetPlayerCharge(ctx, structure.Owner)
    if (playerCharge < weaponCharge) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructAttackResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d for this attack, but player (%s) only had %d", structure.Type, weaponCharge, structure.Owner, playerCharge)
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



    k.DischargePlayer(ctx, structure.Owner)

	return &types.MsgStructAttackResponse{}, nil
}
