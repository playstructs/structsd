package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
    "strconv"
    sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)


func (k msgServer) SubstationAllocationConnect(goCtx context.Context, msg *types.MsgSubstationAllocationConnect) (*types.MsgSubstationAllocationConnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}


	allocation, allocationFound  := k.GetAllocation(ctx, msg.AllocationId)
    if (!allocationFound){
        allocationId := strconv.FormatUint(msg.AllocationId, 10)
        return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationNotFound, "allocation (%s) not found", allocationId)
    }


	substation, substationFound  := k.GetSubstation(ctx, msg.DestinationId)
    if (!substationFound){
        substationId := strconv.FormatUint(allocation.DestinationId, 10)
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrSubstationNotFound, "destination substation (%s) not found", substationId)
    }

    if (substation.Id == msg.DestinationId){
        substationId := strconv.FormatUint(allocation.DestinationId, 10)
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationConnectionChangeImpossible, "destination substation (%s) cannot change to same destination", substationId)
    }


    // Check to see if there is already a destination Substation using this.
    // Disconnect it if so
    if (substation.DestinationId > 0) {
        newEnergy := k.SubstationDecrementEnergy(ctx, substation.DestinationId, allocation.Power)
        k.CascadeSubstationAllocationFailure(ctx, substation)
    }


    newEnergy := k.SubstationIncrementEnergy(ctx, substation.Id, allocation.Power)

    allocation.Connect(ctx, msg.DestinationId)
    k.SetAllocation(ctx, allocation)

	return &types.MsgSubstationAllocationConnectResponse{}, nil
}
