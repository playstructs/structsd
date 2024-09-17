package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) SubstationAllocationConnect(goCtx context.Context, msg *types.MsgSubstationAllocationConnect) (*types.MsgSubstationAllocationConnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    var err error

	allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId)
	if (!allocationFound) {
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "allocation (%s) not found", msg.AllocationId)
	}

	substation, substationFound := k.GetSubstation(ctx, msg.DestinationId)
	if (!substationFound) {
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "destination substation (%s) not found", allocation.DestinationId)
	}


	player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }

    allocationPlayer, AllocationPlayerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, allocation.Controller))
    if (!AllocationPlayerFound) {
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Could not perform substation action with non-player address (%s)", allocation.Controller)
    }
    if (allocationPlayer.Id != player.Id) {
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationConnect, "Trying to manage an Allocation not controlled by player ", player.Id)
    }


    substationObjectPermissionId := GetObjectPermissionIDBytes(substation.Id, player.Id)
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

	// check that the player has reactor permissions
    if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationConnect, "Calling player (%d) has no Substation Connect Allocation permissions ", player.Id)
    }


    // check that the account has energy management permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionGrid)) {
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }


	if (allocation.SourceObjectId == substation.Id) {
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationConnectionChangeImpossible, "destination substation (%s) cannot match allocation source (%s)", allocation.DestinationId, allocation.SourceObjectId)
	}

	if substation.Id == allocation.DestinationId {
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationConnectionChangeImpossible, "destination substation (%s) cannot change to same destination", allocation.DestinationId)
	}


    allocation.DestinationId = substation.Id
    allocation, err = k.SetAllocation(ctx, allocation)

	return &types.MsgSubstationAllocationConnectResponse{}, err
}
