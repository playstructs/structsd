package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerMigrate(goCtx context.Context, msg *types.MsgSubstationPlayerMigrate) (*types.MsgSubstationPlayerMigrateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgSubstationPlayerMigrateResponse{}, err
    }

    substationObjectPermissionId := GetObjectPermissionIDBytes(msg.SubstationId, player.GetPlayerId())
    // check that the calling player has substation permissions
    if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationPlayerMigrateResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "substation", msg.SubstationId, uint64(types.PermissionGrid), "player_migrate")
    }

    // check that the account has energy management permissions
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)
    if(!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationPlayerMigrateResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionGrid), "energy_management")
    }

    substation := cc.GetSubstation(msg.SubstationId)
    if (!substation.LoadSubstation()) {
        return &types.MsgSubstationPlayerMigrateResponse{}, types.NewObjectNotFoundError("substation", msg.SubstationId)
    }


    var targetPlayers []*PlayerCache
    for _, targetPlayerId := range msg.PlayerId {

        // check permissions
        if (player.GetPlayerId() != targetPlayerId) {
            // check that the calling player has target player permissions
            playerObjectPermissionId := GetObjectPermissionIDBytes(targetPlayerId, player.GetPlayerId())
            if (!k.PermissionHasOneOf(ctx, playerObjectPermissionId, types.PermissionGrid)) {
                return &types.MsgSubstationPlayerMigrateResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "player", targetPlayerId, uint64(types.PermissionGrid), "player_migrate")
            }
        }

        targetPlayer, err := cc.GetPlayer(targetPlayerId)
        if err != nil {
            return &types.MsgSubstationPlayerMigrateResponse{}, err
        }
        if !targetPlayer.LoadPlayer() {
            return &types.MsgSubstationPlayerMigrateResponse{}, types.NewObjectNotFoundError("player", targetPlayerId)
        }
        targetPlayers = append(targetPlayers, targetPlayer)
    }


    for _, migratePlayer := range targetPlayers {
        // connect to new substation
    	// This call handles the disconnection from other substations as well
        migratePlayer.MigrateSubstation(substation.GetSubstationId())
    }

	return &types.MsgSubstationPlayerMigrateResponse{}, nil
}
