package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) AddressRegister(goCtx context.Context, msg *types.MsgAddressRegister) (*types.MsgAddressRegisterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgAddressRegisterResponse{}, nil
}
