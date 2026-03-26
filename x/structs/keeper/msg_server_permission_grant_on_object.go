package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PermissionGrantOnObject(goCtx context.Context, msg *types.MsgPermissionGrantOnObject) (*types.MsgPermissionResponse, error) {
    emptyResponse := &types.MsgPermissionResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)


    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

   var err error

    if msg.Permissions == 0 {
        return emptyResponse, types.NewParameterValidationError("permissions", 0, "below_minimum").WithRange(1, 0)
    }

    player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return emptyResponse, err
    }

    permissionedObject := cc.GetPermissionedObject(msg.ObjectId)
    if permissionedObject == nil {
        return emptyResponse, types.NewPermissionError("player", player.GetPlayerId(), "object", msg.ObjectId, uint64(msg.Permissions), "permission_grant")
    }

    permissionErr := cc.PermissionCheck(permissionedObject, player, types.Permission(msg.Permissions))
    if permissionErr != nil {
        return emptyResponse, permissionErr
    }

    targetPlayerPermissionId := GetObjectPermissionIDBytes(msg.ObjectId, msg.PlayerId)
    cc.PermissionAdd(targetPlayerPermissionId, types.Permission(msg.Permissions))

	cc.CommitAll()
	return &types.MsgPermissionResponse{}, nil
}
