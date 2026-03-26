package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	"math"
)

func (k msgServer) PlayerUpdatePrimaryAddress(goCtx context.Context, msg *types.MsgPlayerUpdatePrimaryAddress) (*types.MsgPlayerUpdatePrimaryAddressResponse, error) {
    emptyResponse := &types.MsgPlayerUpdatePrimaryAddressResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return emptyResponse, err
    }

    player, err := cc.GetPlayerByAddress(msg.PrimaryAddress)
    if err != nil {
       return emptyResponse, err
    }

    err = player.CanBeAdministeredBy(callingPlayer)
    if err != nil {
       return emptyResponse, err
    }

    _ , addressValidationError := sdk.AccAddressFromBech32(msg.PrimaryAddress)
    if (addressValidationError != nil){
        return emptyResponse, types.NewAddressValidationError(msg.PrimaryAddress, "invalid_format")
    }

    // Move Funds
    oldAcc, _   := sdk.AccAddressFromBech32(player.GetPrimaryAddress())
    newAcc, _   := sdk.AccAddressFromBech32(msg.PrimaryAddress)

    // Get Balance
    balances := k.bankKeeper.SpendableCoins(ctx, oldAcc)

    // Transfer
    err = k.bankKeeper.SendCoins(ctx, oldAcc, newAcc, balances)
    if err != nil {
        return emptyResponse, err
    }

    // Move Reactor Infusions over
    primaryDelegations, _ := k.stakingKeeper.GetDelegatorDelegations(ctx, oldAcc, math.MaxUint16)
    for _, delegation := range primaryDelegations {
        k.stakingKeeper.RemoveDelegation(ctx, delegation)

        delegation.DelegatorAddress = msg.PrimaryAddress
        k.stakingKeeper.SetDelegation(ctx, delegation)
    }

    // Help the indexer along regarding Ore balances
    _ = ctx.EventManager().EmitTypedEvent(&types.EventOreMigrate{&types.EventOreMigrateDetail{PlayerId: player.GetPlayerId(), PrimaryAddress: msg.PrimaryAddress, OldPrimaryAddress: player.GetPrimaryAddress(), Amount: player.GetStoredOre()}})

    // Finish up
    // This process sets the primary address and upgrades the new address to full rights (careful!)
    player.SetPrimaryAddress(msg.PrimaryAddress)

	cc.CommitAll()
	return &types.MsgPlayerUpdatePrimaryAddressResponse{}, nil
}
