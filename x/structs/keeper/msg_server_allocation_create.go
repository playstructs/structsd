package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationCreate(goCtx context.Context, msg *types.MsgAllocationCreate) (*types.MsgAllocationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // If no controller set, then make it the Creator
    if (msg.Controller == ""){
        msg.Controller = msg.Creator
    }

    player, playerErr := cc.GetPlayerByAddress(msg.Creator)
    if playerErr != nil {
        return &types.MsgAllocationCreateResponse{}, types.NewPlayerRequiredError(msg.Creator, "allocation_create")
    }

    sourceObjectPermissionId := GetObjectPermissionIDBytes(msg.SourceObjectId, player.GetPlayerId())
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

    // Ignore the one case where it's a player creating an allocation on themselves.
    // Surely that doesn't need a lookup.
    if (player.GetPlayerId() != msg.SourceObjectId) {
        // check that the player has permissions
        if (!cc.PermissionHasOneOf(sourceObjectPermissionId, types.PermissionAssets)) {
            return &types.MsgAllocationCreateResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "allocation", msg.SourceObjectId, uint64(types.PermissionAssets), "allocation_create")
        }
    }


    // check that the account has energy management permissions
    if (!cc.PermissionHasOneOf(addressPermissionId, types.Permission(types.PermissionAssets))) {
        return &types.MsgAllocationCreateResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionAssets), "energy_management")
    }

    allocation, err := cc.NewAllocation(
    	msg.AllocationType,
    	msg.SourceObjectId,
    	"",
    	msg.Creator,
    	msg.Controller,
    	msg.Power,
    )

	cc.CommitAll()
	return &types.MsgAllocationCreateResponse{
		AllocationId: allocation.GetAllocationId(),
	}, err

}
