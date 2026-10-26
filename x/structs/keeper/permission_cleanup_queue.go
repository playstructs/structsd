package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"structs/x/structs/types"
)

func permissionCleanupStore(k Keeper, ctx context.Context) prefix.Store {
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PermissionCleanupQueue))
}

// EnqueuePermissionCleanup marks an object as still holding permission rows
// after its own removal. No cursor is stored: deletion is destructive, so the
// next pass re-reads the object's prefix and finds whatever is left.
func (k Keeper) EnqueuePermissionCleanup(ctx context.Context, objectId string) {
	if objectId == "" {
		return
	}

	permissionCleanupStore(k, ctx).Set([]byte(objectId), []byte{0x01})
}

// ClearPermissionCleanup drops an object from the queue once it is clean.
func (k Keeper) ClearPermissionCleanup(ctx context.Context, objectId string) {
	if objectId == "" {
		return
	}

	permissionCleanupStore(k, ctx).Delete([]byte(objectId))
}

// GetPermissionCleanupQueue lists the whole queue for diagnostics and tests.
// Block processing must use the bounded batch helper.
func (k Keeper) GetPermissionCleanupQueue(ctx context.Context) (queue []string) {
	return k.getPermissionCleanupQueueBatch(ctx, int(^uint(0)>>1))
}

func (k Keeper) getPermissionCleanupQueueBatch(ctx context.Context, limit int) (queue []string) {
	store := permissionCleanupStore(k, ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid() && len(queue) < limit; iterator.Next() {
		queue = append(queue, string(iterator.Key()))
	}

	return
}

/* ProcessPermissionCleanupQueue removes a bounded number of the permission rows
 * left behind by destroyed objects.
 *
 * Runs in the EndBlocker, where the alternative was doing all of it inside
 * whichever block the object happened to be destroyed in - for an agreement,
 * that is its expiry, a height the consumer chose when they opened it.
 */
func (k Keeper) ProcessPermissionCleanupQueue(ctx context.Context) {
	queue := k.getPermissionCleanupQueueBatch(ctx, types.PermissionCleanupBudget)
	if len(queue) == 0 {
		return
	}

	budget := types.PermissionCleanupBudget

	for _, objectId := range queue {
		if budget <= 0 {
			return
		}

		// Even an already-empty object consumes a queue visit. If this uses
		// the final unit, leave it queued for a later pass with row budget.
		budget--
		if budget == 0 {
			return
		}

		deleted, morePermissions := k.ClearPermissionByObject(ctx, objectId, budget)
		budget -= len(deleted)

		moreGuildRanks := false
		if budget > 0 {
			var deletedRegisters int
			moreGuildRanks, deletedRegisters = k.clearPermissionGuildRankByObjectCount(ctx, objectId, budget)
			budget -= deletedRegisters
		} else {
			// Out of budget before the register rows were looked at, so whether
			// any remain is unknown. Assume they do rather than dropping the
			// object from the queue on an unchecked guess.
			moreGuildRanks = true
		}

		if morePermissions || moreGuildRanks {
			k.logger.Info("Permission cleanup deferred", "objectId", objectId)
			continue
		}

		k.ClearPermissionCleanup(ctx, objectId)
		k.logger.Info("Permission cleanup complete", "objectId", objectId)
	}
}
