package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) PlayerUpdatePrimaryAddress(goCtx context.Context, msg *types.MsgPlayerUpdatePrimaryAddress) (*types.MsgPlayerUpdatePrimaryAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (callingPlayerIndex == 0) {
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Player Management requires Player account but none associated with %s", msg.Creator)
    }
    callingPlayer, _ := k.GetPlayerFromIndex(ctx, callingPlayerIndex, true)


    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Manage Player permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets)) {
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, sdkerrors.Wrapf(types.ErrPermissionManagePlayer, "Calling address (%s) has no Manage Player permissions ", msg.Creator)
    }


    _ , addressValidationError := sdk.AccAddressFromBech32(msg.PrimaryAddress)
    if (addressValidationError != nil){
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "New Primary Address provided (%s) couldn't be validated as a real address. Update aborted. ", msg.PrimaryAddress)
    }

    relatedPlayerIndex := k.GetPlayerIndexFromAddress(ctx, msg.PrimaryAddress)
    if (relatedPlayerIndex == 0) {
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "New Primary Address provided (%s) is not associated with a player, register it with the player before setting it as Primary. Update aborted.", msg.PrimaryAddress, relatedPlayerIndex, callingPlayerIndex)
    }

    if (relatedPlayerIndex != callingPlayerIndex) {
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "New Primary Address provided (%s) is associated with Player %d instead of Player %d. Update aborted.", msg.PrimaryAddress, relatedPlayerIndex, callingPlayerIndex)
    }

    callingPlayer.PrimaryAddress = msg.PrimaryAddress
    k.SetPlayer(ctx, callingPlayer)

	return &types.MsgPlayerUpdatePrimaryAddressResponse{}, nil
}
