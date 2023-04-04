package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) ReactorAllocationActivate(goCtx context.Context, msg *types.MsgReactorAllocationActivate) (*types.MsgReactorAllocationActivateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgReactorAllocationActivateResponse{}, nil
}
