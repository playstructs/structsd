package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationDelete(goCtx context.Context, msg *types.MsgSubstationDelete) (*types.MsgSubstationDeleteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	// Need all allocations in



	// Need all allocations out



	// Need all players connected


    k.RemoveSubstation(ctx, msg.SubstationId)

	return &types.MsgSubstationDeleteResponse{}, nil
}
