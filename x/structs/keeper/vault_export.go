package keeper

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"structs/x/structs/types"
)

// GetVaultOwnersForExport scans vaults for full genesis export/event replay. Validate
// each row rather than silently dropping a damaged asset ownership index.
func (k Keeper) GetVaultOwnersForExport(ctx context.Context) ([]string, error) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte(types.VaultOwnerKey))
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()
	cc := k.NewCurrentContext(ctx)
	owners := make([]string, 0)
	for ; iterator.Valid(); iterator.Next() {
		id := string(iterator.Value())
		address, err := types.VaultAddress(id)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(address, iterator.Key()) {
			return nil, fmt.Errorf("vault owner index has an invalid address for %s", id)
		}
		if _, err := cc.GetExistingAssetOwner(id); err != nil {
			return nil, err
		}
		account := k.accountKeeper.GetAccount(ctx, address)
		if account == nil {
			return nil, fmt.Errorf("vault %s lacks its auth account", id)
		}
		if account.GetPubKey() != nil {
			return nil, fmt.Errorf("vault %s has a signing public key", id)
		}
		if cc.GetPlayerIndexFromAddress(address.String()) != 0 {
			return nil, fmt.Errorf("vault %s is indexed as a player signing address", id)
		}
		owners = append(owners, id)
	}
	sort.Strings(owners)
	return owners, nil
}
