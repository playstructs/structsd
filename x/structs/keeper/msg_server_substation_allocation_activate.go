package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationAllocationActivate(goCtx context.Context, msg *types.MsgSubstationAllocationActivate) (*types.MsgSubstationAllocationActivateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgSubstationAllocationActivateResponse{}, nil
}
