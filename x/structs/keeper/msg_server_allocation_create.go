package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"strings"
	"strconv"
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

    var sourceObject PermissionedObject

	parts := strings.Split(msg.SourceObjectId, "-")
	if len(parts) < 2 {
	    return emptyResponse, types.NewAllocationError(msg.SourceObjectId, "unacceptable_source")
	}

	typeNum, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
        return emptyResponse, types.NewAllocationError(msg.SourceObjectId, "unacceptable_source")
	}

	switch types.ObjectType(typeNum) {
        case types.ObjectType_player:
            player, err := cc.GetPlayer(msg.SourceObjectId)
            if err != nil {
                return emptyResponse, err
            }
            sourceObject = player
        case types.ObjectType_reactor:
            sourceObject = cc.GetReactor(msg.SourceObjectId)
        case types.ObjectType_substation:
            sourceObject = cc.GetSubstation(msg.SourceObjectId)
        default:
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
        creatorPermissions = creatorPermissions | types.PermAllocationConnection
    } else {
        allocationPermissionId := GetObjectPermissionIDBytes(allocation.ID(), msg.Controller)
        cc.k.SetPermissions(allocationPermissionId, types.PermAllocationConnection)
    }
    
    if creatorPermissions != types.Permissionless {
        allocationPermissionId := GetObjectPermissionIDBytes(allocation.ID(), callingPlayer.ID())
        cc.k.SetPermissions(allocationPermissionId, creatorPermissions)
    }

	cc.CommitAll()
	return &types.MsgAllocationCreateResponse{
		AllocationId: allocation.GetAllocationId(),
	}, err

}
