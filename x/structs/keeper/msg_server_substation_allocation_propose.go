package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)


/*
message MsgSubstationAllocationPropose {
  string creator    = 1;
  uint64 id         = 2;
  string sourceType = 3;
  uint64 sourceId   = 4;
  string power      = 5;
}
*/

func (k msgServer) SubstationAllocationPropose(goCtx context.Context, msg *types.MsgSubstationAllocationPropose) (*types.MsgSubstationAllocationProposeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	proposal := types.AllocationProposal{


	}

    k.AppendAllocationProposal(ctx, proposal)

	return &types.MsgSubstationAllocationProposeResponse{}, nil
}
