package keeper

import (
	"context"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationUpdate(goCtx context.Context, msg *types.MsgAllocationUpdate) (*types.MsgAllocationUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // Check permissions on the substation

	allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId, true)
	if (!allocationFound) {
		return &types.MsgAllocationUpdateResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "allocation (%s) not found", msg.AllocationId)
	}

    player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator), true)
    if (!playerFound) {
        return &types.MsgAllocationUpdateResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }

    sourceObjectPermissionId := GetObjectPermissionIDBytes(allocation.SourceObjectId, player.Id)
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

    // Ignore the one case where it's a player creating an allocation on themselves.
    // Surely that doesn't need a lookup.
    if (player.Id != allocation.SourceObjectId) {
        // check that the player has permissions
        if (!k.PermissionHasOneOf(ctx, sourceObjectPermissionId, types.PermissionAssets)) {
            return &types.MsgAllocationUpdateResponse{}, sdkerrors.Wrapf(types.ErrPermissionAllocation, "Calling player (%s) has no Allocation permissions on source (%s) ", player.Id, allocation.SourceObjectId)
        }
    }

    // check that the account has energy management permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.PermissionAssets))) {
        return &types.MsgAllocationUpdateResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }


    if (allocation.Type != types.AllocationType_dynamic) {
        return &types.MsgAllocationUpdateResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Allocation Type must be Dynamic for updates ")
    }

    allocation.Power = msg.Power
	allocation, _ = k.SetAllocation(ctx, allocation)

	return &types.MsgAllocationUpdateResponse{
		AllocationId: msg.AllocationId,
	}, nil

}
