package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlayerSend(goCtx context.Context, msg *types.MsgPlayerSend) (*types.MsgPlayerSendResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    player, err := cc.GetPlayer(msg.PlayerId)
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
        return &types.MsgPlayerSendResponse{}, types.NewAddressValidationError(msg.FromAddress, "invalid_format")
    }

    relatedPlayerIndex := k.GetPlayerIndexFromAddress(ctx, msg.FromAddress)
    if (relatedPlayerIndex == 0) {
        return &types.MsgPlayerSendResponse{}, types.NewAddressValidationError(msg.FromAddress, "not_registered")
    }

    if relatedPlayerIndex != player.GetIndex() {
        return &types.MsgPlayerSendResponse{}, types.NewAddressValidationError(msg.FromAddress, "wrong_player").WithPlayers(player.GetPlayerId(), "")
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
