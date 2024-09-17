package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) AddressRevoke(goCtx context.Context, msg *types.MsgAddressRevoke) (*types.MsgAddressRevokeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

    // Make sure the address calling this has Revoke permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionDelete)) {
        return &types.MsgAddressRevokeResponse{}, sdkerrors.Wrapf(types.ErrPermissionRevoke, "Calling address (%s) has no Revoke permissions ", msg.Creator)
    }


    if (playerFound) {
        playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Address)
        if (playerIndex == player.Index) {
            addressClearPermissionId := GetAddressPermissionIDBytes(msg.Address)
            k.PermissionClearAll(ctx, addressClearPermissionId)

            k.RevokePlayerIndexForAddress(ctx, msg.Address, playerIndex)

        }
    }

	return &types.MsgAddressRevokeResponse{}, nil
}
