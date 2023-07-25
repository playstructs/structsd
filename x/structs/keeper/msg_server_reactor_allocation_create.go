package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
    "strconv"
    sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) ReactorAllocationCreate(goCtx context.Context, msg *types.MsgReactorAllocationCreate) (*types.MsgReactorAllocationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	allocation := types.Allocation{
	    SourceType: types.ObjectType_reactor,
	    SourceId: msg.SourceId,
	    //DestinationId: 0,
	    Power: 0,
	    Locked: false,
	    Creator: msg.Creator,
	    Owner: msg.Creator,
	}

    sourceReactor, sourceReactorFound := k.GetReactor(ctx, proposal.SourceId)
    if (!sourceReactorFound){
        sourceId := strconv.FormatUint(proposal.SourceId, 10)
        return &types.MsgReactorAllocationActivateResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceNotFound, "source (%s) used for allocation not found", allocation.SourceType.String() + "-" + sourceId)
    }

    /*
    if (!sourceReactor.IsOnline()) {
        sourceId := strconv.FormatUint(proposal.SourceId, 10)
        return &types.MsgReactorAllocationActivateResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceNotOnline, "source (%s) used for allocation must be online to activate", allocation.SourceType.String() + "-" + sourceId)
    }
    */

    // Check to see if the Reactor has the Power available
    newReactorCapacity, decrementError := k.ReactorDecrementCapacity(ctx, msg.Power)
    if decrementError != nil {
        return nil, decrementError
    }

	allocation.SetPower(ctx, msg.Power)
	allocation.SetOwner(ctx, msg.Owner)

    allocationId := k.AppendAllocation(ctx, allocation)

	return &types.MsgReactorAllocationCreateResponse{
        AllocationId: allocationId,
        NewReactorCapacity: newReactorCapacity,
    }, nil
}
