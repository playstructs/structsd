package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"structs/x/structs/types"
)

// Pending rows are the durable work; the position only chooses where the next
// bounded pass starts. It may refer to a deleted row and need not be exported.
// Both iterator ranges share one budget, including malformed or missing keys.
func (k Keeper) dequeueInfusionDestructionBatch(ctx context.Context) []string {
	root := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	queue := prefix.NewStore(root, types.KeyPrefix(types.InfusionDestructionQueue))
	positionKey := types.KeyPrefix(types.InfusionDestructionPosition)
	position := root.Get(positionKey)
	rows := make([]string, 0, types.InfusionDestructionBudget)
	read := func(start, end []byte) {
		iterator := queue.Iterator(start, end)
		defer iterator.Close()
		for ; len(rows) < types.InfusionDestructionBudget && iterator.Valid(); iterator.Next() {
			rows = append(rows, string(iterator.Key()))
		}
	}
	if len(position) == 0 {
		read(nil, nil)
	} else {
		// Appending a zero byte excludes exactly the last key without skipping
		// longer keys that share it as a prefix.
		after := append(append([]byte(nil), position...), 0)
		read(after, nil)
		if len(rows) < types.InfusionDestructionBudget {
			read(nil, after)
		}
	}
	// Iterators are closed before any mutation of their prefix.
	for _, id := range rows {
		queue.Delete([]byte(id))
	}
	if len(rows) != 0 {
		root.Set(positionKey, []byte(rows[len(rows)-1]))
	}
	return rows
}
