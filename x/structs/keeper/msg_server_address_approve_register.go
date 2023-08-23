package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) AddressApproveRegister(goCtx context.Context, msg *types.MsgAddressApproveRegister) (*types.MsgAddressRegisterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))
    if (playerFound) {
        if (msg.Approve) {
            // TODO permission checking to see if this specific account has the ability to grant these permissions

            k.AddressApproveRegisterRequest(ctx, player, msg.Address, types.AddressPermissionAll)
        } else {
            k.AddressDenyRegisterRequest(ctx, player, msg.Address)
        }
    }

	return &types.MsgAddressRegisterResponse{}, nil
}
