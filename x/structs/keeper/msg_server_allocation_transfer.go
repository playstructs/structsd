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

    activePlayer, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
        return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "allocation_transfer")
    }

    // The destination must exist. Nothing else checks it: CanBeTransferBy asks
    // only whether the caller may transfer, so an unchecked id parks the
    // allocation on a player who can never sign for it, and writes a permission
    // row keyed to them.
    if _, err = cc.GetExistingPlayer(msg.Controller); err != nil {
        return emptyResponse, err
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

    // Remove the one bit, never the row. AllocationCreate defaults the controller
    // to the creator, so the outgoing row usually also carries the creator's
    // PermUpdate | PermDelete, and PermDelete is the documented fallback that
    // AllocationDelete tries when the source-side check fails. Clearing the row
    // would take a creator's control of their own allocation away with the
    // transfer.
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
