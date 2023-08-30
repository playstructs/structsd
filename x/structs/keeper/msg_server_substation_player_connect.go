package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerConnect(goCtx context.Context, msg *types.MsgSubstationPlayerConnect) (*types.MsgSubstationPlayerConnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgSubstationPlayerConnectResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }

	targetPlayer, targetPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
    if (!targetPlayerFound) {
        return &types.MsgSubstationPlayerConnectResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Target player (%d) could be be found", player.Id)
    }

    // check that the calling player has substation permissions
    if (!k.SubstationPermissionHasOneOf(ctx, msg.SubstationId, player.Id, types.SubstationPermissionConnectPlayer)) {
        return &types.MsgSubstationPlayerConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%d) has no Substation Connect Player permissions ", player.Id)
    }


    if (player.Id != msg.PlayerId) {
        // check that the calling player has target player permissions
        if (!k.PlayerPermissionHasOneOf(ctx, msg.PlayerId, player.Id, types.PlayerPermissionSubstation)) {
            return &types.MsgSubstationPlayerConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%d) has no Player Substation permissions ", player.Id)
        }
    }

    // check that the account has energy management permissions
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if (playerPermissions&types.AddressPermissionManageEnergy == 0) {
        return &types.MsgSubstationPlayerConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }

    substation, sourceSubstationFound := k.GetSubstation(ctx, msg.SubstationId, false)
    if (!sourceSubstationFound) {
        return &types.MsgSubstationPlayerConnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceNotFound, "substation (%d) used for player connection not found", msg.SubstationId)
    }

	// connect to new substation
	// This call handles the disconnection from other substations as well
    k.SubstationConnectPlayer(ctx, substation, targetPlayer)

	return &types.MsgSubstationPlayerConnectResponse{}, nil
}
