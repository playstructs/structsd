package keeper

import (
	"context"

    //"fmt"

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



    k.DischargePlayer(ctx, structure.GetOwnerId())

	return &types.MsgStructAttackResponse{}, nil
}
