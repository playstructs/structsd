package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationAllocationPropose(goCtx context.Context, msg *types.MsgSubstationAllocationPropose) (*types.MsgSubstationAllocationProposeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgSubstationAllocationProposeResponse{}, nil
}
