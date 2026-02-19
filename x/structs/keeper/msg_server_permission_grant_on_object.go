package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PermissionGrantOnObject(goCtx context.Context, msg *types.MsgPermissionGrantOnObject) (*types.MsgPermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)


    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

   var err error

    if msg.Permissions == 0 {
        return &types.MsgPermissionResponse{}, types.NewParameterValidationError("permissions", 0, "below_minimum").WithRange(1, 0)
    }

    player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return  &types.MsgPermissionResponse{}, err
    }

    if (player.GetPlayerId() != msg.PlayerId) {
        _, err = cc.GetPlayer(msg.PlayerId)
        if err != nil {
            return  &types.MsgPermissionResponse{}, err
        }
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has the Permissions permission for editing permissions
    if (!cc.PermissionHasOneOf(addressPermissionId, types.Permissions)) {
        return &types.MsgPermissionResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.Permissions), "permission_edit")
    }

    // Make sure the calling player has the same permissions that are being applied to the other player
    playerPermissionId := GetObjectPermissionIDBytes(msg.ObjectId, player.GetPlayerId())
    if (!cc.PermissionHasAll(playerPermissionId, types.Permission(msg.Permissions))) {
        return &types.MsgPermissionResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "object", msg.ObjectId, uint64(msg.Permissions), "permission_grant")
    }

    targetPlayerPermissionId := GetObjectPermissionIDBytes(msg.ObjectId, msg.PlayerId)
    cc.PermissionAdd(targetPlayerPermissionId, types.Permission(msg.Permissions))

	cc.CommitAll()
	return &types.MsgPermissionResponse{}, nil
}
