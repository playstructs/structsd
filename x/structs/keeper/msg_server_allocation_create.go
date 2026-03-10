package keeper

import (
	"context"
	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"strings"
	"strconv"
)

func (k msgServer) AllocationCreate(goCtx context.Context, msg *types.MsgAllocationCreate) (*types.MsgAllocationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
       return &types.MsgAllocationCreateResponse{}, err
    }

    // If no controller set, then make it the Creator
    if (msg.Controller == ""){
        msg.Controller = activePlayer.GetPlayerId()
    }

    var sourceObject PermissionedObject

	parts := strings.Split(msg.SourceObjectId, "-")
	if len(parts) < 2 {
	    return &types.MsgAllocationCreateResponse{}, types.NewAllocationError(msg.SourceObjectId, "unacceptable_source")
	}

	typeNum, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
        return &types.MsgAllocationCreateResponse{}, types.NewAllocationError(msg.SourceObjectId, "unacceptable_source")
	}

	switch types.ObjectType(typeNum) {
        case types.ObjectType_player:
            player, err := cc.GetPlayer(msg.SourceObjectId)
            if err != nil {
                return &types.MsgAllocationCreateResponse{}, err
            }
            sourceObject = player
        case types.ObjectType_reactor:
            sourceObject = cc.GetReactor(msg.SourceObjectId)
        case types.ObjectType_substation:
            sourceObject = cc.GetSubstation(msg.SourceObjectId)
        default:
            return &types.MsgAllocationCreateResponse{}, types.NewAllocationError(msg.SourceObjectId, "unacceptable_source")
	}

    permissionErr := sourceObject.CanAllocateAsSourceBy(activePlayer)
    if permissionErr != nil {
            return &types.MsgAllocationCreateResponse{}, permissionErr
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
