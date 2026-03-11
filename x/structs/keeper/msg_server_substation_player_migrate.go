package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerMigrate(goCtx context.Context, msg *types.MsgSubstationPlayerMigrate) (*types.MsgSubstationPlayerMigrateResponse, error) {
    emptyResponse := &types.MsgSubstationPlayerMigrateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	callingPlayer, playerErr := cc.GetPlayerByAddress(msg.Creator)
    if playerErr != nil {
        return emptyResponse, playerErr
    }

    substation := cc.GetSubstation(msg.SubstationId)
    if substation.CheckSubstation() != nil {
        return emptyResponse, types.NewObjectNotFoundError("substation", msg.SubstationId)
    }

    substationPermissionErr := substation.CanManageConnectionsBy(callingPlayer)
    if substationPermissionErr != nil {
        return emptyResponse, substationPermissionErr
    }

    var targetPlayers []*PlayerCache
    for _, targetPlayerId := range msg.PlayerId {

        targetPlayer, err := cc.GetPlayer(targetPlayerId)
        if err != nil {
            return emptyResponse, err
        }
        if targetPlayer.CheckPlayer() != nil {
            return emptyResponse, types.NewObjectNotFoundError("player", targetPlayerId)
        }

        // check permissions
        permissionPlayerErr := targetPlayer.CanManageSubstationConnectionBy(callingPlayer)
        if permissionPlayerErr != nil {
            return emptyResponse, permissionPlayerErr
        }

        targetPlayers = append(targetPlayers, targetPlayer)
    }


    for _, migratePlayer := range targetPlayers {
        // connect to new substation
    	// This call handles the disconnection from other substations as well
        migratePlayer.MigrateSubstation(substation.GetSubstationId())
    }

	cc.CommitAll()
	return &types.MsgSubstationPlayerMigrateResponse{}, nil
}
