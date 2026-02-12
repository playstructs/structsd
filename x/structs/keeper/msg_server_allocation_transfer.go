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

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // Check permissions on the substation

	allocation, allocationFound := cc.GetAllocation(msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgAllocationTransferResponse{}, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

    // TODO Allow for other addresses from a player to control it too
	if (allocation.GetAllocation().Controller != msg.Creator) {
		return &types.MsgAllocationTransferResponse{}, types.NewPermissionError("address", msg.Creator, "allocation", msg.AllocationId, uint64(types.PermissionAssets), "allocation_transfer")
	}

    if (allocation.GetAllocation().DestinationId != "") {
    	return &types.MsgAllocationTransferResponse{}, types.NewAllocationError(msg.AllocationId, "connected").WithDestination(allocation.GetAllocation().DestinationId)
    }

    allocation.SetController(msg.Controller)

	return &types.MsgAllocationTransferResponse{
		AllocationId: msg.AllocationId,
	}, nil

}
