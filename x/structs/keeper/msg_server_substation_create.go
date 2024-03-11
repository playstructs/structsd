package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SubstationCreate(goCtx context.Context, msg *types.MsgSubstationCreate) (*types.MsgSubstationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	connectPlayer := false

    // Make sure the allocation exists
    allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId, true)
    if (!allocationFound) {
        return &types.MsgSubstationCreateResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "allocation (%s) not found", msg.AllocationId)
    }

	// Check to see if ths calling address is a player and if it relates to the allocation
	//
	// If the allocation doesn't have a player associated with it, then we will use this as
	// a player creation point, initiating their player account and connecting it to this newly
	// formed substation later on in the function call.

	allocationPlayerIndex   := k.GetPlayerIndexFromAddress(ctx, allocation.Controller)
	callingPlayerIndex      := k.GetPlayerIndexFromAddress(ctx, msg.Creator)

    allocationPlayer, AllocationPlayerFound := k.GetPlayerFromIndex(ctx, allocationPlayerIndex, true)
    player := k.UpsertPlayer(ctx, msg.Creator, true)

    if (!AllocationPlayerFound) {
        if (allocation.Controller == msg.Creator){
            connectPlayer = true
        } else {
            return &types.MsgSubstationCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationConnect, "Trying to manage an Allocation (%s) controlled by player (%s), not calling player (%s) ", allocation.Id, allocation.Controller, msg.Creator)
        }
    } else {
        if (allocationPlayerIndex != callingPlayerIndex) {
            return &types.MsgSubstationCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationConnect, "Trying to manage an Allocation (%s) controlled by player (%s), not calling player (%s) ", allocation.Id, allocationPlayer.Id, player.Id)
        }

        addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
        // check that the account has energy management permissions
        if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets)) {
            return &types.MsgSubstationCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
        }
    }

    substation, allocation, err := k.AppendSubstation(ctx, allocation, player)

    if (connectPlayer) {
        k.SubstationConnectPlayer(ctx, substation, player)
    }

	return &types.MsgSubstationCreateResponse{SubstationId: substation.Id}, err
}
