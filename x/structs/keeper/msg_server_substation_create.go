package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationCreate(goCtx context.Context, msg *types.MsgSubstationCreate) (*types.MsgSubstationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	connectPlayer := false

    // Make sure the allocation exists
    allocation, allocationFound := cc.GetAllocation(msg.AllocationId)
    if (!allocationFound) {
        return &types.MsgSubstationCreateResponse{}, types.NewObjectNotFoundError("allocation", msg.AllocationId)
    }

	// Check to see if ths calling address is a player and if it relates to the allocation
	//
	// If the allocation doesn't have a player associated with it, then we will use this as
	// a player creation point, initiating their player account and connecting it to this newly
	// formed substation later on in the function call.

	allocationPlayerIndex   := k.GetPlayerIndexFromAddress(ctx, allocation.GetAllocation().Controller)
	callingPlayerIndex      := k.GetPlayerIndexFromAddress(ctx, msg.Creator)

    _, AllocationPlayerFound := k.GetPlayerFromIndex(ctx, allocationPlayerIndex)
    player := cc.UpsertPlayer(msg.Creator)

    if (!AllocationPlayerFound) {
        if (allocation.GetAllocation().Controller == msg.Creator){
            connectPlayer = true
        } else {
            return &types.MsgSubstationCreateResponse{}, types.NewPermissionError("address", msg.Creator, "allocation", allocation.GetAllocationId(), uint64(types.PermissionAssets), "allocation_control")
        }
    } else {
        if (allocationPlayerIndex != callingPlayerIndex) {
            return &types.MsgSubstationCreateResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "allocation", allocation.GetAllocationId(), uint64(types.PermissionAssets), "allocation_control")
        }

        addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
        // check that the account has energy management permissions
        if (!cc.PermissionHasOneOf(addressPermissionId, types.PermissionAssets)) {
            return &types.MsgSubstationCreateResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionAssets), "energy_management")
        }
    }


    substation, err := cc.NewSubstation(msg.Creator, player, allocation)
    if err != nil {
        return &types.MsgSubstationCreateResponse{}, err
    }

    if (connectPlayer) {
        player.MigrateSubstation(substation.GetSubstationId())
    }

	cc.CommitAll()
	return &types.MsgSubstationCreateResponse{SubstationId: substation.GetSubstationId()}, err
}
