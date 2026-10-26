package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"structs/x/structs/types"
)

// The original prefix/API is retained; generator positions now share this
// destination-scoped insertion index with reactor positions.
const reactorInfusionOrderPrefix = "Reactor/infusionOrder/"

type ReactorInfusionOrderEntry struct {
	Ordinal uint64
	Address string
}

func (k Keeper) reactorInfusionOrderStore(ctx context.Context, suffix string) prefix.Store {
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte(reactorInfusionOrderPrefix+suffix))
}

func infusionOrdinalBytes(value uint64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, value)
	return key
}

// ReactorInfusionOrderEnd captures a finite insertion cutoff in one KV read.
// Counters never decrease on deletion and ordinals are not reused on recreation.
func (k Keeper) ReactorInfusionOrderEnd(ctx context.Context, reactorID string) uint64 {
	value := k.reactorInfusionOrderStore(ctx, "end/").Get([]byte(reactorID))
	if value == nil {
		return 0
	}
	if len(value) != 8 {
		panic("invalid reactor infusion order counter")
	}
	return binary.BigEndian.Uint64(value)
}

func (k Keeper) ensureReactorInfusionOrder(ctx context.Context, row types.Infusion) {
	if row.DestinationType != types.ObjectType_reactor && row.DestinationType != types.ObjectType_struct {
		return
	}
	reverse := k.reactorInfusionOrderStore(ctx, "address/"+row.DestinationId+"/")
	if reverse.Has([]byte(row.Address)) {
		return
	}
	ordinal := k.ReactorInfusionOrderEnd(ctx, row.DestinationId)
	if ordinal == ^uint64(0) {
		panic("reactor infusion order exhausted")
	}
	ordinal++
	key := infusionOrdinalBytes(ordinal)
	k.reactorInfusionOrderStore(ctx, "entry/"+row.DestinationId+"/").Set(key, []byte(row.Address))
	reverse.Set([]byte(row.Address), key)
	k.reactorInfusionOrderStore(ctx, "end/").Set([]byte(row.DestinationId), key)
}

func (k Keeper) removeReactorInfusionOrder(ctx context.Context, destination, address string) {
	if !ObjectIdHasType(destination, types.ObjectType_reactor) && !ObjectIdHasType(destination, types.ObjectType_struct) {
		return
	}
	reverse := k.reactorInfusionOrderStore(ctx, "address/"+destination+"/")
	key := reverse.Get([]byte(address))
	if key == nil {
		return
	}
	k.reactorInfusionOrderStore(ctx, "entry/"+destination+"/").Delete(key)
	reverse.Delete([]byte(address))
}

// ReactorInfusionOrderBatch reads (after, through] in insertion order. Deleted
// positions are holes, not work to scan. Later insertions cannot extend a captured
// cutoff even when their address sorts before older addresses. A malformed row
// refuses the batch instead of silently certifying an incomplete inventory.
func (k Keeper) ReactorInfusionOrderBatch(ctx context.Context, reactorID string, after, through uint64, limit int) ([]ReactorInfusionOrderEntry, bool, error) {
	if after >= through {
		return nil, false, nil
	}
	var end []byte
	if through != ^uint64(0) {
		end = infusionOrdinalBytes(through + 1)
	}
	iterator := k.reactorInfusionOrderStore(ctx, "entry/"+reactorID+"/").Iterator(infusionOrdinalBytes(after+1), end)
	defer iterator.Close()
	var entries []ReactorInfusionOrderEntry
	for ; iterator.Valid(); iterator.Next() {
		if len(entries) >= limit {
			return entries, true, nil
		}
		if len(iterator.Key()) != 8 || len(iterator.Value()) == 0 {
			return nil, false, fmt.Errorf("invalid reactor infusion order entry for %s", reactorID)
		}
		entries = append(entries, ReactorInfusionOrderEntry{Ordinal: binary.BigEndian.Uint64(iterator.Key()), Address: string(iterator.Value())})
	}
	return entries, false, nil
}

// RebuildReactorInfusionOrder is upgrade-only. Canonical KV order makes the
// rebuild deterministic; normal cache commits assign ordinals in sorted order.
// Genesis rebuilds through ordinary infusion commits, without exporting indexes.
func (k Keeper) RebuildReactorInfusionOrder(ctx context.Context) {
	store := k.reactorInfusionOrderStore(ctx, "")
	for {
		iterator := storetypes.KVStorePrefixIterator(store, nil)
		keys := make([][]byte, 0, 256)
		for ; iterator.Valid() && len(keys) < 256; iterator.Next() {
			keys = append(keys, append([]byte(nil), iterator.Key()...))
		}
		iterator.Close()
		if len(keys) == 0 {
			break
		}
		for _, key := range keys {
			store.Delete(key)
		}
	}
	canonical := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.InfusionKey))
	iterator := storetypes.KVStorePrefixIterator(canonical, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var row types.Infusion
		k.cdc.MustUnmarshal(iterator.Value(), &row)
		k.ensureReactorInfusionOrder(ctx, row)
	}
	k.ResetReactorRepairSnapshots(ctx)
}
