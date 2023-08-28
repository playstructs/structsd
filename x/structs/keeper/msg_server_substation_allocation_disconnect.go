package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strconv"
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

	allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId)
	if !allocationFound {
		allocationId := strconv.FormatUint(msg.AllocationId, 10)
		return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationNotFound, "allocation (%s) not found", allocationId)
	}


	// check that the player has reactor permissions
    if (!k.SubstationPermissionHasOneOf(ctx, allocation.DestinationId, player.Id, types.SubstationPermissionDisconnectAllocation)) {
        playerIdString := strconv.FormatUint(player.Id, 10)
        return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationDisconnect, "Calling player (%s) has no Substation Allocation Disconnect permissions ", playerIdString)
    }

    // check that the account has energy management permissions
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if (playerPermissions&types.AddressPermissionManageEnergy == 0) {
        return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }

	_ = k.SubstationDecrementEnergy(ctx, allocation.DestinationId, allocation.Power)
	k.CascadeSubstationAllocationFailure(ctx, allocation.DestinationId)

	allocation.Disconnect()
	k.SetAllocation(ctx, allocation)

	return &types.MsgSubstationAllocationDisconnectResponse{}, nil
}
