package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) PlayerUpdatePrimaryAddress(goCtx context.Context, msg *types.MsgPlayerUpdatePrimaryAddress) (*types.MsgPlayerUpdatePrimaryAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    callingPlayerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (callingPlayerId == 0) {
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Player Management requires Player account but none associated with %s", msg.Creator)
    }
    callingPlayer, _ := k.GetPlayer(ctx, callingPlayerId)


    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if ((playerPermissions&types.AddressPermissionManagePlayer) == 0) {
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, sdkerrors.Wrapf(types.ErrPermissionManagePlayer, "Calling address (%s) has no Manage Player permissions ", msg.Creator)
    }

    _ , addressValidationError := sdk.AccAddressFromBech32(msg.PrimaryAddress)
    if (addressValidationError != nil){
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "New Primary Address provided (%s) couldn't be validated as a real address. Update aborted. ", msg.PrimaryAddress)
    }

    relatedPlayerId := k.GetPlayerIdFromAddress(ctx, msg.PrimaryAddress)
    if (relatedPlayerId == 0) {
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "New Primary Address provided (%s) is not associated with a player, register it with the player before setting it as Primary. Update aborted.", msg.PrimaryAddress, relatedPlayerId, callingPlayerId)
    }

    if (relatedPlayerId != callingPlayerId) {
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "New Primary Address provided (%s) is associated with Player %d instead of Player %d. Update aborted.", msg.PrimaryAddress, relatedPlayerId, callingPlayerId)
    }

    callingPlayer.SetPrimaryAddress(msg.PrimaryAddress)
    k.SetPlayer(ctx, callingPlayer)

	return &types.MsgPlayerUpdatePrimaryAddressResponse{}, nil
}
