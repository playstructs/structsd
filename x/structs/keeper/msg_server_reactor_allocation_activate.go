package keeper

import (
	"context"
    math "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
    "strconv"
    sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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

	proposal, AllocationProposalFound  := k.GetAllocationProposal(ctx, msg.AllocationId)
    if (!AllocationProposalFound){
        allocationProposalId := strconv.FormatUint(msg.AllocationId, 10)
        return &types.MsgReactorAllocationActivateResponse{}, sdkerrors.Wrapf(types.ErrAllocationNotFound, "allocation proposal (%s) not found", allocationProposalId)
    }

	allocation := types.Allocation{
	    SourceType: types.ObjectType_reactor,
	    SourceId: proposal.SourceId,
	    DestinationId: proposal.DestinationId,
	    Power: math.NewIntFromUint64(0),
	    TransmissionLoss: math.NewIntFromUint64(0),

	}

    sourceReactor, sourceReactorFound := k.GetReactor(ctx, proposal.SourceId)
    if (!sourceReactorFound){
        sourceId := strconv.FormatUint(allocation.SourceId, 10)
        return &types.MsgReactorAllocationActivateResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceNotFound, "source (%s) used for allocation not found", allocation.SourceType.String() + "-" + sourceId)
    }


	allocation.SetPower(ctx, proposal)
    sourceReactor.ApplyAllocation(allocation)


    _ = k.AppendAllocation(ctx, allocation)
    k.SetReactor(ctx, sourceReactor)
    k.RemoveAllocationProposal(ctx, msg.AllocationId)

	return &types.MsgReactorAllocationActivateResponse{}, nil
}
