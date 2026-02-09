package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationTransfer(goCtx context.Context, msg *types.MsgAllocationTransfer) (*types.MsgAllocationTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()
	_ = cc

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // Check permissions on the substation

	allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgAllocationTransferResponse{}, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

	if (allocation.Controller != msg.Creator) {
		return &types.MsgAllocationTransferResponse{}, types.NewPermissionError("address", msg.Creator, "allocation", msg.AllocationId, uint64(types.PermissionAssets), "allocation_transfer")
	}

    if (allocation.DestinationId != "") {
    	return &types.MsgAllocationTransferResponse{}, types.NewAllocationError(msg.AllocationId, "connected").WithDestination(allocation.DestinationId)
    }

    allocation.Controller = msg.Controller
	allocation, _ = k.SetAllocationOnly(ctx, allocation)

	return &types.MsgAllocationTransferResponse{
		AllocationId: msg.AllocationId,
	}, nil

}
