package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationAllocationConnect(goCtx context.Context, msg *types.MsgSubstationAllocationConnect) (*types.MsgSubstationAllocationConnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    var err error

	allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgSubstationAllocationConnectResponse{}, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

	substation, substationFound := k.GetSubstation(ctx, msg.DestinationId)
	if (!substationFound) {
		return &types.MsgSubstationAllocationConnectResponse{}, types.NewObjectNotFoundError("substation", msg.DestinationId)
	}


	player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgSubstationAllocationConnectResponse{}, types.NewPlayerRequiredError(msg.Creator, "substation_allocation_connect")
    }

    allocationPlayer, AllocationPlayerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, allocation.Controller))
    if (!AllocationPlayerFound) {
        return &types.MsgSubstationAllocationConnectResponse{}, types.NewPlayerRequiredError(allocation.Controller, "substation_allocation_connect")
    }
    if (allocationPlayer.Id != player.Id) {
        return &types.MsgSubstationAllocationConnectResponse{}, types.NewPermissionError("player", player.Id, "allocation", msg.AllocationId, uint64(types.PermissionGrid), "allocation_connect")
    }


    substationObjectPermissionId := GetObjectPermissionIDBytes(substation.Id, player.Id)
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

	// check that the player has reactor permissions
    if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationAllocationConnectResponse{}, types.NewPermissionError("player", player.Id, "substation", substation.Id, uint64(types.PermissionGrid), "allocation_connect")
    }


    // check that the account has energy management permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationAllocationConnectResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionGrid), "energy_management")
    }


	if (allocation.SourceObjectId == substation.Id) {
		return &types.MsgSubstationAllocationConnectResponse{}, types.NewAllocationError(allocation.SourceObjectId, "source_destination_match").WithDestination(substation.Id)
	}

	if substation.Id == allocation.DestinationId {
		return &types.MsgSubstationAllocationConnectResponse{}, types.NewAllocationError(allocation.SourceObjectId, "same_destination").WithDestination(allocation.DestinationId)
	}

    power := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id))
    allocation.DestinationId = substation.Id
    allocation, _, err = k.SetAllocation(ctx, allocation, power)

	return &types.MsgSubstationAllocationConnectResponse{}, err
}
