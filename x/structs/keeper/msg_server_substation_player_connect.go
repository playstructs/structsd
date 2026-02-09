package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerConnect(goCtx context.Context, msg *types.MsgSubstationPlayerConnect) (*types.MsgSubstationPlayerConnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgSubstationPlayerConnectResponse{}, err
    }

	targetPlayer, err := cc.GetPlayer(msg.PlayerId)
    if err != nil {
        return &types.MsgSubstationPlayerConnectResponse{}, err
    }
    if !targetPlayer.LoadPlayer() {
        return &types.MsgSubstationPlayerConnectResponse{}, types.NewObjectNotFoundError("player", msg.PlayerId)
    }

    substationObjectPermissionId := GetObjectPermissionIDBytes(msg.SubstationId, player.GetPlayerId())
    // check that the calling player has substation permissions
    if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationPlayerConnectResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "substation", msg.SubstationId, uint64(types.PermissionGrid), "player_connect")
    }

    if (player.GetPlayerId() != msg.PlayerId) {
        // check that the calling player has target player permissions
        playerObjectPermissionId := GetObjectPermissionIDBytes(msg.PlayerId, player.GetPlayerId())
        if (!k.PermissionHasOneOf(ctx, playerObjectPermissionId, types.PermissionGrid)) {
            return &types.MsgSubstationPlayerConnectResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "player", msg.PlayerId, uint64(types.PermissionGrid), "player_connect")
        }
    }

    // check that the account has energy management permissions
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)
    if(!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationPlayerConnectResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionGrid), "energy_management")
    }

    substation := cc.GetSubstation(msg.SubstationId)
    if (!substation.LoadSubstation()) {
        return &types.MsgSubstationPlayerConnectResponse{}, types.NewObjectNotFoundError("substation", msg.SubstationId)
    }

	// connect to new substation
	// This call handles the disconnection from other substations as well
    targetPlayer.MigrateSubstation(substation.GetSubstationId())

	return &types.MsgSubstationPlayerConnectResponse{}, nil
}
