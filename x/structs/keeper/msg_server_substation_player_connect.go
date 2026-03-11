package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerConnect(goCtx context.Context, msg *types.MsgSubstationPlayerConnect) (*types.MsgSubstationPlayerConnectResponse, error) {
    emptyResponse := &types.MsgSubstationPlayerConnectResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	callingPlayer, playerErr := cc.GetPlayerByAddress(msg.Creator)
    if playerErr != nil {
        return emptyResponse, playerErr
    }

	targetPlayer, _ := cc.GetPlayer(msg.PlayerId)
    if targetPlayer.CheckPlayer() != nil {
        return emptyResponse, targetPlayer.CheckPlayer()
    }

    substation := cc.GetSubstation(msg.SubstationId)
    if substation.CheckSubstation() != nil {
        return emptyResponse, types.NewObjectNotFoundError("substation", msg.SubstationId)
    }

    substationPermissionErr := substation.CanManageConnectionsBy(callingPlayer)
    if substationPermissionErr != nil {
        return emptyResponse, substationPermissionErr
    }

    permissionPlayerErr := targetPlayer.CanManageSubstationConnectionBy(callingPlayer)
    if permissionPlayerErr != nil {
        return emptyResponse, permissionPlayerErr
    }

    // connect to new substation
	// This call handles the disconnection from other substations as well
    targetPlayer.MigrateSubstation(substation.GetSubstationId())

	cc.CommitAll()
	return &types.MsgSubstationPlayerConnectResponse{}, nil
}
