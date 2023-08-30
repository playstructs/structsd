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

	allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId)
	if !allocationFound {
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationNotFound, "allocation (%d) not found", msg.AllocationId)
	}

	substation, substationFound := k.GetSubstation(ctx, msg.DestinationSubstationId, false)
	if !substationFound {
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
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionAllocation, "Could not perform connect allocation with allocation that does not belong to calling player] (%s)", msg.Creator)
    }


	// check that the player has reactor permissions
    if (!k.SubstationPermissionHasOneOf(ctx, substation.Id, player.Id, types.SubstationPermissionConnectAllocation)) {
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationConnect, "Calling player (%d) has no Substation Connect Allocation permissions ", player.Id)
    }

    // check that the account has energy management permissions
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if (playerPermissions&types.AddressPermissionManageEnergy == 0) {
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }


	if substation.Id == allocation.DestinationId {
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationConnectionChangeImpossible, "destination substation (%d) cannot change to same destination", allocation.DestinationId)
	}

    k.SubstationConnectAllocation(ctx, substation, allocation)

	return &types.MsgSubstationAllocationConnectResponse{}, nil
}
