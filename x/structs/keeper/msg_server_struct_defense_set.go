package keeper

import (
	"context"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"


	"structs/x/structs/types"
)

/*
message MsgStructDefenseSet {
  option (cosmos.msg.v1.signer) = "creator";

  string creator              = 1;
  string defenderStructId     = 2;
  string protectedStructId    = 3;
}
*/

func (k msgServer) StructDefenseSet(goCtx context.Context, msg *types.MsgStructDefenseSet) (*types.MsgStructStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // load struct
    structure := k.GetStructCacheFromId(ctx, msg.DefenderStructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    if !structure.LoadStruct(){
        return &types.MsgStructStatusResponse{}, types.NewObjectNotFoundError("struct", msg.DefenderStructId)
    }

    if structure.GetOwner().IsHalted() {
        return &types.MsgStructStatusResponse{}, types.NewPlayerHaltedError(structure.GetOwnerId(), "defense_set").WithStruct(msg.DefenderStructId)
    }

    if structure.IsOffline() {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, types.NewStructStateError(msg.DefenderStructId, "offline", "online", "defense_set")
    }

    if !structure.IsCommandable() {
        k.DischargePlayer(ctx, structure.GetOwnerId())
        return &types.MsgStructStatusResponse{}, types.NewFleetCommandError(structure.GetStructId(), "no_command_struct")
    }

    // Check Player Charge
    if (structure.GetOwner().GetCharge() < structure.GetStructType().DefendChangeCharge) {
        err := types.NewInsufficientChargeError(structure.GetOwnerId(), structure.GetStructType().DefendChangeCharge, structure.GetOwner().GetCharge(), "defend").WithStructType(structure.GetStructType().Id)
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, err
    }

    if structure.GetOwner().IsOffline(){
        return &types.MsgStructStatusResponse{}, types.NewPlayerPowerError(structure.GetOwnerId(), "offline")
    }


    // TODO move this into the cache system directly. 

    //load target
    protectedStructure, protectedStructureFound := k.GetStruct(ctx,  msg.ProtectedStructId)
    if (!protectedStructureFound) {
        return &types.MsgStructStatusResponse{}, types.NewObjectNotFoundError("struct", msg.ProtectedStructId)
    }

    // Are they within defensive range
        // Are they at the same location - great
        // Is defender on a fleet and protected on a planet and fleet is at the planet - perf
        // is protected on a fleet and defender on a planet and fleet is at the planet - lfg
    inRange := false
    if (protectedStructure.LocationId == structure.GetLocationId()) {
        inRange = true
    } else {
       if (structure.GetLocationType() == types.ObjectType_fleet) && (protectedStructure.LocationType == types.ObjectType_planet){
            structureFleet, _ := k.GetFleet(ctx, structure.GetLocationId())
            if (structureFleet.LocationId == protectedStructure.LocationId) {
                inRange = true
            }
       } else if (structure.GetLocationType() == types.ObjectType_planet) && (protectedStructure.LocationType == types.ObjectType_fleet){
            protectedStructureFleet, _ := k.GetFleet(ctx, protectedStructure.LocationId)
            if (protectedStructureFleet.LocationId == structure.GetLocationId()) {
                inRange = true
            }
       }
    }

    if (!inRange) {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, types.NewStructLocationError(structure.GetStructType().Id, "", "not_in_range").WithStruct(structure.GetStructId()).WithLocation("struct", msg.ProtectedStructId)
    }


    k.SetStructDefender(ctx, msg.ProtectedStructId, protectedStructure.Index, structure.GetStructId())

    structure.GetOwner().Discharge()
    structure.Commit()

	return &types.MsgStructStatusResponse{}, nil
}
