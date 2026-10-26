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

    callingPlayer, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
        return emptyResponse, err
    }

    fromAcc, addressValidationError := sdk.AccAddressFromBech32(msg.FromAddress)
    if addressValidationError != nil {
        return emptyResponse, types.NewAddressValidationError(msg.FromAddress, "invalid_format")
    }

    // A vault is an asset source, never a signing identity. Resolve its live
    // owner before applying the same token-key ceiling as other vault spends.
    if ownerID, found := k.GetVaultOwner(ctx, fromAcc); found {
        vault, err := cc.GetVault(ownerID)
        if err != nil { return emptyResponse, err }
        if err := vault.CanUseBy(callingPlayer, types.PermTokenTransfer); err != nil {
            return emptyResponse, err
        }
    } else {
        player, err := cc.GetPlayerByAddress(msg.FromAddress)
        if err != nil { return emptyResponse, err }
        if err := player.CanTransferTokensBy(callingPlayer); err != nil {
            return emptyResponse, err
        }
    }

    // The recipient is the one address in this module a transaction gets to
    // choose freely, so it is the one that has to be held to the bank's own
    // policy: SendCoins applies neither the blocked-address set nor the
    // send-enabled flags. See resolveExternalRecipient.
    //
    // The error used to be discarded here while FromAddress was checked two
    // lines above, so a malformed recipient became the empty AccAddress rather
    // than a rejection.
    toAcc, recipientError := k.resolveExternalRecipient(msg.ToAddress)
    if recipientError != nil {
        return emptyResponse, recipientError
    }

    if amountError := k.requireSendableAmount(ctx, msg.Amount); amountError != nil {
        return emptyResponse, amountError
    }

    // Transfer
    err = k.bankKeeper.SendCoins(ctx, fromAcc, toAcc, msg.Amount)
    if err != nil {
        return emptyResponse, err
    }

	cc.CommitAll()
	return &types.MsgPlayerSendResponse{}, nil
}
