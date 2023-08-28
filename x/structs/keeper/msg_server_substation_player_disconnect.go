package keeper

import (
	"context"
    "strconv"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerDisconnect(goCtx context.Context, msg *types.MsgSubstationPlayerDisconnect) (*types.MsgSubstationPlayerDisconnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)


	player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }

	targetPlayer, targetPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
    if (!targetPlayerFound) {
        playerIdString := strconv.FormatUint(msg.PlayerId, 10)
        return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Target player (%s) could be be found", playerIdString)
    }

    // check that the calling player has substation permissions
    if (!k.SubstationPermissionHasOneOf(ctx, targetPlayer.SubstationId, player.Id, types.SubstationPermissionConnectPlayer)) {
        playerIdString := strconv.FormatUint(player.Id, 10)
        return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%s) has no Substation Connect Player permissions ", playerIdString)
    }


    if (player.Id != msg.PlayerId) {
        // check that the calling player has target player permissions
        if (!k.PlayerPermissionHasOneOf(ctx, msg.PlayerId, player.Id, types.PlayerPermissionSubstation)) {
            playerIdString := strconv.FormatUint(player.Id, 10)
            return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%s) has no Player Substation permissions ", playerIdString)
        }
    }

    // check that the account has energy management permissions
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if (playerPermissions&types.AddressPermissionManageEnergy == 0) {
        return &types.MsgSubstationPlayerDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }


    k.SubstationDecrementConnectedPlayerLoad(ctx, player.SubstationId, 1)

    player.SetSubstation(0)
    k.SetPlayer(ctx, player)

	return &types.MsgSubstationPlayerDisconnectResponse{}, nil
}
