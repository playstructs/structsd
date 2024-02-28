package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) AddressRevoke(goCtx context.Context, msg *types.MsgAddressRevoke) (*types.MsgAddressRevokeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator), false)

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

    // Make sure the address calling this has Revoke permissions
    if (k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionRevoke))) {
        return &types.MsgAddressRevokeResponse{}, sdkerrors.Wrapf(types.ErrPermissionRevoke, "Calling address (%s) has no Revoke permissions ", msg.Creator)
    }


    if (playerFound) {
        // TODO Add address proof signature verification
        playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Address)
        if (playerIndex == player.Index) {
            addressClearPermissionId := GetAddressPermissionIDBytes(msg.Address)
            k.PermissionClearAll(ctx, addressClearPermissionId)
        }
    }

	return &types.MsgAddressRevokeResponse{}, nil
}
