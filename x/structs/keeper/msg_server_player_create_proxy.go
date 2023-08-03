package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlayerCreateProxy(goCtx context.Context, msg *types.MsgPlayerCreateProxy) (*types.MsgPlayerCreateProxyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgPlayerCreateProxyResponse{}, nil
}
