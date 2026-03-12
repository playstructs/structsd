package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationCreate(goCtx context.Context, msg *types.MsgSubstationCreate) (*types.MsgSubstationCreateResponse, error) {
    emptyResponse := &types.MsgSubstationCreateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }

    // Make sure the allocation exists
    allocation, allocationFound := cc.GetAllocation(msg.AllocationId)
    if (!allocationFound) {
        return emptyResponse, types.NewObjectNotFoundError("allocation", msg.AllocationId)
    }

    allocationPermissionErr := allocation.CanBeConnectedBy(callingPlayer)
    if allocationPermissionErr != nil {
        return emptyResponse, allocationPermissionErr
    }

    substation, err := cc.NewSubstation(msg.Creator, callingPlayer, allocation)
    if err != nil {
        return emptyResponse, err
    }

    substationPermissionId := GetObjectPermissionIDBytes(substation.ID(), callingPlayer.ID())
    cc.SetPermissions(substationPermissionId, types.PermSubstationAll)

    /*
        Maybe make this an option...
        player.MigrateSubstation(substation.GetSubstationId())
    */

	cc.CommitAll()
	return &types.MsgSubstationCreateResponse{SubstationId: substation.GetSubstationId()}, err
}
