package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationUpdate(goCtx context.Context, msg *types.MsgAllocationUpdate) (*types.MsgAllocationUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	allocation, allocationFound := cc.GetAllocation(msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgAllocationUpdateResponse{}, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

    player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgAllocationUpdateResponse{}, types.NewPlayerRequiredError(msg.Creator, "allocation_update")
    }

    sourceObjectPermissionId := GetObjectPermissionIDBytes(allocation.GetAllocation().SourceObjectId, player.GetPlayerId())
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

    // Ignore the one case where it's a player creating an allocation on themselves.
    // Surely that doesn't need a lookup.
    if (player.GetPlayerId() != allocation.GetAllocation().SourceObjectId) {
        // check that the player has permissions
        if (!cc.PermissionHasOneOf(sourceObjectPermissionId, types.PermissionAssets)) {
            return &types.MsgAllocationUpdateResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "allocation", allocation.GetAllocation().SourceObjectId, uint64(types.PermissionAssets), "allocation_update")
        }
    }

    // check that the account has energy management permissions
    if (!cc.PermissionHasOneOf(addressPermissionId, types.Permission(types.PermissionAssets))) {
        return &types.MsgAllocationUpdateResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionAssets), "energy_management")
    }


    if (allocation.GetAllocation().Type != types.AllocationType_dynamic) {
        return &types.MsgAllocationUpdateResponse{}, types.NewAllocationError(allocation.GetAllocation().SourceObjectId, "immutable_type").WithFieldChange("type", allocation.GetAllocation().Type.String(), "dynamic")
    }

    if (msg.Power == 0) {
        return &types.MsgAllocationUpdateResponse{}, types.NewParameterValidationError("power", 0, "below_minimum").WithRange(1, 0)
    }

    allocation.SetDynamicPower(msg.Power)

	return &types.MsgAllocationUpdateResponse{
		AllocationId: msg.AllocationId,
	}, err

}
