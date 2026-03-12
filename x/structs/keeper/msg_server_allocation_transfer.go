package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationTransfer(goCtx context.Context, msg *types.MsgAllocationTransfer) (*types.MsgAllocationTransferResponse, error) {
    emptyResponse := &types.MsgAllocationTransferResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "allocation_transfer")
    }

    _, err = cc.GetPlayerByAddress(msg.Controller)
    if err != nil {
        return emptyResponse, types.NewPlayerRequiredError(msg.Controller, "allocation_transfer")
    }


    // Check permissions on the substation
	allocation, allocationFound := cc.GetAllocation(msg.AllocationId)
	if (!allocationFound) {
		return emptyResponse, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

    permissionErr := allocation.CanBeTransferBy(activePlayer)
    if permissionErr != nil {
        return emptyResponse, permissionErr
    }

    oldControllerPermissionId := GetObjectPermissionIDBytes(allocation.ID(), allocation.GetAllocation().Controller)
    cc.PermissionRemove(oldControllerPermissionId, types.PermAllocationConnection)

    newControllerPermissionId := GetObjectPermissionIDBytes(allocation.ID(), msg.Controller)
    cc.SetPermissions(newControllerPermissionId, types.PermAllocationConnection)

    allocation.SetController(msg.Controller)

	cc.CommitAll()
	return &types.MsgAllocationTransferResponse{
		AllocationId: msg.AllocationId,
	}, nil

}
