package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strconv"
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
		allocationId := strconv.FormatUint(msg.AllocationId, 10)
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationNotFound, "allocation (%s) not found", allocationId)
	}

	substation, substationFound := k.GetSubstation(ctx, msg.DestinationSubstationId, false)
	if !substationFound {
		substationId := strconv.FormatUint(allocation.DestinationId, 10)
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrSubstationNotFound, "destination substation (%s) not found", substationId)
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
        playerIdString := strconv.FormatUint(player.Id, 10)
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationConnect, "Calling player (%s) has no Substation Connect Allocation permissions ", playerIdString)
    }

    // check that the account has energy management permissions
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if (playerPermissions&types.AddressPermissionManageEnergy == 0) {
        return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }


	if substation.Id == allocation.DestinationId {
		substationId := strconv.FormatUint(allocation.DestinationId, 10)
		return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationConnectionChangeImpossible, "destination substation (%s) cannot change to same destination", substationId)
	}

    k.SubstationConnectAllocation(ctx, substation, allocation)

	return &types.MsgSubstationAllocationConnectResponse{}, nil
}
