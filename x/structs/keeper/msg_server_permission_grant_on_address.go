package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) PermissionGrantOnAddress(goCtx context.Context, msg *types.MsgPermissionGrantOnAddress) (*types.MsgPermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    var err error

    if msg.Permissions == 0 {
        return &types.MsgPermissionResponse{}, sdkerrors.Wrapf(types.ErrPermission, "Cannot Grant 0")
    }

    player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return nil, err
    }

    targetPlayer, targetPlayerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Address))
    if (!targetPlayerFound) {
         return nil, err
     }

     if (targetPlayer.Id != player.Id) {
        return nil, err // Can only set address permissions on your own player
     }


    // Make sure the calling address has enough permissions to apply to another address
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    if (!k.PermissionHasAll(ctx, addressPermissionId, types.Permission(msg.Permissions) | types.Permissions)) {
        return &types.MsgPermissionResponse{}, sdkerrors.Wrapf(types.ErrPermission, "Calling address (%s) does not have the permissions needed to grant this level", msg.Creator)
    }

    targetAddressPermissionId := GetAddressPermissionIDBytes(msg.Address)
    k.PermissionAdd(ctx, targetAddressPermissionId, types.Permission(msg.Permissions))

	return &types.MsgPermissionResponse{}, nil
}
