package keeper

import (
	"bytes"
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"structs/x/structs/types"
)

// PruneOrphanedAllocationPower is an upgrade-only inventory walk. Historical
// Destroy calls removed the allocation but left its power attribute forever.
// Only that attribute namespace is visited, and an existing allocation's row
// is always preserved. No aggregate load/capacity is decremented a second time.
// Iterate in bounded memory batches and close each iterator before deleting.
// The total walk is deliberately confined to the atomic upgrade handler.
func (k Keeper) PruneOrphanedAllocationPower(ctx context.Context) uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	grid := prefix.NewStore(store, types.KeyPrefix(types.GridAttributeKey))
	allocations := prefix.NewStore(store, types.KeyPrefix(types.AllocationKey))
	attributePrefix := fmt.Sprintf("%d-", types.GridAttributeType_power)
	namespace := []byte(fmt.Sprintf("%s%d-", attributePrefix, types.ObjectType_allocation))
	start, end := namespace, storetypes.PrefixEndBytes(namespace)
	var removed uint64
	for {
		it := grid.Iterator(start, end)
		var orphaned []string
		for visited := 0; it.Valid() && visited < 128; visited++ {
			key := bytes.Clone(it.Key())
			start = append(bytes.Clone(key), 0)
			if !allocations.Has(key[len(attributePrefix):]) {
				orphaned = append(orphaned, string(key))
			}
			it.Next()
		}
		more := it.Valid()
		it.Close()
		for _, id := range orphaned {
			k.ClearGridAttribute(ctx, id)
			removed++
		}
		if !more {
			return removed
		}
	}
}
