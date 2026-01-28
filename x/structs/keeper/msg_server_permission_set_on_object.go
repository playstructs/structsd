package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PermissionSetOnObject(goCtx context.Context, msg *types.MsgPermissionSetOnObject) (*types.MsgPermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

   var err error

    player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return nil, err
    }

    if (player.Id != msg.PlayerId) {
        _, targetPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
        if (!targetPlayerFound) {
            return nil, err
        }
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has the Permissions permission for editing permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permissions)) {
        return &types.MsgPermissionResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.Permissions), "permission_edit")
    }

    // Make sure the calling player has the same permissions that are being applied to the other player
    playerPermissionId := GetObjectPermissionIDBytes(msg.ObjectId, player.Id)
    if (!k.PermissionHasAll(ctx, playerPermissionId, types.Permission(msg.Permissions))) {
        return &types.MsgPermissionResponse{}, types.NewPermissionError("player", player.Id, "object", msg.ObjectId, uint64(msg.Permissions), "permission_set")
    }

    targetPlayerPermissionId := GetObjectPermissionIDBytes(msg.ObjectId, msg.PlayerId)
    k.SetPermissionsByBytes(ctx, targetPlayerPermissionId, types.Permission(msg.Permissions))

	return &types.MsgPermissionResponse{}, nil
}
