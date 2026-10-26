package keeper

import (
	"context"
	"encoding/binary"
	"sort"
	"strconv"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

const addressOwnerIndexPrefix = "Address/owner/"

func addressOwnerPrefix(index uint64) []byte {
	return []byte(addressOwnerIndexPrefix + strconv.FormatUint(index, 10) + "/")
}

func (k Keeper) updateAddressOwnerIndex(ctx context.Context, address string, previous, next uint64) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	if previous != 0 && previous != next {
		store.Delete(append(addressOwnerPrefix(previous), []byte(address)...))
	}
	if next != 0 {
		store.Set(append(addressOwnerPrefix(next), []byte(address)...), []byte{1})
	}
}

// RebuildAddressOwnerIndex is upgrade-only. Genesis populates it through the
// canonical association setter. Each runtime lookup uses one owner's prefix.
func (k Keeper) RebuildAddressOwnerIndex(ctx context.Context) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte(addressOwnerIndexPrefix))
	it := store.Iterator(nil, nil)
	var keys [][]byte
	for ; it.Valid(); it.Next() {
		keys = append(keys, append([]byte{}, it.Key()...))
	}
	it.Close()
	for _, key := range keys {
		store.Delete(key)
	}
	associations := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte(types.AddressPlayerKey))
	it = associations.Iterator(nil, nil)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		k.updateAddressOwnerIndex(ctx, string(it.Key()), 0, binary.BigEndian.Uint64(it.Value()))
	}
}

func (cache *AddressCache) indexCachedPlayer() {
	index := uint64(0)
	if cache.Loaded && !cache.Deleted {
		index = cache.PlayerIndex
	}
	if cache.cachedPlayerIndex != 0 && cache.cachedPlayerIndex != index {
		delete(cache.CC.addressesByPlayer[cache.cachedPlayerIndex], cache.Address)
	}
	if index != 0 {
		if cache.CC.addressesByPlayer[index] == nil {
			cache.CC.addressesByPlayer[index] = make(map[string]*AddressCache)
		}
		cache.CC.addressesByPlayer[index][cache.Address] = cache
	}
	cache.cachedPlayerIndex = index
}

// getOwnerStakeAddressesUpTo resolves the complete address set or refuses the
// uncertain bonus. The protocol vault is included only when associated. Cached
// registration/revocation is overlaid without charging another owner's rows.
func (cc *CurrentContext) getOwnerStakeAddressesUpTo(owner string, limit int) ([]sdk.AccAddress, bool) {
	kind, index, err := types.ParseAssetOwnerID(owner)
	if err != nil || limit < 1 {
		return nil, false
	}
	addresses := make(map[string]bool)
	vault, err := types.VaultAddress(owner)
	if err != nil {
		return nil, false
	}
	if bound, found := cc.k.GetVaultOwner(cc.ctx, vault); found {
		if bound != owner {
			return nil, false
		}
		addresses[vault.String()] = true
	}
	if kind == types.ObjectType_player {
		if len(cc.addressesByPlayer[index]) > limit {
			return nil, false
		}
		store := prefix.NewStore(runtime.KVStoreAdapter(cc.k.storeService.OpenKVStore(cc.ctx)), addressOwnerPrefix(index))
		it := store.Iterator(nil, nil)
		work := len(addresses)
		for ; it.Valid(); it.Next() {
			if work >= limit {
				it.Close()
				return nil, false
			}
			work++
			address := string(it.Key())
			current := cc.GetPlayerIndexFromAddress(address)
			if current != index {
				if !cc.addresses[address].Changed {
					it.Close()
					return nil, false
				}
				continue
			}
			addresses[address] = true
		}
		it.Close()
		for address := range cc.addressesByPlayer[index] {
			addresses[address] = true
		}
	}
	if len(addresses) > limit {
		return nil, false
	}
	keys := make([]string, 0, len(addresses))
	for key := range addresses {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]sdk.AccAddress, 0, len(keys))
	for _, key := range keys {
		address, err := sdk.AccAddressFromBech32(key)
		if err != nil {
			return nil, false
		}
		result = append(result, address)
	}
	return result, true
}
