package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	"math"
)

func (k msgServer) AddressRevoke(goCtx context.Context, msg *types.MsgAddressRevoke) (*types.MsgAddressRevokeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
       return &types.MsgAddressRegisterResponse{}, err
    }

    player, err := cc.GetPlayerByAddress(msg.Address)
    if err != nil {
       return &types.MsgAddressRevokeResponse{}, err
    }

    err = player.CanRevokeAddressBy(activePlayer)
    if err != nil {
       return &types.MsgAddressRevokeResponse{}, err
    }

    // Check is msg.Address is the current Primary Address
    if player.GetPrimaryAddress() == msg.Address {
        return &types.MsgAddressRevokeResponse{}, types.NewAddressValidationError(msg.Address, "primary_address")
    }

    /* Got this far, make it so... */
    // Move Funds
    primaryAcc, _   := sdk.AccAddressFromBech32(player.GetPrimaryAddress())
    oldAcc, _       := sdk.AccAddressFromBech32(msg.Address)

    // Get Balance
    balances := k.bankKeeper.SpendableCoins(ctx, oldAcc)

    // Transfer
    err = k.bankKeeper.SendCoins(ctx, oldAcc, primaryAcc, balances)
    if err != nil {
        return &types.MsgAddressRevokeResponse{}, err
    }

    // Move Reactor Infusions over
    primaryDelegations, _ := k.stakingKeeper.GetDelegatorDelegations(ctx, oldAcc, math.MaxUint16)
    for _, delegation := range primaryDelegations {
        k.stakingKeeper.RemoveDelegation(ctx, delegation)

        delegation.DelegatorAddress = player.GetPrimaryAddress()
        k.stakingKeeper.SetDelegation(ctx, delegation)
    }


    // Clear Permissions
    addressClearPermissionId := GetAddressPermissionIDBytes(msg.Address)
    k.PermissionClearAll(ctx, addressClearPermissionId)

    // Clear Address Index
    k.RevokePlayerIndexForAddress(ctx, msg.Address, player.GetIndex())

	cc.CommitAll()
	return &types.MsgAddressRevokeResponse{}, nil
}
