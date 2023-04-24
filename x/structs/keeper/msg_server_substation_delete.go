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


    /*
     * This is going to start out very inefficient. We'll need to tackle
     * ways to improve these types of graph traversal
     */

	// Need all allocations in



	// Need all allocations out



	// Need all players connected


    k.RemoveSubstation(ctx, msg.SubstationId)

	return &types.MsgSubstationDeleteResponse{}, nil
}
