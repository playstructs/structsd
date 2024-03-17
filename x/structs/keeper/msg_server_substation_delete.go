package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) SubstationDelete(goCtx context.Context, msg *types.MsgSubstationDelete) (*types.MsgSubstationDeleteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator), false)
    if (!playerFound) {
        return &types.MsgSubstationDeleteResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Could not perform substation action with non-player address (%s)", msg.Creator)
    }


    substationObjectPermissionId := GetObjectPermissionIDBytes(msg.SubstationId, player.Id)
	// check that the player has reactor permissions
    if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionDelete)) {
        return &types.MsgSubstationDeleteResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationDelete, "Calling player (%d) has no Substation Delete permissions ", player.Id)
    }


    // check that the account has energy management permissions
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets)) {
        return &types.MsgSubstationDeleteResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }



	k.RemoveSubstation(ctx, msg.SubstationId, msg.MigrationSubstationId)

	return &types.MsgSubstationDeleteResponse{}, nil
}
