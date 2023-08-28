package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strconv"
	"structs/x/structs/types"
)

func (k msgServer) ReactorAllocationCreate(goCtx context.Context, msg *types.MsgReactorAllocationCreate) (*types.MsgAllocationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    allocation := types.Allocation{
        SourceType: types.ObjectType_reactor,
        SourceId:   msg.SourceId,
        //DestinationId: 0,
        Power:      0,
        Locked:     false,
        Creator:    msg.Creator,
        Controller: msg.Creator,
    }

	player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform reactor action with non-player address (%s)", msg.Creator)
    }

	reactor, sourceReactorFound := k.GetReactor(ctx, msg.SourceId, false)
	if !sourceReactorFound {
		sourceId := strconv.FormatUint(msg.SourceId, 10)
		return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceNotFound, "source (%s) used for allocation not found", allocation.SourceType.String()+"-"+sourceId)
	}

	// check that the player has reactor permissions
    if (!k.ReactorPermissionHasOneOf(ctx, reactor.Id, player.Id, types.ReactorPermissionAllocate)) {
        playerIdString := strconv.FormatUint(player.Id, 10)
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionReactorAllocationCreate, "Calling player (%s) has no Reactor Allocation permissions ", playerIdString)
    }

    // check that the account has energy management permissions
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if (playerPermissions&types.AddressPermissionManageEnergy == 0) {
        return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }




	// Check to see if the Reactor has the Power available
	// Calling ReactorIncrementLoad will update the Memory store so the change has already been applied if successful
	//
	// Maybe this will change but currently a new allocation can't be created without the
	// available capacity to bring it online. In the future, we could allow for this and it would
	// blow up older allocations until it hits the threshold, but that feels overly destructive.
	_, incrementLoadError := k.ReactorIncrementLoad(ctx, msg.SourceId, msg.Power)
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
