package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlayerSend(goCtx context.Context, msg *types.MsgPlayerSend) (*types.MsgPlayerSendResponse, error) {
    emptyResponse := &types.MsgPlayerSendResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return emptyResponse, err
    }

    player, err := cc.GetPlayerByAddress(msg.FromAddress)
    if err != nil {
       return emptyResponse, err
    }

    err = player.CanTransferTokensBy(callingPlayer)
    if err != nil {
       return emptyResponse, err
    }

    _ , addressValidationError := sdk.AccAddressFromBech32(msg.FromAddress)
    if (addressValidationError != nil){
        return emptyResponse, types.NewAddressValidationError(msg.FromAddress, "invalid_format")
    }

    // Accounts involved
    fromAcc, _   := sdk.AccAddressFromBech32(msg.FromAddress)
    toAcc, _   := sdk.AccAddressFromBech32(msg.ToAddress)

    // Transfer
    err = k.bankKeeper.SendCoins(ctx, fromAcc, toAcc, msg.Amount)
    if err != nil {
        return emptyResponse, err
    }

	cc.CommitAll()
	return &types.MsgPlayerSendResponse{}, nil
}
