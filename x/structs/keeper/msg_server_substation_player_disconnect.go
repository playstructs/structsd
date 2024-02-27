package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerDisconnect(goCtx context.Context, msg *types.MsgSubstationPlayerDisconnect) (*types.MsgSubstationPlayerDisconnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)


	player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator), true)
    if (!playerFound) {
        return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }

	targetPlayer, targetPlayerFound := k.GetPlayer(ctx, msg.PlayerId, true)
    if (!targetPlayerFound) {
        return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Target player (%d) could be be found", msg.PlayerId)
    }

    // check that the calling player has substation permissions
    if (!k.SubstationPermissionHasOneOf(ctx, targetPlayer.SubstationId, player.Id, types.SubstationPermissionConnectPlayer)) {
        return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%d) has no Substation Connect Player permissions ", player.Id)
    }


    if (player.Id != msg.PlayerId) {
        // check that the calling player has target player permissions
        if (!k.PlayerPermissionHasOneOf(ctx, msg.PlayerId, player.Id, types.PlayerPermissionSubstation)) {
            return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%d) has no Player Substation permissions ", player.Id)
        }
    }

    // check that the account has energy management permissions
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if (playerPermissions&types.AddressPermissionManageEnergy == 0) {
        return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }


	// connect to new substation
	// This call handles the disconnection from other substations as well
    k.SubstationDisconnectPlayer(ctx, targetPlayer)

	return &types.MsgSubstationPlayerDisconnectResponse{}, nil
}
