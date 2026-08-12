package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlayerUpdatePrimaryAddress(goCtx context.Context, msg *types.MsgPlayerUpdatePrimaryAddress) (*types.MsgPlayerUpdatePrimaryAddressResponse, error) {
    emptyResponse := &types.MsgPlayerUpdatePrimaryAddressResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
        return emptyResponse, err
    }

    player, err := cc.GetPlayerByAddress(msg.PrimaryAddress)
    if err != nil {
       return emptyResponse, err
    }

    err = player.CanUpdatePrimaryAddressBy(callingPlayer)
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

    // Move Reactor Infusions over.
    //
    // Ahead of the coin sweep so that the staking rewards the transfer settles
    // are swept along with everything else. Strict: the old primary stays a
    // registered address of this player, so a refusal costs nothing but a wait.
    err = k.MoveDelegationsToAddress(ctx, cc, oldAcc, msg.PrimaryAddress, DelegationTransferStrict)
    if err != nil {
        return emptyResponse, err
    }

    // Get Balance
    balances := k.bankKeeper.SpendableCoins(ctx, oldAcc)

    // Transfer
    err = k.bankKeeper.SendCoins(ctx, oldAcc, newAcc, balances)
    if err != nil {
        return emptyResponse, err
    }

    // Help the indexer along regarding Ore balances
    _ = ctx.EventManager().EmitTypedEvent(&types.EventOreMigrate{&types.EventOreMigrateDetail{PlayerId: player.GetPlayerId(), PrimaryAddress: msg.PrimaryAddress, OldPrimaryAddress: player.GetPrimaryAddress(), Amount: player.GetStoredOre()}})

    // Finish up
    // This process sets the primary address and upgrades the new address to full rights (careful!)
    player.SetPrimaryAddress(msg.PrimaryAddress)

	cc.CommitAll()
	return &types.MsgPlayerUpdatePrimaryAddressResponse{}, nil
}
