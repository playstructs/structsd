package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) VaultSend(goCtx context.Context, msg *types.MsgVaultSend) (*types.MsgVaultSendResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	actor, err := cc.GetSigningPlayer(msg.Creator)
	if err != nil {
		return nil, err
	}
	source, err := cc.GetVault(msg.OwnerId)
	if err != nil {
		return nil, err
	}
	if err := source.CanUseBy(actor, types.PermTokenTransfer); err != nil {
		return nil, err
	}
	destination, err := cc.GetVault(msg.RecipientOwnerId)
	if err != nil {
		return nil, err
	}
	if len(msg.Amount) == 0 {
		return nil, types.NewParameterValidationError("amount", 0, "not_positive")
	}
	if err := k.requireSendableAmount(ctx, msg.Amount); err != nil {
		return nil, err
	}
	if k.bankKeeper.BlockedAddr(destination.Address()) {
		return nil, types.NewAddressValidationError(destination.Address().String(), "blocked_recipient")
	}
	// Provision both accounts in deterministic order before the bank restriction
	// resolves the recipient. SDK transaction rollback covers a failed bank send.
	addresses, err := cc.EnsureVaults(source.ID(), destination.ID())
	if err != nil {
		return nil, err
	}
	if err := k.bankKeeper.SendCoins(ctx, addresses[0], addresses[1], msg.Amount); err != nil {
		return nil, err
	}
	k.AddressEmitActivity(ctx, msg.Creator)
	cc.CommitAll()
	return &types.MsgVaultSendResponse{}, nil
}
