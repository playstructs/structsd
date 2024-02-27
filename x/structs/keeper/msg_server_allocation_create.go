package keeper

import (
	"context"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) AllocationCreate(goCtx context.Context, msg *types.MsgAllocationCreate) (*types.MsgAllocationCreateResponse, error) {

	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	allocation := types.CreateAllocationStub(msg.AllocationType, msg.SourceObjectId, msg.Power, msg.Creator, msg.Controller)

    player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator), true)
    if (!playerFound) {
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }

    sourceObjectPermissionId := GetObjectPermissionIDBytes(msg.SourceObjectId, player.Id)
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

    // check that the player has permissions
    if (!k.PermissionHasOneOf(ctx, sourceObjectPermissionId, types.Permission(types.SubstationPermissionAllocate))) {
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationCreate, "Calling player (%d) has no Substation Allocation permissions ", player.Id)
    }

    // check that the account has energy management permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionManageEnergy)) {
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }

	allocationId, err := k.AppendAllocation(ctx, allocation)

	return &types.MsgAllocationCreateResponse{
		AllocationId: allocationId,
	}, err

}
