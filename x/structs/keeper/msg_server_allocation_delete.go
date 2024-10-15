package keeper

import (
	"context"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationDelete(goCtx context.Context, msg *types.MsgAllocationDelete) (*types.MsgAllocationDeleteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	allocation, allocationFound := k.GetAllocation(ctx,  msg.AllocationSourceId, msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgAllocationDeleteResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "allocation (%s) not found", msg.AllocationId)
	}

    player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgAllocationDeleteResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Could not perform allocation action with non-player address (%s)", msg.Creator)
    }

    sourceObjectPermissionId := GetObjectPermissionIDBytes(allocation.SourceObjectId, player.Id)
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

    // Ignore the one case where it's a player creating an allocation on themselves.
    // Surely that doesn't need a lookup.
    if (player.Id != allocation.SourceObjectId) {
        // check that the player has permissions
        if (!k.PermissionHasOneOf(ctx, sourceObjectPermissionId, types.PermissionAssets)) {
            return &types.MsgAllocationDeleteResponse{}, sdkerrors.Wrapf(types.ErrPermissionAllocation, "Calling player (%s) has no Allocation permissions on source (%s) ", player.Id, allocation.SourceObjectId)
        }
    }

    // check that the account has energy management permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.PermissionAssets))) {
        return &types.MsgAllocationDeleteResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }


    if (allocation.Type != types.AllocationType_dynamic) {
        return &types.MsgAllocationDeleteResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Allocation Type must be Dynamic for deleting ")
    }

    k.DestroyAllocation(ctx, msg.AllocationSourceId, msg.AllocationId)

	return &types.MsgAllocationDeleteResponse{
		AllocationId: msg.AllocationId,
	}, nil

}
