package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationAllocationConnect(goCtx context.Context, msg *types.MsgSubstationAllocationConnect) (*types.MsgSubstationAllocationConnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	allocation, allocationFound := cc.GetAllocation(msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgSubstationAllocationConnectResponse{}, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

	substation := cc.GetSubstation(msg.DestinationId)
	if (!substation.LoadSubstation()) {
		return &types.MsgSubstationAllocationConnectResponse{}, types.NewObjectNotFoundError("substation", msg.DestinationId)
	}


	player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgSubstationAllocationConnectResponse{}, err
    }

    allocationPlayer, err := cc.GetPlayerByAddress(allocation.GetAllocation().Controller)
    if err != nil {
        return &types.MsgSubstationAllocationConnectResponse{}, err
    }
    if (allocationPlayer.GetPlayerId() != player.GetPlayerId()) {
        return &types.MsgSubstationAllocationConnectResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "allocation", msg.AllocationId, uint64(types.PermissionGrid), "allocation_connect")
    }


    substationObjectPermissionId := GetObjectPermissionIDBytes(substation.GetSubstationId(), player.GetPlayerId())
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

	// check that the player has reactor permissions
    if (!cc.PermissionHasOneOf(substationObjectPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationAllocationConnectResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "substation", substation.GetSubstationId(), uint64(types.PermissionGrid), "allocation_connect")
    }


    // check that the account has energy management permissions
    if (!cc.PermissionHasOneOf(addressPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationAllocationConnectResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionGrid), "energy_management")
    }


	if (allocation.GetAllocation().SourceObjectId == substation.GetSubstationId()) {
		return &types.MsgSubstationAllocationConnectResponse{}, types.NewAllocationError(allocation.GetAllocation().SourceObjectId, "source_destination_match").WithDestination(substation.GetSubstationId())
	}

	if substation.GetSubstationId() == allocation.GetAllocation().DestinationId {
		return &types.MsgSubstationAllocationConnectResponse{}, types.NewAllocationError(allocation.GetAllocation().SourceObjectId, "same_destination").WithDestination(allocation.GetAllocation().DestinationId)
	}


    allocation.SetDestination(substation.GetSubstationId())

	return &types.MsgSubstationAllocationConnectResponse{}, err
}
