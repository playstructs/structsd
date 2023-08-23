package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) AddressRevoke(goCtx context.Context, msg *types.MsgAddressRevoke) (*types.MsgAddressRevokeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))
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
