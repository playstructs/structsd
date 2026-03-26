package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationAllocationConnect(goCtx context.Context, msg *types.MsgSubstationAllocationConnect) (*types.MsgSubstationAllocationConnectResponse, error) {
	emptyResponse := &types.MsgSubstationAllocationConnectResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }

	allocation, allocationFound := cc.GetAllocation(msg.AllocationId)
	if (!allocationFound) {
		return emptyResponse, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

	allocationPermissionErr := allocation.CanBeConnectedBy(callingPlayer)
    if allocationPermissionErr != nil {
        return emptyResponse, allocationPermissionErr
    }

	substation := cc.GetSubstation(msg.DestinationId)
	if (!substation.LoadSubstation()) {
		return emptyResponse, types.NewObjectNotFoundError("substation", msg.DestinationId)
	}

	if (allocation.GetAllocation().SourceObjectId == substation.GetSubstationId()) {
		return emptyResponse, types.NewAllocationError(allocation.GetAllocation().SourceObjectId, "source_destination_match").WithDestination(substation.GetSubstationId())
	}

	if substation.GetSubstationId() == allocation.GetAllocation().DestinationId {
		return emptyResponse, types.NewAllocationError(allocation.GetAllocation().SourceObjectId, "same_destination").WithDestination(allocation.GetAllocation().DestinationId)
	}

    allocation.SetDestination(substation.GetSubstationId())

	cc.CommitAll()
	return &types.MsgSubstationAllocationConnectResponse{}, err
}
