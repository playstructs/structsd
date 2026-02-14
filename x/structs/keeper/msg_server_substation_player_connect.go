package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerConnect(goCtx context.Context, msg *types.MsgSubstationPlayerConnect) (*types.MsgSubstationPlayerConnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, playerErr := cc.GetPlayerByAddress(msg.Creator)
    if playerErr != nil {
        return &types.MsgSubstationPlayerConnectResponse{}, playerErr
    }

	targetPlayer, _ := cc.GetPlayer(msg.PlayerId)
    if targetPlayer.CheckPlayer() != nil {
        return &types.MsgSubstationPlayerConnectResponse{}, targetPlayer.CheckPlayer()
    }

    substation := cc.GetSubstation(msg.SubstationId)
    if substation.CheckSubstation() != nil {
        return &types.MsgSubstationPlayerConnectResponse{}, types.NewObjectNotFoundError("substation", msg.SubstationId)
    }

    permissionSubstationErr := substation.CanManagePlayerConnections(player)
    if permissionSubstationErr != nil {
        return &types.MsgSubstationPlayerConnectResponse{}, permissionSubstationErr
    }

    permissionPlayerErr := targetPlayer.CanManageGridBy(msg.Creator)
    if permissionPlayerErr != nil {
        return &types.MsgSubstationPlayerConnectResponse{}, permissionPlayerErr
    }

    // connect to new substation
	// This call handles the disconnection from other substations as well
    targetPlayer.MigrateSubstation(substation.GetSubstationId())

	cc.CommitAll()
	return &types.MsgSubstationPlayerConnectResponse{}, nil
}
