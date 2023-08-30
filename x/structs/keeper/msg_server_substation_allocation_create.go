package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SubstationAllocationCreate(goCtx context.Context, msg *types.MsgSubstationAllocationCreate) (*types.MsgAllocationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	allocation := types.Allocation{
		SourceType: types.ObjectType_substation,
		SourceId:   msg.SourceId,
		//DestinationId: 0,
		Power:      0,
		Locked:     false,
		Creator:    msg.Creator,
		Controller: msg.Creator,
	}

	substation, sourceSubstationFound := k.GetSubstation(ctx, msg.SourceId, false)
	if (!sourceSubstationFound) {
		return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceNotFound, "source (%s - %d) used for allocation not found", allocation.SourceType.String(), msg.SourceId)
	}

	player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }

	// check that the player has reactor permissions
    if (!k.SubstationPermissionHasOneOf(ctx, substation.Id, player.Id, types.SubstationPermissionAllocate)) {
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationAllocationCreate, "Calling player (%d) has no Substation Allocation permissions ", player.Id)
    }

    // check that the account has energy management permissions
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if (playerPermissions&types.AddressPermissionManageEnergy == 0) {
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }


	// Check to see if the Substation has the Power available
	// Calling SubstationIncrementAllocationLoad will update the Memory store so the change has already been applied if successful
	//
	// Maybe this will change but currently a new allocation can't be created without the
	// available capacity to bring it online. In the future, we could allow for this and it would
	// blow up older allocations until it hits the threshold, but that feels overly destructive.
	_, incrementLoadError := k.SubstationIncrementAllocationLoad(ctx, msg.SourceId, msg.Power)
	if incrementLoadError != nil {
		return nil, incrementLoadError
	}

	allocation.SetPower(msg.Power)
	allocation.SetController(msg.Controller)

	allocationId := k.AppendAllocation(ctx, allocation)


	return &types.MsgAllocationCreateResponse{
		AllocationId: allocationId,
	}, nil
}
