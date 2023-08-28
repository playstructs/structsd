package keeper

import (
	"context"
    "strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SubstationDelete(goCtx context.Context, msg *types.MsgSubstationDelete) (*types.MsgSubstationDeleteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}


	player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgSubstationDeleteResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }


	// check that the player has reactor permissions
    if (!k.SubstationPermissionHasOneOf(ctx, msg.SubstationId, player.Id, types.SubstationPermissionDelete)) {
        playerIdString := strconv.FormatUint(player.Id, 10)
        return &types.MsgSubstationDeleteResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationDelete, "Calling player (%s) has no Substation Delete permissions ", playerIdString)
    }


    // check that the account has energy management permissions
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if (playerPermissions&types.AddressPermissionManageEnergy == 0) {
        return &types.MsgSubstationDeleteResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }


	/*
	 * This is going to start out very inefficient. We'll need to tackle
	 * ways to improve these types of graph traversal
	 */

	// Need all allocations in

	// Need all allocations out

	// Need all players connected

	k.RemoveSubstation(ctx, msg.SubstationId)

	return &types.MsgSubstationDeleteResponse{}, nil
}
