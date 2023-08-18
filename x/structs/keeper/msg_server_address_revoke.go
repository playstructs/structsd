package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) AddressRevoke(goCtx context.Context, msg *types.MsgAddressRevoke) (*types.MsgAddressRevokeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgAddressRevokeResponse{}, nil
}
