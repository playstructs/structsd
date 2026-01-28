package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerMigrate(goCtx context.Context, msg *types.MsgSubstationPlayerMigrate) (*types.MsgSubstationPlayerMigrateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgSubstationPlayerMigrateResponse{}, types.NewPlayerRequiredError(msg.Creator, "substation_player_migrate")
    }

    substationObjectPermissionId := GetObjectPermissionIDBytes(msg.SubstationId, player.Id)
    // check that the calling player has substation permissions
    if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationPlayerMigrateResponse{}, types.NewPermissionError("player", player.Id, "substation", msg.SubstationId, uint64(types.PermissionGrid), "player_migrate")
    }

    // check that the account has energy management permissions
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)
    if(!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationPlayerMigrateResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionGrid), "energy_management")
    }

    substation, sourceSubstationFound := k.GetSubstation(ctx, msg.SubstationId)
    if (!sourceSubstationFound) {
        return &types.MsgSubstationPlayerMigrateResponse{}, types.NewObjectNotFoundError("substation", msg.SubstationId)
    }


    var targetPlayers []types.Player
    for _, targetPlayerId := range msg.PlayerId {

        // check permissions
        if (player.Id != targetPlayerId) {
            // check that the calling player has target player permissions
            playerObjectPermissionId := GetObjectPermissionIDBytes(targetPlayerId, player.Id)
            if (!k.PermissionHasOneOf(ctx, playerObjectPermissionId, types.PermissionGrid)) {
                return &types.MsgSubstationPlayerMigrateResponse{}, types.NewPermissionError("player", player.Id, "player", targetPlayerId, uint64(types.PermissionGrid), "player_migrate")
            }
        }

        targetPlayer, targetPlayerFound := k.GetPlayer(ctx, targetPlayerId)
        if (!targetPlayerFound) {
            return &types.MsgSubstationPlayerMigrateResponse{}, types.NewObjectNotFoundError("player", targetPlayerId)
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
