package keeper

import (
	"context"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) AllocationCreate(goCtx context.Context, msg *types.MsgAllocationCreate) (*types.MsgAllocationCreateResponse, error) {


// TODO turn this into a single allocation system now that source IDs are generalized

	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    // TODO This sucks
    // should be using CreateAllocation
	allocation := types.Allocation{
		SourceType: types.ObjectType_substation,
		SourceId:   msg.SourceId,
		//DestinationId: 0,
		Power:      0,
		Locked:     false,
		Creator:    msg.Creator,
		Controller: msg.Creator,
	}


    player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }

    // check that the player has permissions
    // TODO: Generalize permissions now too
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
