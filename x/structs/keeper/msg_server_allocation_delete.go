package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationDelete(goCtx context.Context, msg *types.MsgAllocationDelete) (*types.MsgAllocationDeleteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgAllocationDeleteResponse{}, types.NewPlayerRequiredError(msg.Creator, "allocation_delete")
    }

	allocation, allocationFound := cc.GetAllocation(msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgAllocationDeleteResponse{}, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

    permissionErr := allocation.CanSourceDetailsBeUpdatedBy(activePlayer)
    if permissionErr != nil {
        return &types.MsgAllocationDeleteResponse{}, permissionErr
    }

    if (allocation.GetAllocation().Type != types.AllocationType_dynamic) {
        return &types.MsgAllocationDeleteResponse{}, types.NewAllocationError(allocation.GetAllocation().SourceObjectId, "immutable_type").WithFieldChange("type", allocation.GetAllocation().Type.String(), "dynamic")
    }

    allocation.Destroy()

	cc.CommitAll()
	return &types.MsgAllocationDeleteResponse{
		AllocationId: msg.AllocationId,
	}, nil

}
