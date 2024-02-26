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

	var player types.Player
	connectPlayer := false

    // Make sure the allocation exists
    allocation, allocationFound := k.GetAllocation(ctx, msg.AllocationId, true)
    if (!allocationFound) {
        return &types.MsgSubstationCreateResponse{}, sdkerrors.Wrapf(types.ErrAllocationNotFound, "allocation (%s) not found", msg.AllocationId)
    }

	// Check to see if ths calling address is a player and if it relates to the allocation
    allocationPlayer, AllocationPlayerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, allocation.Controller))
    if (!AllocationPlayerFound) {
        if (allocation.Controller == msg.Creator){
            player := k.UpsertPlayer(ctx, msg.Creator)
            connectPlayer = true
        } else {
            return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationConnect, "Trying to manage an Allocation not controlled by player ", player.Id)
        }
    } else {
        if (allocationPlayer.Id != player.Id) {
            return &types.MsgSubstationAllocationConnectResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationConnect, "Trying to manage an Allocation not controlled by player ", player.Id)
        }
    }


    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // check that the account has energy management permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionManageEnergy))) {
        return &types.MsgSubstationCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }

    substation, allocation, err := k.AppendSubstation(ctx, allocation, player)

    if (connectPlayer) {
        // TODO - need to likely do some player work next
        k.SubstationConnectPlayer(ctx, substation, player)
    }

	return &types.MsgSubstationCreateResponse{SubstationId: substation.Id}, err
}
