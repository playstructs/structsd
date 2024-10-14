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

    player, err := k.GetPlayerCacheFromId(ctx, msg.PlayerId)
    if err != nil {
       return &types.MsgPlayerUpdatePrimaryAddressResponse{}, err
    }

    // Check if msg.Creator has PermissionDelete on the Address and Account
    err = player.CanBeAdministratedBy(msg.Creator, types.PermissionAssets)
    if err != nil {
       return &types.MsgPlayerUpdatePrimaryAddressResponse{}, err
    }

    _ , addressValidationError := sdk.AccAddressFromBech32(msg.PrimaryAddress)
    if (addressValidationError != nil){
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "New Primary Address provided (%s) couldn't be validated as a real address. Update aborted. ", msg.PrimaryAddress)
    }

    relatedPlayerIndex := k.GetPlayerIndexFromAddress(ctx, msg.PrimaryAddress)
    if (relatedPlayerIndex == 0) {
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "New Primary Address provided (%s) is not associated with a player, register it with the player before setting it as Primary. Update aborted.", msg.PrimaryAddress, relatedPlayerIndex, player.GetIndex())
    }

    if relatedPlayerIndex != player.GetIndex() {
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "New Primary Address provided (%s) is associated with Player %d instead of Player %d. Update aborted.", msg.PrimaryAddress, relatedPlayerIndex, player.GetIndex())
    }


    // Move Funds
    oldAcc, _   := sdk.AccAddressFromBech32(player.GetPrimaryAddress())
    newAcc, _   := sdk.AccAddressFromBech32(msg.PrimaryAddress)

    // Get Balance
    balances := k.bankKeeper.SpendableCoins(ctx, oldAcc)

    // Transfer
    err = k.bankKeeper.SendCoins(ctx, oldAcc, newAcc, balances)
    if err != nil {
        return &types.MsgPlayerUpdatePrimaryAddressResponse{}, err
    }

    // TODO Move Reactor Infusions over


    player.SetPrimaryAddress(msg.PrimaryAddress)
    player.Commit()



	return &types.MsgPlayerUpdatePrimaryAddressResponse{}, nil
}
