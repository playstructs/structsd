package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// Can't decide if this should be SubstationAllocationDisconnect, or AllocationDisconnect - since there are no other types of disconnections
func (k msgServer) SubstationAllocationDisconnect(goCtx context.Context, msg *types.MsgSubstationAllocationDisconnect) (*types.MsgSubstationAllocationDisconnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgSubstationAllocationDisconnectResponse{}, err
    }

	allocation, allocationFound := cc.GetAllocation(msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgSubstationAllocationDisconnectResponse{}, types.NewObjectNotFoundError("allocation", msg.AllocationId)
	}

    allocationPlayer, err := cc.GetPlayerByAddress(allocation.GetAllocation().Controller)
    if err != nil {
        return &types.MsgSubstationAllocationDisconnectResponse{}, err
    }
    if (allocationPlayer.GetPlayerId() != player.GetPlayerId()) {
        sourceObjectPermissionId := GetObjectPermissionIDBytes(allocation.GetAllocation().DestinationId, player.GetPlayerId())
        // check that the player has reactor permissions
        if (!cc.PermissionHasOneOf(sourceObjectPermissionId, types.PermissionGrid)) {
            // technically both correct. Refactor this to be clearer
            return &types.MsgSubstationAllocationDisconnectResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "substation", allocation.GetAllocation().DestinationId, uint64(types.PermissionGrid), "allocation_disconnect")
        }
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

    // check that the account has energy management permissions
    if (!cc.PermissionHasOneOf(addressPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationAllocationDisconnectResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionGrid), "energy_management")
    }

    allocation.SetDestination("")

	cc.CommitAll()
	return &types.MsgSubstationAllocationDisconnectResponse{}, err
}
