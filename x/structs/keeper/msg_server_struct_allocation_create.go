package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructAllocationCreate(goCtx context.Context, msg *types.MsgStructAllocationCreate) (*types.MsgAllocationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	allocation := types.Allocation{
		SourceType: types.ObjectType_struct,
		SourceId:   msg.SourceId,
		//DestinationId: 0,
		Power:      0,
		Locked:     false,
		Creator:    msg.Creator,
		Controller: msg.Creator,
	}

    structure, structureFound := k.GetStruct(ctx, msg.SourceId)
    if (!structureFound) {
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%d) not found", msg.SourceId)
    }

    if (structure.PowerSystem != 1) {
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrStructAllocationCreate, "Struct (%d) has no power systems to allocate from", msg.SourceId)
    }

	player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }

    // check that the account has energy management permissions
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if (playerPermissions&types.AddressPermissionManageEnergy == 0) {
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }

    /*
     * Until we let players give out player permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo build structs for others, no permission to allocate (%s)", structure.Id)
    }


	// Check to see if the Struct has the Power available
	//
	// Maybe this will change but currently a new allocation can't be created without the
	// available capacity to bring it online. In the future, we could allow for this and it would
	// blow up older allocations until it hits the threshold, but that feels overly destructive.
	_, incrementLoadError := k.StructIncrementLoad(ctx, msg.SourceId, msg.Power)
	if incrementLoadError != nil {
       return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrStructAllocationCreate, "Failed to allocate that amount from %d", structure.Id)
	}

	allocation.SetPower(msg.Power)
	allocation.SetController(msg.Controller)

	allocationId := k.AppendAllocation(ctx, allocation)


	return &types.MsgAllocationCreateResponse{
		AllocationId: allocationId,
	}, nil
}
