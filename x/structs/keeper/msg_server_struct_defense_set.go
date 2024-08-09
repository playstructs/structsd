package keeper

import (
	"context"
    /*
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	*/

	"structs/x/structs/types"
)


func (k msgServer) StructDefenseSet(goCtx context.Context, msg *types.MsgStructDefenseSet) (*types.MsgStructStatusResponse, error) {
/*	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (callingPlayerIndex == 0) {
        return &types.MsgStructOreMinerStatusResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct actions requires Player account but none associated with %s", msg.Creator)
    }
    callingPlayerId := GetObjectID(types.ObjectType_player, callingPlayerIndex)

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionPlay)) {
        return &types.MsgStructOreMinerStatusResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structStatusAttributeId := GetStructAttributeIDByObjectId(types.StructAttributeType_status, msg.StructId)

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructOreMinerStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct (%s) not found", msg.StructId)
    }

    // Is the Struct online?
    if (k.StructAttributeFlagHasOneOf(ctx, structStatusAttributeId, uint64(types.StructStateOnline))) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructOreMinerStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) is offline. Activate it", msg.StructId)
    }

    if (callingPlayerId != structure.Owner) {
        // Check permissions on Creator on Planet
        playerPermissionId := GetObjectPermissionIDBytes(structure.Owner, callingPlayerId)
        if (!k.PermissionHasOneOf(ctx, playerPermissionId, types.PermissionPlay)) {
            return &types.MsgStructOreMinerStatusResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling account (%s) has no play permissions on target player (%s)", callingPlayerId, structure.Owner)
        }
    }
    sudoPlayer, _ := k.GetPlayer(ctx, structure.Owner, true)
    if (!sudoPlayer.IsOnline()){
        return &types.MsgStructOreMinerStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",sudoPlayer.Id)
    }

    // Load Struct Type
    structType, structTypeFound := k.GetStructType(ctx, structure.Type)
    if (!structTypeFound) {
        return &types.MsgStructOreMinerStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct Type (%d) was not found. Building a Struct with schematics might be tough", structure.Type)
    }

    // Check Sudo Player Charge
    // Maaaayybe we let the calling player use its charge but idk
    // Then people could have a stack of accounts to increase action throughput
    playerCharge := k.GetPlayerCharge(ctx, structure.Owner)
    if (playerCharge < structType.OreMiningCharge) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructOreMinerStatusResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d for mining, but player (%s) only had %d", structure.Type, structType.ActivateCharge, structure.Owner, playerCharge)
    }

    k.DischargePlayer(ctx, structure.Owner)
*/
	return &types.MsgStructStatusResponse{}, nil
}
