package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SubstationAllocationConnect(goCtx context.Context, msg *types.MsgSubstationAllocationConnect) (*types.MsgSubstationAllocationConnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId, true)
	if (!allocationFound) {
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationNotFound, "allocation (%d) not found", msg.AllocationId)
	}

	substation, substationFound := k.GetSubstation(ctx, msg.DestinationObjectId, false)
	if (!substationFound) {
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrSubstationNotFound, "destination substation (%d) not found", allocation.DestinationId)
	}


	player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }

    allocationPlayer, AllocationPlayerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, allocation.Controller))
    if (!AllocationPlayerFound) {
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform substation action with non-player address (%s)", allocation.Controller)
    }
    if (allocationPlayer.Id != player.Id) {
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationConnect, "Trying to manage an Allocation not controlled by player ", player.Id)
    }


    substationObjectPermissionId := GetObjectPermissionIDBytes(substation.Id, player.Id)
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

	// check that the player has reactor permissions
    if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.Permission(types.SubstationPermissionConnectAllocation))) {
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationConnect, "Calling player (%d) has no Substation Connect Allocation permissions ", player.Id)
    }


    // check that the account has energy management permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionManageEnergy))) {
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }


	if (allocation.SourceObjectId == substation.Id) {
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationConnectionChangeImpossible, "destination substation (%s) cannot match allocation source (%s)", allocation.DestinationObjectId, allocation.SourceObjectId)
	}

	if substation.Id == allocation.DestinationObjectId {
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationConnectionChangeImpossible, "destination substation (%s) cannot change to same destination", allocation.DestinationObjectId)
	}


    allocation.DestinationObjectId = substation.Id
    allocation, err = k.SetAllocation(ctx, allocation)

	return &types.MsgSubstationAllocationConnectResponse{}, err
}
