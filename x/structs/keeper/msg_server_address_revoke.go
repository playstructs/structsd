package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) AddressRevoke(goCtx context.Context, msg *types.MsgAddressRevoke) (*types.MsgAddressRevokeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))


    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)

    // Make sure the address calling this has Revoke permissions
    if (playerPermissions&types.AddressPermissionRevoke != 0) {
        // TODO permission error
        return &types.MsgAddressRevokeResponse{}, sdkerrors.Wrapf(types.ErrPermissionRevoke, "Calling address (%s) has no Revoke permissions ", msg.Creator)
    }


    if (playerFound) {
        // TODO Add address proof signature verification
        playerId := k.GetPlayerIdFromAddress(ctx, msg.Address)
        if (playerId == player.Id) {
            // TODO check permissions that the specific address has revoke capabilities
            k.AddressPermissionClearAll(ctx, msg.Address)
        }
    }

	return &types.MsgAddressRevokeResponse{}, nil
}
