package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationUpdate(goCtx context.Context, msg *types.MsgAllocationUpdate) (*types.MsgAllocationUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgAllocationUpdateResponse{}, types.NewPlayerRequiredError(msg.Creator, "allocation_update")
    }

	allocation, allocationFound := cc.GetAllocation(msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgAllocationUpdateResponse{}, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

    permissionErr := allocation.CanSourceDetailsBeUpdatedBy(activePlayer)
    if permissionErr != nil {
        return &types.MsgAllocationUpdateResponse{}, permissionErr
    }

    setErr := allocation.SetDynamicPower(msg.Power)
    if setErr != nil {
        return &types.MsgAllocationUpdateResponse{}, setErr
    }

	cc.CommitAll()
	return &types.MsgAllocationUpdateResponse{
		AllocationId: msg.AllocationId,
	}, err

}
