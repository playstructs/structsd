package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// Can't decide if this should be SubstationAllocationDisconnect, or AllocationDisconnect - since there are no other types of disconnections
func (k msgServer) SubstationAllocationDisconnect(goCtx context.Context, msg *types.MsgSubstationAllocationDisconnect) (*types.MsgSubstationAllocationDisconnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    var err error

	player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgSubstationAllocationDisconnectResponse{}, types.NewPlayerRequiredError(msg.Creator, "substation_allocation_disconnect")
    }

	allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgSubstationAllocationDisconnectResponse{}, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

    allocationPlayer, AllocationPlayerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, allocation.Controller))
    if (!AllocationPlayerFound) {
        return &types.MsgSubstationAllocationDisconnectResponse{}, types.NewPlayerRequiredError(allocation.Controller, "substation_allocation_disconnect")
    }
    if (allocationPlayer.Id != player.Id) {
        sourceObjectPermissionId := GetObjectPermissionIDBytes(allocation.DestinationId, player.Id)
        // check that the player has reactor permissions
        if (!k.PermissionHasOneOf(ctx, sourceObjectPermissionId, types.PermissionGrid)) {
            // technically both correct. Refactor this to be clearer
            return &types.MsgSubstationAllocationDisconnectResponse{}, types.NewPermissionError("player", player.Id, "substation", allocation.DestinationId, uint64(types.PermissionGrid), "allocation_disconnect")
        }

    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

    // check that the account has energy management permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationAllocationDisconnectResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionGrid), "energy_management")
    }


    power := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id))
    allocation.DestinationId = ""
    allocation, _, err = k.SetAllocation(ctx, allocation, power)

	return &types.MsgSubstationAllocationDisconnectResponse{}, err
}
