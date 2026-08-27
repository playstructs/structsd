package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PermissionRevokeOnObject(goCtx context.Context, msg *types.MsgPermissionRevokeOnObject) (*types.MsgPermissionResponse, error) {
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
        return  &types.MsgPermissionResponse{},  types.NewPermissionError("player", player.GetPlayerId(), "object", msg.ObjectId, uint64(msg.Permissions), "permission_grant")
    }

    permissionErr := cc.PermissionCheck(permissionedObject, player, types.Permission(msg.Permissions))
    if permissionErr != nil {
        return  &types.MsgPermissionResponse{}, permissionErr
    }

    /* The target is a player id chosen by the transaction, and it becomes half
     * of a KV key. Resolve it, or any string at all is storable: the object
     * check above says nothing about the target, and PermissionCheck's owner
     * shortcut passes an owner acting on their own object whatever they name.
     *
     * This is the phantom-cache rule reached by a different road. The other
     * handlers hand a message id to cc.GetPlayer, which allocates a cache and
     * cannot say whether the player is real; these never resolved it at all and
     * fed it straight into the key.
     */
    if _, targetErr := cc.GetExistingPlayer(msg.PlayerId); targetErr != nil {
        return emptyResponse, targetErr
    }

    targetPlayerPermissionId := GetObjectPermissionIDBytes(msg.ObjectId, msg.PlayerId)
    cc.PermissionRemove(targetPlayerPermissionId, types.Permission(msg.Permissions))

	cc.CommitAll()
	return &types.MsgPermissionResponse{}, nil
}
