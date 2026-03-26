package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationUpdate(goCtx context.Context, msg *types.MsgAllocationUpdate) (*types.MsgAllocationUpdateResponse, error) {
    emptyResponse := &types.MsgAllocationUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "allocation_update")
    }

	allocation, allocationFound := cc.GetAllocation(msg.AllocationId)
	if (!allocationFound) {
		return emptyResponse, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

    permissionErr := allocation.CanSourceDetailsBeUpdatedBy(activePlayer)
    if permissionErr != nil {
        return emptyResponse, permissionErr
    }

    _, setErr := allocation.SetDynamicPower(msg.Power)
    if setErr != nil {
        return emptyResponse, setErr
    }

	cc.CommitAll()
	return &types.MsgAllocationUpdateResponse{
		AllocationId: msg.AllocationId,
	}, err

}
