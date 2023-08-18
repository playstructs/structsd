package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strconv"
	"structs/x/structs/types"
)

func (k msgServer) SubstationAllocationConnect(goCtx context.Context, msg *types.MsgSubstationAllocationConnect) (*types.MsgSubstationAllocationConnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId)
	if !allocationFound {
		allocationId := strconv.FormatUint(msg.AllocationId, 10)
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationNotFound, "allocation (%s) not found", allocationId)
	}

	substation, substationFound := k.GetSubstation(ctx, msg.DestinationSubstationId, false)
	if !substationFound {
		substationId := strconv.FormatUint(allocation.DestinationId, 10)
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrSubstationNotFound, "destination substation (%s) not found", substationId)
	}

	if substation.Id == allocation.DestinationId {
		substationId := strconv.FormatUint(allocation.DestinationId, 10)
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationConnectionChangeImpossible, "destination substation (%s) cannot change to same destination", substationId)
	}

    k.SubstationConnectAllocation(ctx, substation, allocation)

	return &types.MsgSubstationAllocationConnectResponse{}, nil
}
