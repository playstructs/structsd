package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PermissionGrantOnAddress(goCtx context.Context, msg *types.MsgPermissionGrantOnAddress) (*types.MsgPermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    var err error

    if msg.Permissions == 0 {
        return &types.MsgPermissionResponse{}, types.NewParameterValidationError("permissions", 0, "below_minimum").WithRange(1, 0)
    }

    player, _ := cc.GetPlayerByAddress(msg.Creator)
    err = player.CheckPlayer()
    if err != nil {
        return  &types.MsgPermissionResponse{}, err
    }

    targetPlayer, _ := cc.GetPlayerByAddress(msg.Address)
    err = targetPlayer.CheckPlayer()
    if err != nil {
         return  &types.MsgPermissionResponse{}, err
     }

     if (targetPlayer.GetPlayerId() != player.GetPlayerId()) {
        return  &types.MsgPermissionResponse{}, types.NewObjectNotFoundError("player", targetPlayer.GetPlayerId()) // Can only set address permissions on your own player
     }

    // Make sure the calling address has enough permissions to apply to another address
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    if (!cc.PermissionHasAll(addressPermissionId, types.Permission(msg.Permissions) | types.Permissions)) {
        return &types.MsgPermissionResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(msg.Permissions), "permission_grant")
    }

    targetAddressPermissionId := GetAddressPermissionIDBytes(msg.Address)
    cc.PermissionAdd(targetAddressPermissionId, types.Permission(msg.Permissions))

	return &types.MsgPermissionResponse{}, nil
}
