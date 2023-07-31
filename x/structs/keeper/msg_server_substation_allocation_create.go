package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
    "strconv"
    sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) SubstationAllocationCreate(goCtx context.Context, msg *types.MsgSubstationAllocationCreate) (*types.MsgAllocationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	allocation := types.Allocation{
	    SourceType: types.ObjectType_substation,
	    SourceId: msg.SourceId,
	    //DestinationId: 0,
	    Power: 0,
	    Locked: false,
	    Creator: msg.Creator,
	    Controller: msg.Creator,
	}

    _, sourceSubstationFound := k.GetSubstation(ctx, msg.SourceId)
    if (!sourceSubstationFound){
        sourceId := strconv.FormatUint(msg.SourceId, 10)
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceNotFound, "source (%s) used for allocation not found", allocation.SourceType.String() + "-" + sourceId)
    }



    // Check to see if the Substation has the Power available
    // Calling SubstationIncrementAllocationLoad will update the Memory store so the change has already been applied if successful
    //
    // Maybe this will change but currently a new allocation can't be created without the
    // available capacity to bring it online. In the future, we could allow for this and it would
    // blow up older allocations until it hits the threshold, but that feels overly destructive.
    _, incrementLoadError := k.SubstationIncrementAllocationLoad(ctx, msg.SourceId, msg.Power)
    if incrementLoadError != nil {
        return nil, incrementLoadError
    }

	allocation.SetPower(ctx, msg.Power)
	allocation.SetController(ctx, msg.Controller)

    allocationId := k.AppendAllocation(ctx, allocation)

	return &types.MsgAllocationCreateResponse{
        AllocationId: allocationId,
    }, nil
}
