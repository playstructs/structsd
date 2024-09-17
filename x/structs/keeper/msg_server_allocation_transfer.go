package keeper

import (
	"context"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationTransfer(goCtx context.Context, msg *types.MsgAllocationTransfer) (*types.MsgAllocationTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // Check permissions on the substation

	allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgAllocationTransferResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "allocation (%s) not found", msg.AllocationId)
	}

	if (allocation.Controller != msg.Creator) {
		return &types.MsgAllocationTransferResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "allocation (%s) not controller by transaction creator. Unable to transfer", msg.AllocationId)
	}

    if (allocation.DestinationId != "") {
    	return &types.MsgAllocationTransferResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "allocation (%s) must not be connected to a substation during a transfer", msg.AllocationId)
    }

    allocation.Controller = msg.Controller
	allocation, _ = k.SetAllocation(ctx, allocation)

	return &types.MsgAllocationTransferResponse{
		AllocationId: msg.AllocationId,
	}, nil

}
