package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerDisconnect(goCtx context.Context, msg *types.MsgSubstationPlayerDisconnect) (*types.MsgSubstationPlayerDisconnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }

	targetPlayer, targetPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
    if (!targetPlayerFound) {
        return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Target player (%s) could be be found", msg.PlayerId)
    }

    // Check if the Calling Player isn't Target Player
    // If they aren't they'll either need Grid Permission on the Player or on the Substation
    if (player.Id != msg.PlayerId) {
        // check that the Calling Player has Grid Permissions on the Substation
        substationObjectPermissionId := GetObjectPermissionIDBytes(player.SubstationId, player.Id)
        if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {

            // Check that the Calling Player has Grid Permissions on the Target Player
            playerObjectPermissionId := GetObjectPermissionIDBytes(msg.PlayerId, player.Id)
            if (!k.PermissionHasOneOf(ctx, playerObjectPermissionId, types.PermissionGrid)) {

                // Calling Player has no authority over this process
                return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%s) has no Energy Management permissions on Player (%s) or Substation (%s)", player.Id, targetPlayer.Id, player.SubstationId)
            }
        }
    }

    // check that the account has energy management permissions
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionGrid)) {
       return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }

	// connect to new substation
	// This call handles the disconnection from other substations as well
    k.SubstationDisconnectPlayer(ctx, targetPlayer)

	return &types.MsgSubstationPlayerDisconnectResponse{}, nil
}
