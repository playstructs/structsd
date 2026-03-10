package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationTransfer(goCtx context.Context, msg *types.MsgAllocationTransfer) (*types.MsgAllocationTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgAllocationTransferResponse{}, types.NewPlayerRequiredError(msg.Creator, "allocation_delete")
    }

    _, err = cc.GetPlayerByAddress(msg.Controller)
    if err != nil {
        return &types.MsgAllocationTransferResponse{}, types.NewPlayerRequiredError(msg.Controller, "allocation_delete")
    }


    // Check permissions on the substation
	allocation, allocationFound := cc.GetAllocation(msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgAllocationTransferResponse{}, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

    permissionErr := allocation.CanBeTransferBy(activePlayer)
    if permissionErr != nil {
        return &types.MsgAllocationTransferResponse{}, permissionErr
    }

    allocation.SetController(msg.Controller)

	cc.CommitAll()
	return &types.MsgAllocationTransferResponse{
		AllocationId: msg.AllocationId,
	}, nil

}
