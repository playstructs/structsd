package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerConnect(goCtx context.Context, msg *types.MsgSubstationPlayerConnect) (*types.MsgSubstationPlayerConnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgSubstationPlayerConnectResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }

	targetPlayer, targetPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
    if (!targetPlayerFound) {
        return &types.MsgSubstationPlayerConnectResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Target player (%s) could be be found", player.Id)
    }

    substationObjectPermissionId := GetObjectPermissionIDBytes(msg.SubstationId, player.Id)
    // check that the calling player has substation permissions
    if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationPlayerConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%s) has no Energy Management permissions on Substation (%s) ", player.Id, msg.SubstationId)
    }

    if (player.Id != msg.PlayerId) {
        // check that the calling player has target player permissions
        playerObjectPermissionId := GetObjectPermissionIDBytes(msg.PlayerId, player.Id)
        if (!k.PermissionHasOneOf(ctx, playerObjectPermissionId, types.PermissionGrid)) {
            return &types.MsgSubstationPlayerConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%s) has no Energy Management permissions on target player (%s) ", player.Id, msg.PlayerId)
        }
    }

    // check that the account has energy management permissions
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)
    if(!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationPlayerConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }

    substation, sourceSubstationFound := k.GetSubstation(ctx, msg.SubstationId)
    if (!sourceSubstationFound) {
        return &types.MsgSubstationPlayerConnectResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "substation (%s) used for player connection not found", msg.SubstationId)
    }

	// connect to new substation
	// This call handles the disconnection from other substations as well
    k.SubstationConnectPlayer(ctx, substation, targetPlayer)

	return &types.MsgSubstationPlayerConnectResponse{}, nil
}
