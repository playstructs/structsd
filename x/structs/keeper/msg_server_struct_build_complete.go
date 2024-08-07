package keeper

import (
	"context"
    "strconv"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructBuildComplete(goCtx context.Context, msg *types.MsgStructBuildComplete) (*types.MsgStructStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (callingPlayerIndex == 0) {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build actions requires Player account but none associated with %s", msg.Creator)
    }
    callingPlayerId := GetObjectID(types.ObjectType_player, callingPlayerIndex)

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionPlay)) {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct (%s) not found", msg.StructId)
    }

    if (callingPlayer.Id != structure.Owner) {
        // Check permissions on Creator on Planet
        playerPermissionId := GetObjectPermissionIDBytes(structure.Owner, callingPlayerId)
        if (!k.PermissionHasOneOf(ctx, playerPermissionId, types.PermissionPlay)) {
            return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling account (%s) has no play permissions on target player (%s)", callingPlayerId, structure.Owner)
        }
    }
    sudoPlayer, _ := k.GetPlayer(ctx, structure.Owner, true)

    // Load Struct Type
    structType, structTypeFound := k.GetStructType(ctx, structure.Type)
    if (!structTypeFound) {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct Type (%d) was not found. Building a Struct with schematics might be tough", structure.Type)
    }

    // Check Sudo Player Charge
    playerCharge := k.GetPlayerCharge(ctx, structure.Owner)
    if (playerCharge < structType.ActivateCharge) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d to activate, but player (%s) only had %d", msg.StructTypeId, structType.ActivateCharge, structure.Owner, playerCharge)
    }

    // Check player Load for the passiveDraw capacity
    sudoPlayerTotalLoad := sudoPlayer.Load + sudoPlayer.StructsLoad
    sudoPlayerTotalCapacity := sudoPlayer.Capacity + sudoPlayer.CapacitySecondary

    // Remove BuildDraw
    if (sudoPlayerTotalLoad < structType.BuildDraw) {
        // Could add some code here that detects the issue and rebuilds the players grid details.
        // Not a great solutions but might not be awful for longevity of gamestate
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Player %s grid details wrong. Not enough load to remove buildDraw load... something wrong. Pls tell an adult", sudoPlayer.Id)
    }
    sudoPlayerTotalLoad = sudoPlayerTotalLoad - structType.BuildDraw

    // Is load completely shot already?
    if (sudoPlayerTotalLoad > sudoPlayerTotalCapacity) {
        k.DischargePlayer(ctx, sudoPlayer.Id)
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct Type (%d) required a draw of %d during activation, but player (%s) has none available", msg.StructTypeId, structType.BuildDraw, sudoPlayer.Id)

    // Otherwise is the difference enough to support the buildDraw rate
    } else if ((sudoPlayerTotalCapacity - sudoPlayerTotalLoad) < structType.PassiveDraw) {
        k.DischargePlayer(ctx, sudoPlayer.Id)
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct Type (%d) required a draw of %d during activation, but player (%s) has %d available", msg.StructTypeId, structType.BuildDraw, sudoPlayer.Id,(sudoPlayerTotalCapacity - sudoPlayerTotalLoad))
    }

    // Check the Proof
    buildStartBlock                 := GetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structure.Id))
    buildStartBlockString           := strconv.FormatUint(structure.BuildStartBlock , 10)
    hashInput                       := structure.Id + "BUILD" + buildStartBlockString + "NONCE" + msg.Nonce

    currentAge := uint64(ctx.BlockHeight()) - structure.BuildStartBlock

    if (!types.HashBuildAndCheckBuildDifficulty(hashInput, msg.Proof, currentAge)) {
       k.DischargePlayer(ctx, sudoPlayer.Id)
       return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrStructBuildComplete, "Work failure for input (%s) when trying to build Struct %s", hashInput, structure.Id)
    }

    // Inefficient but less likely to mess up
    if (structType.BuildDraw != structType.PassiveDraw) {
        k.SetGridAttributeDecrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, sudoPlayer.Id), structType.BuildDraw)
        k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, sudoPlayer.Id), structType.PassiveDraw)
    }

    // Set the struct status flag to include built
    k.SetStructAttributeFlagAdd(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_status, structure.Id), types.StructStateBuilt | types.StructStateOnline)

    // Shouldn't need to actually update this object
    // k.SetStruct(ctx, structure)


	return &types.MsgStructStatusResponse{Struct: structure}, nil
}
