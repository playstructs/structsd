package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerDisconnect(goCtx context.Context, msg *types.MsgSubstationPlayerDisconnect) (*types.MsgSubstationPlayerDisconnectResponse, error) {
    emptyResponse := &types.MsgSubstationPlayerDisconnectResponse{}
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

    substation := cc.GetSubstation(targetPlayer.GetSubstationId())
    if substation.CheckSubstation() != nil {
        return emptyResponse, types.NewObjectNotFoundError("substation", targetPlayer.GetSubstationId())
    }

    permissionPlayerErr := targetPlayer.CanManageSubstationConnectionBy(callingPlayer)
    if permissionPlayerErr != nil {
        substationPermissionErr := substation.CanManageConnectionsBy(callingPlayer)
        if substationPermissionErr != nil {
            return emptyResponse, substationPermissionErr
        }
    }

    // connect to new substation
	// This call handles the disconnection from other substations as well
    targetPlayer.DisconnectSubstation()

	cc.CommitAll()
	return &types.MsgSubstationPlayerDisconnectResponse{}, nil
}
