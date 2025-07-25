package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) PermissionGrantOnObject(goCtx context.Context, msg *types.MsgPermissionGrantOnObject) (*types.MsgPermissionResponse, error) {
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

    if (player.Id != msg.PlayerId) {
        _, targetPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
        if (!targetPlayerFound) {
            return nil, err
        }
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has the Permissions permission for editing permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permissions)) {
        return &types.MsgPermissionResponse{}, sdkerrors.Wrapf(types.ErrPermissionAssociation, "Calling address (%s) has no permissions permission", msg.Creator)
    }

    // Make sure the calling player has the same permissions that are being applied to the other player
    playerPermissionId := GetObjectPermissionIDBytes(msg.ObjectId, player.Id)
    if (!k.PermissionHasAll(ctx, playerPermissionId, types.Permission(msg.Permissions))) {
        return &types.MsgPermissionResponse{}, sdkerrors.Wrapf(types.ErrGuildUpdate, "Calling player (%s) does not have the authority over the object", player.Id)
    }

    targetPlayerPermissionId := GetObjectPermissionIDBytes(msg.ObjectId, msg.PlayerId)
    k.PermissionAdd(ctx, targetPlayerPermissionId, types.Permission(msg.Permissions))

	return &types.MsgPermissionResponse{}, nil
}
