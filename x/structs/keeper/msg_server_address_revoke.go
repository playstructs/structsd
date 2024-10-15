package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	"math"
)

func (k msgServer) AddressRevoke(goCtx context.Context, msg *types.MsgAddressRevoke) (*types.MsgAddressRevokeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    player, err := k.GetPlayerCacheFromAddress(ctx, msg.Address)
    if err != nil {
       return &types.MsgAddressRevokeResponse{}, err
    }

    // Check if msg.Creator has PermissionDelete on the Address and Account
    err = player.CanBeAdministratedBy(msg.Creator, types.PermissionDelete)
    if err != nil {
       return &types.MsgAddressRevokeResponse{}, err
    }

    // Check is msg.Address is the current Primary Address
    if player.GetPrimaryAddress() == msg.Address {
        return &types.MsgAddressRevokeResponse{}, sdkerrors.Wrapf(types.ErrPermissionRevoke, "Cannot Revoke Primary Address. Update Primary Address First")
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

	return &types.MsgAddressRevokeResponse{}, nil
}
