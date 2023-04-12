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

    if (proposal.SourceType != types.ObjectType_reactor) {
        return &types.MsgReactorAllocationActivateResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceTypeMismatch, "allocation proposal type (%s) does not match reactor", proposal.SourceType.String())
    }


	allocation := types.Allocation{
	    SourceType: types.ObjectType_reactor,
	    SourceId: proposal.SourceId,
	    DestinationId: proposal.DestinationId,
	    Power: math.ZeroInt(),
	    TransmissionLoss: math.ZeroInt(),

	}

    sourceReactor, sourceReactorFound := k.GetReactor(ctx, proposal.SourceId)
    if (!sourceReactorFound){
        sourceId := strconv.FormatUint(proposal.SourceId, 10)
        return &types.MsgReactorAllocationActivateResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceNotFound, "source (%s) used for allocation not found", allocation.SourceType.String() + "-" + sourceId)
    }

    destinationSubstation, destinationSubstationFound := k.GetSubstation(ctx, proposal.DestinationId)
    if (!destinationSubstationFound){
        destinationId := strconv.FormatUint(proposal.DestinationId, 10)
        return &types.MsgReactorAllocationActivateResponse{}, sdkerrors.Wrapf(types.ErrSubstationNotFound, "destination substation (%s) used for allocation not found", destinationId)
    }



	allocation.SetPower(ctx, proposal)
    sourceReactor.ApplyAllocationSource(allocation)
    destinationSubstation.ApplyAllocationDestination(allocation)


    _ = k.AppendAllocation(ctx, allocation)
    k.SetReactor(ctx, sourceReactor)
    k.SetSubstation(ctx, destinationSubstation)
    k.RemoveAllocationProposal(ctx, msg.AllocationId)

	return &types.MsgReactorAllocationActivateResponse{}, nil
}
