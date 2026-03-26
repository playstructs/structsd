package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// Can't decide if this should be SubstationAllocationDisconnect, or AllocationDisconnect - since there are no other types of disconnections
func (k msgServer) SubstationAllocationDisconnect(goCtx context.Context, msg *types.MsgSubstationAllocationDisconnect) (*types.MsgSubstationAllocationDisconnectResponse, error) {
    emptyResponse := &types.MsgSubstationAllocationDisconnectResponse{}
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

	allocationPermissionErr := allocation.CanBeDisconnectedBy(callingPlayer)
	if allocationPermissionErr != nil {
	    return emptyResponse, allocationPermissionErr
	}

    allocation.SetDestination("")

	cc.CommitAll()
	return emptyResponse, err
}
