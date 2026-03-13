package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AllocationCreate(goCtx context.Context, msg *types.MsgAllocationCreate) (*types.MsgAllocationCreateResponse, error) {
    emptyResponse := &types.MsgAllocationCreateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }

    // If no controller set, then make it the Creator
    if (msg.Controller == ""){
        msg.Controller = callingPlayer.GetPlayerId()
    }

    sourceObject := cc.GetPermissionedObject(msg.SourceObjectId)
    if sourceObject == nil {
        return emptyResponse, types.NewAllocationError(msg.SourceObjectId, "unacceptable_source")
    }

    permissionErr := sourceObject.CanAllocateAsSourceBy(callingPlayer)
    if permissionErr != nil {
            return emptyResponse, permissionErr
    }

    allocation, err := cc.NewAllocation(
    	msg.AllocationType,
    	msg.SourceObjectId,
    	"",
    	msg.Creator,
    	msg.Controller,
    	msg.Power,
    )


    var creatorPermissions types.Permission
    if allocation.IsAutomated() {
        creatorPermissions = types.PermDelete 
    }

    if allocation.IsDynamic() {
        creatorPermissions = types.PermUpdate | types.PermDelete
    }

    if callingPlayer.ID() == msg.Controller {
        creatorPermissions = creatorPermissions | types.PermAllocationConnection | types.PermAdmin
    } else {
        allocationPermissionId := GetObjectPermissionIDBytes(allocation.ID(), msg.Controller)
        cc.SetPermissions(allocationPermissionId, types.PermAllocationConnection | types.PermAdmin)
    }

    if creatorPermissions != types.Permissionless {
        allocationPermissionId := GetObjectPermissionIDBytes(allocation.ID(), callingPlayer.ID())
        cc.SetPermissions(allocationPermissionId, creatorPermissions)
    }

	cc.CommitAll()
	return &types.MsgAllocationCreateResponse{
		AllocationId: allocation.GetAllocationId(),
	}, err

}
