package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationCreate(goCtx context.Context, msg *types.MsgAllocationCreate) (*types.MsgAllocationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // If no controller set, then make it the Creator
    if (msg.Controller == ""){
        msg.Controller = msg.Creator
    }

	allocation := types.CreateAllocationStub(msg.AllocationType, msg.SourceObjectId, msg.Creator, msg.Controller)

    player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgAllocationCreateResponse{}, types.NewPlayerRequiredError(msg.Creator, "allocation_create")
    }

    sourceObjectPermissionId := GetObjectPermissionIDBytes(msg.SourceObjectId, player.Id)
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

    // Ignore the one case where it's a player creating an allocation on themselves.
    // Surely that doesn't need a lookup.
    if (player.Id != msg.SourceObjectId) {
        // check that the player has permissions
        if (!k.PermissionHasOneOf(ctx, sourceObjectPermissionId, types.PermissionAssets)) {
            return &types.MsgAllocationCreateResponse{}, types.NewPermissionError("player", player.Id, "allocation", msg.SourceObjectId, uint64(types.PermissionAssets), "allocation_create")
        }
    }


    // check that the account has energy management permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.PermissionAssets))) {
        return &types.MsgAllocationCreateResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionAssets), "energy_management")
    }

	_ = cc

	allocation, _ , err := k.AppendAllocation(ctx, allocation, msg.Power)

	return &types.MsgAllocationCreateResponse{
		AllocationId: allocation.Id,
	}, err

}
