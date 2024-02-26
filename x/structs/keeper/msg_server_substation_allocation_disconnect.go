package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

// Can't decide if this should be SubstationAllocationDisconnect, or AllocationDisconnect - since there are no other types of disconnections
func (k msgServer) SubstationAllocationDisconnect(goCtx context.Context, msg *types.MsgSubstationAllocationDisconnect) (*types.MsgSubstationAllocationDisconnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }

	allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId, true)
	if (!allocationFound) {
		return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationNotFound, "allocation (%s) not found", msg.AllocationId)
	}

    allocationPlayer, AllocationPlayerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, allocation.Controller))
    if (!AllocationPlayerFound) {
        return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform substation action with non-player address (%s)", allocation.Controller)
    }
    if (allocationPlayer.Id != player.Id) {
        return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationConnect, "Trying to manage an Allocation not controlled by player ", player.Id)
    }


    sourceObjectPermissionId := GetObjectPermissionIDBytes(substation.Id, player.Id)
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

	// check that the player has reactor permissions
    if (!k.PermissionHasOneOf(ctx, sourceObjectPermissionId, types.Permission(types.SubstationPermissionDisconnectAllocation))) {
        return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationDisconnect, "Calling player (%d) has no Substation Disconnect Allocation permissions ", player.Id)
    }


    // check that the account has energy management permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionManageEnergy))) {
        return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }


    allocation.DestinationObjectId = ""
    allocation, err = k.SetAllocation(ctx, allocation)

	return &types.MsgSubstationAllocationDisconnectResponse{}, err
}
