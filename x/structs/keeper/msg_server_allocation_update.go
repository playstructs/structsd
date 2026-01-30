package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationUpdate(goCtx context.Context, msg *types.MsgAllocationUpdate) (*types.MsgAllocationUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgAllocationUpdateResponse{}, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

    player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgAllocationUpdateResponse{}, types.NewPlayerRequiredError(msg.Creator, "allocation_update")
    }

    sourceObjectPermissionId := GetObjectPermissionIDBytes(allocation.SourceObjectId, player.Id)
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

    // Ignore the one case where it's a player creating an allocation on themselves.
    // Surely that doesn't need a lookup.
    if (player.Id != allocation.SourceObjectId) {
        // check that the player has permissions
        if (!k.PermissionHasOneOf(ctx, sourceObjectPermissionId, types.PermissionAssets)) {
            return &types.MsgAllocationUpdateResponse{}, types.NewPermissionError("player", player.Id, "allocation", allocation.SourceObjectId, uint64(types.PermissionAssets), "allocation_update")
        }
    }

    // check that the account has energy management permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.PermissionAssets))) {
        return &types.MsgAllocationUpdateResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionAssets), "energy_management")
    }


    if (allocation.Type != types.AllocationType_dynamic) {
        return &types.MsgAllocationUpdateResponse{}, types.NewAllocationError(allocation.SourceObjectId, "immutable_type").WithFieldChange("type", allocation.Type.String(), "dynamic")
    }

    if (msg.Power == 0) {
        return &types.MsgAllocationUpdateResponse{}, types.NewParameterValidationError("power", 0, "below_minimum").WithRange(1, 0)
    }

	allocation, _, err := k.SetAllocation(ctx, allocation, msg.Power)

	return &types.MsgAllocationUpdateResponse{
		AllocationId: msg.AllocationId,
	}, err

}
