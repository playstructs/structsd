package keeper

import (
	"context"
    math "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
    "strconv"
    sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) SubstationAllocationActivate(goCtx context.Context, msg *types.MsgSubstationAllocationActivate) (*types.MsgSubstationAllocationActivateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    if (!msg.Decision) {
        // TODO: Check permissions rather than just doin it
        k.RemoveAllocationProposal(ctx, msg.AllocationId)
        return &types.MsgSubstationAllocationActivateResponse{}, nil
    }

	proposal, AllocationProposalFound  := k.GetAllocationProposal(ctx, msg.AllocationId)
    if (!AllocationProposalFound){
        allocationProposalId := strconv.FormatUint(msg.AllocationId, 10)
        return &types.MsgSubstationAllocationActivateResponse{}, sdkerrors.Wrapf(types.ErrAllocationNotFound, "allocation proposal (%s) not found", allocationProposalId)
    }

    if (proposal.SourceType != types.ObjectType_substation) {
        return &types.MsgSubstationAllocationActivateResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceTypeMismatch, "allocation proposal type (%s) does not match substation", proposal.SourceType.String())
    }

	allocation := types.Allocation{
	    SourceType: types.ObjectType_substation,
	    SourceId: proposal.SourceId,
	    DestinationId: proposal.DestinationId,
	    Power: math.ZeroInt(),
	    TransmissionLoss: math.ZeroInt(),

	}

    sourceSubstation, sourceSubstationFound := k.GetSubstation(ctx, proposal.SourceId)
    if (!sourceSubstationFound){
        sourceId := strconv.FormatUint(proposal.SourceId, 10)
        return &types.MsgSubstationAllocationActivateResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceNotFound, "source (%s) used for allocation not found", allocation.SourceType.String() + "-" + sourceId)
    }

    destinationSubstation, destinationSubstationFound := k.GetSubstation(ctx, proposal.DestinationId)
    if (!destinationSubstationFound){
        destinationId := strconv.FormatUint(proposal.DestinationId, 10)
        return &types.MsgSubstationAllocationActivateResponse{}, sdkerrors.Wrapf(types.ErrSubstationNotFound, "destination substation (%s) used for allocation not found", destinationId)
    }



	allocation.SetPower(ctx, proposal)
    sourceSubstation.ApplyAllocationSource(allocation)
    destinationSubstation.ApplyAllocationDestination(allocation)


    _ = k.AppendAllocation(ctx, allocation)
    k.SetSubstation(ctx, sourceSubstation)
    k.SetSubstation(ctx, destinationSubstation)
    k.RemoveAllocationProposal(ctx, msg.AllocationId)

	return &types.MsgSubstationAllocationActivateResponse{}, nil
}
