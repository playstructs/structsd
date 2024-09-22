package keeper

import (
	"context"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"


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
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct (%s) does not exist", msg.DefenderStructId)
    }

    if structure.IsOffline() {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) already built", msg.DefenderStructId)
    }

    // Check Player Charge
    if (structure.GetOwner().GetCharge() < structure.GetStructType().DefendChangeCharge) {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d to change defensive stance, but player (%s) only had %d", structure.GetStructType().Id, structure.GetStructType().DefendChangeCharge, structure.GetOwnerId(), structure.GetOwner().GetCharge() )
    }

    if structure.GetOwner().IsOffline(){
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",structure.GetOwnerId())
    }


    // TODO move this into the cache system directly. 

    //load target
    protectedStructure, protectedStructureFound := k.GetStruct(ctx,  msg.ProtectedStructId)
    if (!protectedStructureFound) {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct (%s) not found", msg.ProtectedStructId)
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
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) is not within ranger to defend Struct (%s) ", structure.GetStructId(), msg.ProtectedStructId)
    }

    k.SetStructDefender(ctx, protectedStructure, structure.GetStruct())

    structure.GetOwner().Discharge()
    structure.Commit()

	return &types.MsgStructStatusResponse{}, nil
}
