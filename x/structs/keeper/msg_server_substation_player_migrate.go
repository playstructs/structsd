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

	player, _ := cc.GetPlayer(msg.Creator)
    if player.CheckPlayer() != nil {
        return &types.MsgSubstationPlayerMigrateResponse{}, player.CheckPlayer()
    }

    substation := cc.GetSubstation(msg.SubstationId)
    if substation.CheckSubstation() != nil {
        return &types.MsgSubstationPlayerMigrateResponse{}, types.NewObjectNotFoundError("substation", msg.SubstationId)
    }

    permissionSubstationErr := substation.CanManagePlayerConnections(player)
    if permissionSubstationErr != nil {
        return &types.MsgSubstationPlayerMigrateResponse{}, permissionSubstationErr
    }

    var targetPlayers []*PlayerCache
    for _, targetPlayerId := range msg.PlayerId {

        targetPlayer, err := cc.GetPlayer(targetPlayerId)
        if err != nil {
            return &types.MsgSubstationPlayerMigrateResponse{}, err
        }
        if targetPlayer.CheckPlayer() != nil {
            return &types.MsgSubstationPlayerMigrateResponse{}, types.NewObjectNotFoundError("player", targetPlayerId)
        }

        // check permissions
        if (player.GetPlayerId() != targetPlayerId) {
            if targetPlayer.CanManageGridBy(msg.Creator) != nil {
                return &types.MsgSubstationPlayerMigrateResponse{}, targetPlayer.CanManageGridBy(msg.Creator)
            }
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
