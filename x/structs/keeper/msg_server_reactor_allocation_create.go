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



    // Check to see if the Reactor has the Power available
    // Calling ReactorIncrementLoad will update the Memory store so the change has already been applied if successful
    //
    // Maybe this will change but currently a new allocation can't be created without the
    // available capacity to bring it online. In the future, we could allow for this and it would
    // blow up older allocations until it hits the threshold, but that feels overly destructive.
    newReactorLoad, incrementLoadError := k.ReactorIncrementLoad(ctx, proposal.SourceId, msg.Power)
    if incrementLoadError != nil {
        return nil, incrementLoadError
    }

	allocation.SetPower(ctx, msg.Power)
	allocation.SetOwner(ctx, msg.Owner)

    allocationId := k.AppendAllocation(ctx, allocation)

	return &types.MsgReactorAllocationCreateResponse{
        AllocationId: allocationId,
        NewReactorLoad: newReactorLoad,
    }, nil
}
