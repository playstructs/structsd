package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PermissionSetOnObject(goCtx context.Context, msg *types.MsgPermissionSetOnObject) (*types.MsgPermissionResponse, error) {
    emptyResponse := &types.MsgPermissionResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)


    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

   var err error

    player, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
        return emptyResponse, err
    }

    permissionedObject := cc.GetPermissionedObject(msg.ObjectId)
    if permissionedObject == nil {
        return emptyResponse, types.NewPermissionError("player", player.GetPlayerId(), "object", msg.ObjectId, uint64(msg.Permissions), "permission_grant")
    }

    // This overwrites rather than adds, so the call destroys the target player's
    // existing grant on the object as well as writing msg.Permissions. Require
    // the caller to hold both, otherwise a narrowly granted player could strip
    // a broader grant off someone else.
    targetPlayerPermissionId := GetObjectPermissionIDBytes(msg.ObjectId, msg.PlayerId)
    requiredPermissions := types.Permission(msg.Permissions) | cc.GetPermissions(targetPlayerPermissionId)

    permissionErr := cc.PermissionCheck(permissionedObject, player, requiredPermissions)
    if permissionErr != nil {
        return emptyResponse, permissionErr
    }

    cc.SetPermissions(targetPlayerPermissionId, types.Permission(msg.Permissions))

	cc.CommitAll()
	return &types.MsgPermissionResponse{}, nil
}
