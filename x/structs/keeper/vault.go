package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// GetVaultOwner resolves a protocol vault in one indexed read. Validate the
// derivation too: a corrupt index must not turn an arbitrary address into an
// eligible guild-token holder or claim ownership of someone else's stake.
func (k Keeper) GetVaultOwner(ctx context.Context, address sdk.AccAddress) (string, bool) {
	store := k.storeService.OpenKVStore(ctx)
	value, err := store.Get(append([]byte(types.VaultOwnerKey), address...))
	if err != nil || len(value) == 0 {
		return "", false
	}
	ownerID := string(value)
	expected, err := types.VaultAddress(ownerID)
	if err != nil || !expected.Equals(address) {
		return "", false
	}
	return ownerID, true
}

func (k Keeper) setVaultOwner(ctx context.Context, address sdk.AccAddress, ownerID string) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(append([]byte(types.VaultOwnerKey), address...), []byte(ownerID))
}
