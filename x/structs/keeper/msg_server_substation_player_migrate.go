package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerMigrate(goCtx context.Context, msg *types.MsgSubstationPlayerMigrate) (*types.MsgSubstationPlayerMigrateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator), true)
    if (!playerFound) {
        return &types.MsgSubstationPlayerMigrateResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }



    substationObjectPermissionId := GetObjectPermissionIDBytes(msg.SubstationId, player.Id)
    // check that the calling player has substation permissions
    if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationPlayerMigrateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%d) has no Substation Connect Player permissions ", player.Id)
    }

    // check that the account has energy management permissions
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)
    if(!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationPlayerMigrateResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }

    substation, sourceSubstationFound := k.GetSubstation(ctx, msg.SubstationId, false)
    if (!sourceSubstationFound) {
        return &types.MsgSubstationPlayerMigrateResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "substation (%s) used for player connection not found", msg.SubstationId)
    }


    var targetPlayers []types.Player
    for _, targetPlayerId := range msg.PlayerId {

        // check permissions
        if (player.Id != targetPlayerId) {
            // check that the calling player has target player permissions
            playerObjectPermissionId := GetObjectPermissionIDBytes(targetPlayerId, player.Id)
            if (!k.PermissionHasOneOf(ctx, playerObjectPermissionId, types.PermissionGrid)) {
                return &types.MsgSubstationPlayerMigrateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%s) has no Grid permissions on target (%s) ", player.Id, targetPlayerId)
            }
        }

        targetPlayer, targetPlayerFound := k.GetPlayer(ctx, targetPlayerId, true)
        if (!targetPlayerFound) {
            return &types.MsgSubstationPlayerMigrateResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Target player (%s) could be be found", targetPlayerId)
        }
        targetPlayers = append(targetPlayers, targetPlayer)
    }


    for _, migratePlayer := range targetPlayers {
        // connect to new substation
    	// This call handles the disconnection from other substations as well
        k.SubstationConnectPlayer(ctx, substation, migratePlayer)
    }

	return &types.MsgSubstationPlayerMigrateResponse{}, nil
}
