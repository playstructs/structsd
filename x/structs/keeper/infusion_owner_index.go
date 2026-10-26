package keeper

import (
	"context"
	"strings"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"structs/x/structs/types"
)

const infusionOwnerIndexPrefix = "Infusion/owner/"

type InfusionOwnerEntry struct {
	DestinationId string
	Address       string
}

func infusionOwnerStore(k Keeper, ctx context.Context, owner string) prefix.Store {
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte(infusionOwnerIndexPrefix+owner+"/"))
}

func (k Keeper) changeInfusionOwnerIndex(ctx context.Context, previousOwner string, infusion types.Infusion) {
	owner := infusion.AssetOwnerID()
	if previousOwner == owner {
		return
	}
	key := []byte(infusion.DestinationId + "/" + infusion.Address)
	if previousOwner != "" {
		infusionOwnerStore(k, ctx, previousOwner).Delete(key)
	}
	if owner != "" {
		infusionOwnerStore(k, ctx, owner).Set(key, []byte{1})
	}
}

// GetInfusionOwnerEntriesUpTo returns a bounded prefix, not a complete owner
// inventory when more is true. Malformed rows still consume the returned row
// count; callers must resolve records and verify their current owner separately.
func (k Keeper) GetInfusionOwnerEntriesUpTo(ctx context.Context, owner string, limit int) (entries []InfusionOwnerEntry, more bool, rowsRead int) {
	iterator := storetypes.KVStorePrefixIterator(infusionOwnerStore(k, ctx, owner), nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		if rowsRead >= limit {
			return entries, true, rowsRead
		}
		rowsRead++
		destination, address, ok := strings.Cut(string(iterator.Key()), "/")
		if ok && destination != "" && address != "" {
			entries = append(entries, InfusionOwnerEntry{DestinationId: destination, Address: address})
		}
	}
	return entries, false, rowsRead
}

// RebuildInfusionOwnerIndex is upgrade-only. Clear bounded chunks, then stream
// canonical infusion records while writing a different prefix. Genesis builds
// the same derived index through ordinary infusion commits.
func (k Keeper) RebuildInfusionOwnerIndex(ctx context.Context) {
	root := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	index := prefix.NewStore(root, []byte(infusionOwnerIndexPrefix))
	for {
		iterator := storetypes.KVStorePrefixIterator(index, nil)
		keys := make([][]byte, 0, 256)
		for ; iterator.Valid() && len(keys) < 256; iterator.Next() {
			keys = append(keys, append([]byte(nil), iterator.Key()...))
		}
		iterator.Close()
		if len(keys) == 0 {
			break
		}
		for _, key := range keys {
			index.Delete(key)
		}
	}
	store := prefix.NewStore(root, types.KeyPrefix(types.InfusionKey))
	iterator := storetypes.KVStorePrefixIterator(store, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var infusion types.Infusion
		k.cdc.MustUnmarshal(iterator.Value(), &infusion)
		k.changeInfusionOwnerIndex(ctx, "", infusion)
	}
}
