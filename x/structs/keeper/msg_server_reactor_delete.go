package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) ReactorDelete(goCtx context.Context, msg *types.MsgReactorDelete) (*types.MsgReactorDeleteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgReactorDeleteResponse{}, nil
}
