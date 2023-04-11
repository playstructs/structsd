package keeper

import (
	"context"
    math "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) ReactorAllocationActivate(goCtx context.Context, msg *types.MsgReactorAllocationActivate) (*types.MsgReactorAllocationActivateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    if (!msg.Decision) {
        // TODO: Check permissions rather than just doin it
        k.RemoveAllocationProposal(ctx, msg.AllocationId)
        return &types.MsgReactorAllocationActivateResponse{}, nil
    }

	proposal, _  := k.GetAllocationProposal(ctx, msg.AllocationId)


	allocation := types.Allocation{
	    SourceType: types.ObjectType_reactor,
	    SourceId: proposal.SourceId,
	    DestinationId: proposal.DestinationId,
	    Power: math.NewIntFromUint64(0),
	    TransmissionLoss: math.NewIntFromUint64(0),

	}


    // Failing ://
	allocation.SetPower(ctx, proposal)

    _ = k.AppendAllocation(ctx, allocation)
    k.RemoveAllocationProposal(ctx, msg.AllocationId)

	return &types.MsgReactorAllocationActivateResponse{}, nil
}
