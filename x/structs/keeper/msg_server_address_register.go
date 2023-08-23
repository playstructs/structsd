package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) AddressRegister(goCtx context.Context, msg *types.MsgAddressRegister) (*types.MsgAddressRegisterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    player, playerFound := k.GetPlayer(ctx, msg.PlayerId)
    if (playerFound) {
        // TODO Add address proof signature verification
        k.AddressSetRegisterRequest(ctx, player, msg.Address)
    }

	return &types.MsgAddressRegisterResponse{}, nil
}
