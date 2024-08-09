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

    callingPlayerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (callingPlayerIndex == 0) {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct actions requires Player account but none associated with %s", msg.Creator)
    }
    callingPlayerId := GetObjectID(types.ObjectType_player, callingPlayerIndex)

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionPlay)) {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structStatusAttributeId := GetStructAttributeIDByObjectId(types.StructAttributeType_status, msg.DefenderStructId)

    structure, structureFound := k.GetStruct(ctx, msg.DefenderStructId)
    if (!structureFound) {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct (%s) not found", msg.DefenderStructId)
    }

    // Is the Struct online?
    if (k.StructAttributeFlagHasOneOf(ctx, structStatusAttributeId, uint64(types.StructStateOnline))) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) is offline. Activate it", msg.DefenderStructId)
    }

    if (callingPlayerId != structure.Owner) {
        // Check permissions on Creator on Planet
        playerPermissionId := GetObjectPermissionIDBytes(structure.Owner, callingPlayerId)
        if (!k.PermissionHasOneOf(ctx, playerPermissionId, types.PermissionPlay)) {
            return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling account (%s) has no play permissions on target player (%s)", callingPlayerId, structure.Owner)
        }
    }
    sudoPlayer, _ := k.GetPlayer(ctx, structure.Owner, true)
    if (!sudoPlayer.IsOnline()){
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",sudoPlayer.Id)
    }

    // Load Struct Type
    structType, structTypeFound := k.GetStructType(ctx, structure.Type)
    if (!structTypeFound) {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct Type (%d) was not found. Pls tell an adult.", structure.Type)
    }

    // Check Sudo Player Charge
    // Maaaayybe we let the calling player use its charge but idk
    // Then people could have a stack of accounts to increase action throughput
    playerCharge := k.GetPlayerCharge(ctx, structure.Owner)
    if (playerCharge < structType.DefendChangeCharge) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d for defensive changes, but player (%s) only had %d", structure.Type, structType.ActivateCharge, structure.Owner, playerCharge)
    }

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
    if (protectedStructure.LocationId == structure.LocationId) {
        inRange = true
    } else {
       if (structure.LocationType == types.ObjectType_fleet) && (protectedStructure.LocationType == types.ObjectType_planet){
            structureFleet, _ := k.GetFleet(ctx, structure.LocationId)
            if (structureFleet.LocationId == protectedStructure.LocationId) {
                inRange = true
            }
       } else if (structure.LocationType == types.ObjectType_planet) && (protectedStructure.LocationType == types.ObjectType_fleet){
            protectedStructureFleet, _ := k.GetFleet(ctx, protectedStructure.LocationId)
            if (protectedStructureFleet.LocationId == structure.LocationId) {
                inRange = true
            }
       }
    }

    if (!inRange) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) is not within ranger to defend Struct (%s) ", structure.Id, msg.ProtectedStructId)
    }

    k.SetStructDefender(ctx, protectedStructure, structure)

    k.DischargePlayer(ctx, structure.Owner)

	return &types.MsgStructStatusResponse{}, nil
}
