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

func (k msgServer) StructBuildInitiate(goCtx context.Context, msg *types.MsgStructBuildInitiate) (*types.MsgStructStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // Load the Owner Player
    owner, err := cache.K.GetPlayerCacheFromId(cache.Ctx, msg.PlayerId)
    if (err != nil) {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build initialization requires Player account but none associated with %s", msg.Creator)
    }

    // Check address play permissions
    permissionError := owner.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    // Load the Struct Type
    structType, structTypeFound := k.GetStructType(ctx, msg.StructTypeId)
    if !structTypeFound {
        return MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct Type (%d) was not found. Building a Struct with schematics might be tough", msg.StructTypeId)
    }

    // Check that the player can build more of this type of Struct
    if (structType.GetBuildLimit() > 0) {
        if (k.GetStructAttribute(ctx, GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, owner.GetPlayerId(), msg.StructTypeId)) >= structType.GetBuildLimit()) {
            k.DischargePlayer(ctx, sudoPlayer.Id)
            return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) cannot build more of this type ",sudoPlayer.Id)
        }
    }

    // Check Sudo Player Charge
    playerCharge := k.GetPlayerCharge(ctx, owner.GetPlayerId())
    if (playerCharge < structType.BuildCharge) {
        k.DischargePlayer(ctx, owner.GetPlayerId())
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d to build, but player (%s) only had %d", msg.StructTypeId, structType.BuildCharge, sudoPlayer.Id, playerCharge)
    }










    if (!sudoPlayer.IsOnline()){
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",sudoPlayer.Id)
    }



    var planetSlots uint64
    var planetSlot string
    // Check Ambit / Slot
    switch msg.OperatingAmbit {
        case types.Ambit_land:
            planetSlots = planet.LandSlots
            planetSlot  = planet.Land[msg.Slot]
        case types.Ambit_water:
            planetSlots = planet.WaterSlots
            planetSlot  = planet.Water[msg.Slot]
        case types.Ambit_air:
            planetSlots = planet.AirSlots
            planetSlot  = planet.Air[msg.Slot]
        case types.Ambit_space:
            planetSlots = planet.SpaceSlots
            planetSlot  = planet.Space[msg.Slot]
        default:
            return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The Struct Build was initiated on a non-existent ambit")
    }

    if (msg.Slot >= planetSlots) {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The planet (%s) specified doesn't have that slot available to build on", msg.PlanetId)
    }
    if (planetSlot != "") {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The planet (%s) specified already has a struct on that slot", msg.PlanetId)
    }

    // This process will check the location details to make sure they're acceptable based on the structType
    structure, err := types.CreateBaseStruct(structType, msg.Creator, sudoPlayer.Id, planet.Id, types.ObjectType_planet, msg.OperatingAmbit, msg.Slot)
    if (err != nil) {
        return &types.MsgStructStatusResponse{}, err
    }

    // Check player Load for the buildDraw capacity
    sudoPlayerTotalLoad := sudoPlayer.Load + sudoPlayer.StructsLoad
    sudoPlayerTotalCapacity := sudoPlayer.Capacity + sudoPlayer.CapacitySecondary
    // Is load completely shot already?
    if (sudoPlayerTotalLoad > sudoPlayerTotalCapacity) {
        k.DischargePlayer(ctx, sudoPlayer.Id)
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct Type (%d) required a draw of %d during build, but player (%s) has none available", msg.StructTypeId, structType.BuildDraw, sudoPlayer.Id)

    // Otherwise is the difference enough to support the buildDraw rate
    } else if ((sudoPlayerTotalCapacity - sudoPlayerTotalLoad) < structType.BuildDraw) {
        k.DischargePlayer(ctx, sudoPlayer.Id)
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct Type (%d) required a draw of %d during build, but player (%s) has %d available", msg.StructTypeId, structType.BuildDraw, sudoPlayer.Id,(sudoPlayerTotalCapacity - sudoPlayerTotalLoad))
    }

    // Increase the Struct load of the sudoPlayer
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, sudoPlayer.Id), structType.BuildDraw)

    // Discharge Owner Player Charge  (set last block time)
    k.DischargePlayer(ctx, sudoPlayer.Id)

    // Append Struct
    structure = k.AppendStruct(ctx, structure, structType)
    k.SetStructAttributeIncrement(ctx, GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, sudoPlayer.Id, msg.StructTypeId), 1)

    // Update the cross reference on the planet
    err = planet.SetSlot(structure)
    if (err != nil) {
        // This is a pretty huge problem if we get here since all the other crap is done.
        // Roll back transaction?
        return &types.MsgStructStatusResponse{}, err
    }
    k.SetPlanet(ctx, planet)

	return &types.MsgStructStatusResponse{Struct: structure}, nil
}
