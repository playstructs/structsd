package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PermissionSetOnAddress(goCtx context.Context, msg *types.MsgPermissionSetOnAddress) (*types.MsgPermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    var err error

    player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return  &types.MsgPermissionResponse{}, err
    }

    targetPlayer, err := cc.GetPlayerByAddress(msg.Address)
    if err != nil {
         return  &types.MsgPermissionResponse{}, err
     }

     if (targetPlayer.GetPlayerId() != player.GetPlayerId()) {
        return  &types.MsgPermissionResponse{}, types.NewObjectNotFoundError("player", targetPlayer.GetPlayerId()) // Can only set address permissions on your own player
     }


    // Make sure the calling address has enough permissions to apply to another address
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    if (!cc.PermissionHasAll(addressPermissionId, types.Permission(msg.Permissions) | types.Permissions)) {
        return &types.MsgPermissionResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(msg.Permissions), "permission_set")
    }

    targetAddressPermissionId := GetAddressPermissionIDBytes(msg.Address)
    cc.SetPermissions(targetAddressPermissionId, msg.Permissions)

	return &types.MsgPermissionResponse{}, nil
}
