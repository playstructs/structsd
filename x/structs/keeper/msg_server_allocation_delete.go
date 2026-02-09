package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationDelete(goCtx context.Context, msg *types.MsgAllocationDelete) (*types.MsgAllocationDeleteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgAllocationDeleteResponse{}, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

    player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgAllocationDeleteResponse{}, types.NewPlayerRequiredError(msg.Creator, "allocation_delete")
    }

    sourceObjectPermissionId := GetObjectPermissionIDBytes(allocation.SourceObjectId, player.PlayerId)
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

    // Ignore the one case where it's a player creating an allocation on themselves.
    // Surely that doesn't need a lookup.
    if (player.PlayerId != allocation.SourceObjectId) {
        // check that the player has permissions
        if (!k.PermissionHasOneOf(ctx, sourceObjectPermissionId, types.PermissionAssets)) {
            return &types.MsgAllocationDeleteResponse{}, types.NewPermissionError("player", player.PlayerId, "allocation", allocation.SourceObjectId, uint64(types.PermissionAssets), "allocation_delete")
        }
    }

    // check that the account has energy management permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.PermissionAssets))) {
        return &types.MsgAllocationDeleteResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionAssets), "energy_management")
    }


    if (allocation.Type != types.AllocationType_dynamic) {
        return &types.MsgAllocationDeleteResponse{}, types.NewAllocationError(allocation.SourceObjectId, "immutable_type").WithFieldChange("type", allocation.Type.String(), "dynamic")
    }

    k.DestroyAllocation(ctx, msg.AllocationId)

	return &types.MsgAllocationDeleteResponse{
		AllocationId: msg.AllocationId,
	}, nil

}
