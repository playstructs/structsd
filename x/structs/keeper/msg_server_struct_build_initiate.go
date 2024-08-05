package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

/* MsgStructBuildInitiate
  string creator        = 1;
  uint64 structTypeId   = 2;
  string planetId       = 3;
  ambit operatingAmbit  = 4;
  uint64 slot           = 5;
  */

func (k msgServer) StructBuildInitiate(goCtx context.Context, msg *types.MsgStructBuildInitiate) (*types.MsgStructBuildInitiateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // Load Initiator Player from the Creator
    callingPlayerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (callingPlayerIndex == 0) {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build initialization requires Player account but none associated with %s", msg.Creator)
    }
    callingPlayer, _ := k.GetPlayerFromIndex(ctx, callingPlayerIndex, true)


    // Load Planet from the PlanetId
    planet, planetFound := k.GetPlanet(ctx, msg.PlanetId)
    if (!planetFound) {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Planet (%s) was ot found. Building a Struct in a void might be tough", msg.PlanetId)
    }

    sudoPlayerIndex := k.GetPlayerIndexFromAddress(ctx, planet.Owner)
    if (sudoPlayerIndex == 0) {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build initialization requires Player but somehow planet has none %s", planet.Owner)
    }
    sudoPlayer, _ := k.GetPlayerFromIndex(ctx, sudoPlayerIndex, true)

    if (sudoPlayer.Id != callingPlayer.Id) {
        // Check permissions on Creator on Planet
        playerPermissionId := GetObjectPermissionIDBytes(sudoPlayer.Id, callingPlayer.Id),
        if (!k.PermissionHasOneOf(ctx, playerPermissionId, types.PermissionPlay)) {
            return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling account (%s) has no play permissions on target player (%s)", callingPlayer.Id, sudoPlayer.Id)
        }

    }

    // Make sure the address calling this has Play permissions
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionPlay)) {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    // Check Ambit / Slot
    switch msg.OperatingAmbit {
        case types.Ambit_land:
            planetSlots := planet.LandSlots
            planetSlot  := planet.Land[msg.Slot]
        case types.Ambit_water:
            planetSlots := planet.WaterSlots
            planetSlot  := planet.Water[msg.Slot]
        case types.Ambit_air:
            planetSlots := planet.AirSlots
            planetSlot  := planet.Air[msg.Slot]
        case types.Ambit_space:
            planetSlots := planet.SpaceSlots
            planetSlot  := planet.Space[msg.Slot]
        default:
            return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The Struct Build was initiated on a non-existant ambit")
    }

    if (msg.Slot >= planetSlots) {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The planet (%s) specified doesn't have that slot available to build on", msg.PlanetId)
    }
    if (planetSlot != "") {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The planet (%s) specified already has a struct on that slot", msg.PlanetId)
    }

    // Load Struct Type
    structType, structTypeFound := k.GetStructType(ctx, msg.StructTypeId)
    if (!structTypeFound) {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct Type (%d) was not found. Building a Struct with schematics might be tough", msg.StructTypeId)
    }

    // Check Sudo Player Charge
    playerCharge := k.GetPlayerCharge(ctx, sudoPlayer)
    if (playerCharge < structType.BuildCharge) {
        k.DischargePlayer(ctx, sudoPlayer)
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d to build, but player (%s) only had %d", msg.StructTypeId, structType.BuildCharge, sudoPlayer.Id, playerCharge)
    }

    // This process will check the location details to make sure they're acceptable based on the structType
    structure, err := types.CreateBaseStruct(structType, msg.Creator, sudoPlayer.Id, planet.Id, types.ObjectType_planet, msg.Ambit, msg.Slot)
    if (err != nil) {
        return &types.MsgStructBuildInitiateResponse{}, err
    }

    // Check player Load for the buildDraw capacity
    sudoPlayerTotalLoad := sudoPlayer.Load + sudoPlayer.StructsLoad
    sudoPlayerTotalCapacity := sudoPlayer.Capacity + sudoPlayer.CapacitySecondary
    // Is load complete shot already?
    if (sudoPlayerTotalLoad > sudoPlayerTotalCapacity) {
        k.DischargePlayer(ctx, sudoPlayer)
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct Type (%d) required a draw of %d during build, but player (%s) has none available", msg.StructTypeId, structType.BuildDraw, sudoPlayer.Id)

    // Otherwise is the difference enough to support the buildDraw rate
    } else if ((sudoPlayerTotalCapacity - sudoPlayerTotalLoad) < structType.BuildDraw) {
        k.DischargePlayer(ctx, sudoPlayer)
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct Type (%d) required a draw of %d during build, but player (%s) has %d available", msg.StructTypeId, structType.BuildDraw, sudoPlayer.Id,(sudoPlayerTotalCapacity - sudoPlayerTotalLoad))
    }

    // Increase the Struct load of the sudoPlayer
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, sudoPlayer.Id), structType.BuildDraw)

    // Discharge Owner Player Charge  (set last block time)
    k.DischargePlayer(ctx, sudoPlayer)

    // Append Struct
    structure = k.AppendStruct(ctx, structure, structType)

    // Update the cross reference on the planet
    err = planet.SetSlot(structure)
    if (err != nil) {
        // This is a pretty huge problem if we get here since all the other crap is done.
        // Roll back transaction?
        return &types.MsgStructBuildInitiateResponse{}, err
    }
    k.SetPlanet(ctx, planet)

	return &types.MsgStructBuildInitiateResponse{Struct: structure}, nil
}
