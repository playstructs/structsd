package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) ReactorSetup(goCtx context.Context, msg *types.MsgReactorSetup) (*types.MsgReactorSetupResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgReactorSetupResponse{}, nil
}
