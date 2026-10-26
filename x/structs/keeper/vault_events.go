package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// EmitVaultAddressEvents is a read-only genesis replay, not provisioning. A full
// scan is appropriate only when every association is requested; live transfers
// emit individual events through VaultCache.Commit instead.
func (k Keeper) EmitVaultAddressEvents(ctx context.Context) error {
	owners, err := k.GetVaultOwnersForExport(ctx)
	if err != nil {
		return err
	}
	events := make([]*types.EventVaultAddress, 0, len(owners))
	for _, owner := range owners {
		address, err := types.VaultAddress(owner)
		if err != nil {
			return err
		}
		account := k.accountKeeper.GetAccount(ctx, address)
		if account == nil {
			return fmt.Errorf("vault %s lacks its auth account", owner)
		}
		events = append(events, &types.EventVaultAddress{EventVaultAddressDetail: &types.EventVaultAddressDetail{
			OwnerId: owner, VaultAddress: address.String(), AccountNumber: account.GetAccountNumber(), AccountCreated: false,
		}})
	}
	for _, event := range events {
		if err := sdk.UnwrapSDKContext(ctx).EventManager().EmitTypedEvent(event); err != nil {
			return err
		}
	}
	return nil
}
