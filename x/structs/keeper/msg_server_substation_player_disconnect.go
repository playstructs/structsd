package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerDisconnect(goCtx context.Context, msg *types.MsgSubstationPlayerDisconnect) (*types.MsgSubstationPlayerDisconnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, _ := cc.GetPlayer(msg.Creator)
    if player.CheckPlayer() != nil {
        return &types.MsgSubstationPlayerDisconnectResponse{}, player.CheckPlayer()
    }

	targetPlayer, _ := cc.GetPlayer(msg.PlayerId)
    if targetPlayer.CheckPlayer() != nil {
        return &types.MsgSubstationPlayerDisconnectResponse{}, targetPlayer.CheckPlayer()
    }

    substation := cc.GetSubstation(targetPlayer.GetSubstationId())
    if substation.CheckSubstation() != nil {
        return &types.MsgSubstationPlayerDisconnectResponse{}, types.NewObjectNotFoundError("substation", targetPlayer.GetSubstationId())
    }

    permissionSubstationErr := substation.CanManagePlayerConnections(player)
    if permissionSubstationErr != nil {
        // It might be ok if they don't have permissions on the substation
        // as long as they have permissions on themselves.
        permissionPlayerErr := targetPlayer.CanManageGridBy(msg.Creator)
        if permissionPlayerErr != nil {
            return &types.MsgSubstationPlayerDisconnectResponse{}, permissionPlayerErr
        }
    }

    // connect to new substation
	// This call handles the disconnection from other substations as well
    targetPlayer.DisconnectSubstation()

	return &types.MsgSubstationPlayerDisconnectResponse{}, nil
}
