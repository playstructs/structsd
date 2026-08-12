package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) AddressRevoke(goCtx context.Context, msg *types.MsgAddressRevoke) (*types.MsgAddressRevokeResponse, error) {
    emptyResponse := &types.MsgAddressRevokeResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }

    player, err := cc.GetPlayerByAddress(msg.Address)
    if err != nil {
       return emptyResponse, err
    }

    err = player.CanRevokeAddressBy(activePlayer)
    if err != nil {
       return emptyResponse, err
    }

    // Check is msg.Address is the current Primary Address
    if player.GetPrimaryAddress() == msg.Address {
        return emptyResponse, types.NewAddressValidationError(msg.Address, "primary_address")
    }

    /* Got this far, make it so... */
    // Move Funds
    primaryAcc, _   := sdk.AccAddressFromBech32(player.GetPrimaryAddress())
    oldAcc, _       := sdk.AccAddressFromBech32(msg.Address)

    // Move Reactor Infusions over.
    //
    // Ahead of the index revocation below, so the source address still resolves
    // to this player while its infusion is being wound down. Ahead of the coin
    // sweep too: the transfer settles the source's staking rewards, which land
    // in the source's own account, and revoking is the one path where anything
    // left there is gone for good.
    //
    // Disown rather than refuse. Revoking is what a player does about a key
    // they no longer trust, and a refusal is a state whoever holds that key
    // could sustain indefinitely.
    err = k.MoveDelegationsToAddress(ctx, cc, oldAcc, player.GetPrimaryAddress(), DelegationTransferDisown)
    if err != nil {
        return emptyResponse, err
    }

    // Get Balance
    balances := k.bankKeeper.SpendableCoins(ctx, oldAcc)

    // Transfer
    err = k.bankKeeper.SendCoins(ctx, oldAcc, primaryAcc, balances)
    if err != nil {
        return emptyResponse, err
    }

    // Clear Permissions
    addressClearPermissionId := GetAddressPermissionIDBytes(msg.Address)
    k.PermissionClearAll(ctx, addressClearPermissionId)

    // Clear Address Index
    k.RevokePlayerIndexForAddress(ctx, msg.Address, player.GetIndex())

	cc.CommitAll()
	return &types.MsgAddressRevokeResponse{}, nil
}
