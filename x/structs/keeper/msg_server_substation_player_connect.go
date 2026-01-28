package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerConnect(goCtx context.Context, msg *types.MsgSubstationPlayerConnect) (*types.MsgSubstationPlayerConnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgSubstationPlayerConnectResponse{}, types.NewPlayerRequiredError(msg.Creator, "substation_player_connect")
    }

	targetPlayer, targetPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
    if (!targetPlayerFound) {
        return &types.MsgSubstationPlayerConnectResponse{}, types.NewObjectNotFoundError("player", msg.PlayerId)
    }

    substationObjectPermissionId := GetObjectPermissionIDBytes(msg.SubstationId, player.Id)
    // check that the calling player has substation permissions
    if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationPlayerConnectResponse{}, types.NewPermissionError("player", player.Id, "substation", msg.SubstationId, uint64(types.PermissionGrid), "player_connect")
    }

    if (player.Id != msg.PlayerId) {
        // check that the calling player has target player permissions
        playerObjectPermissionId := GetObjectPermissionIDBytes(msg.PlayerId, player.Id)
        if (!k.PermissionHasOneOf(ctx, playerObjectPermissionId, types.PermissionGrid)) {
            return &types.MsgSubstationPlayerConnectResponse{}, types.NewPermissionError("player", player.Id, "player", msg.PlayerId, uint64(types.PermissionGrid), "player_connect")
        }
    }

    // check that the account has energy management permissions
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)
    if(!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationPlayerConnectResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionGrid), "energy_management")
    }

    substation, sourceSubstationFound := k.GetSubstation(ctx, msg.SubstationId)
    if (!sourceSubstationFound) {
        return &types.MsgSubstationPlayerConnectResponse{}, types.NewObjectNotFoundError("substation", msg.SubstationId)
    }

	// connect to new substation
	// This call handles the disconnection from other substations as well
    k.SubstationConnectPlayer(ctx, substation, targetPlayer)

	return &types.MsgSubstationPlayerConnectResponse{}, nil
}
