package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) PlayerSend(goCtx context.Context, msg *types.MsgPlayerSend) (*types.MsgPlayerSendResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    player, err := k.GetPlayerCacheFromId(ctx, msg.PlayerId)
    if err != nil {
       return &types.MsgPlayerSendResponse{}, err
    }

    // Check if msg.Creator has PermissionDelete on the Address and Account
    err = player.CanBeAdministratedBy(msg.Creator, types.PermissionAssets)
    if err != nil {
       return &types.MsgPlayerSendResponse{}, err
    }

    _ , addressValidationError := sdk.AccAddressFromBech32(msg.FromAddress)
    if (addressValidationError != nil){
        return &types.MsgPlayerSendResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "From Address provided (%s) couldn't be validated as a real address. Update aborted. ", msg.FromAddress)
    }

    relatedPlayerIndex := k.GetPlayerIndexFromAddress(ctx, msg.FromAddress)
    if (relatedPlayerIndex == 0) {
        return &types.MsgPlayerSendResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "From Address provided (%s) is not associated with a player, register it with the player before setting it as Primary. Update aborted.", msg.FromAddress)
    }

    if relatedPlayerIndex != player.GetIndex() {
        return &types.MsgPlayerSendResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "From Address provided (%s) is associated with Player %d instead of Player %d. Update aborted.", msg.FromAddress, relatedPlayerIndex, player.GetIndex())
    }

    // Accounts involved
    fromAcc, _   := sdk.AccAddressFromBech32(msg.FromAddress)
    toAcc, _   := sdk.AccAddressFromBech32(msg.ToAddress)

    // Transfer
    err = k.bankKeeper.SendCoins(ctx, fromAcc, toAcc, msg.Amount)
    if err != nil {
        return &types.MsgPlayerSendResponse{}, err
    }

	return &types.MsgPlayerSendResponse{}, nil
}
