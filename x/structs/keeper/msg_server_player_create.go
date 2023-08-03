package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlayerCreate(goCtx context.Context, msg *types.MsgPlayerCreate) (*types.MsgPlayerCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgPlayerCreateResponse{}, nil
}
