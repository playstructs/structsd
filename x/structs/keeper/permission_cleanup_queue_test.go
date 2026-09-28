package keeper_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

/* Regression suite for unbounded permission cleanup.
 *
 * Destroying an object clears every permission granted on it, and the count of
 * those is chosen by whoever owns the object. Agreement expiry reaches that
 * cleanup from the EndBlocker, which has no gas meter, and the expiry height is
 * chosen by the consumer when they open the agreement - so the cost of one block
 * was whatever had been granted before it, materialised all at once as key
 * copies, returned strings and retained events.
 *
 * The remainder is safe to defer, which is what separates this from an agreement
 * expiry: a permission is only ever consulted after its object has been loaded,
 * and a destroyed object cannot be. Rows left behind grant nothing, so this is
 * garbage collection and it can take as many blocks as it needs.
 */

func grantMany(t *testing.T, k keeperlib.Keeper, ctx sdk.Context, objectId string, count int) {
	t.Helper()

	for i := 0; i < count; i++ {
		playerId := fmt.Sprintf("1-%d", i+1)
		k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(objectId, playerId), types.PermPlay)
	}
	require.Len(t, k.GetPermissionsByObject(ctx, objectId), count)
}

func TestPermissionCleanup_BoundedPerPass(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const objectId = "5-1"

	grantMany(t, k, ctx, objectId, types.PermissionCleanupBudget+40)

	deleted, more := k.ClearPermissionByObject(ctx, objectId, types.PermissionCleanupBudget)
	require.Len(t, deleted, types.PermissionCleanupBudget, "one pass must stop at the budget")
	require.True(t, more, "and must report that rows remain")

	require.Len(t, k.GetPermissionsByObject(ctx, objectId), 40)
}

func TestPermissionCleanup_ExactCountIsNotTruncation(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const objectId = "5-1"

	grantMany(t, k, ctx, objectId, 10)

	deleted, more := k.ClearPermissionByObject(ctx, objectId, 10)
	require.Len(t, deleted, 10)
	require.False(t, more, "clearing exactly what is there is not truncation")
	require.Empty(t, k.GetPermissionsByObject(ctx, objectId))
}

func TestPermissionCleanup_RankOnlyQueueSharesOneBlockBudget(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const objects = 3
	for i := 0; i < objects; i++ {
		objectId := fmt.Sprintf("5-%d", i+1)
		for j := 0; j < types.PermissionCleanupBudget; j++ {
			k.SetGuildRankPermissionStoreOnly(ctx, objectId, fmt.Sprintf("0-%d", j+1), types.PermPlay, 3)
		}
		k.EnqueuePermissionCleanup(ctx, objectId)
	}
	k.ProcessPermissionCleanupQueue(ctx)
	// Each deleted register emits one event in this fixture. A queue visit
	// also costs budget, so the first pass cannot delete a full block's rows.
	deleted := 0
	for _, event := range ctx.EventManager().Events() {
		if event.Type == "structs.structs.EventGuildRankPermission" {
			deleted++
		}
	}
	require.Equal(t, types.PermissionCleanupBudget-1, deleted)
	require.Len(t, k.GetPermissionCleanupQueue(ctx), objects)
	for pass := 0; pass < objects+3; pass++ {
		k.ProcessPermissionCleanupQueue(ctx)
	}
	require.Empty(t, k.GetPermissionCleanupQueue(ctx))
	for i := 0; i < objects; i++ {
		for j := 0; j < types.PermissionCleanupBudget; j++ {
			register := k.ReadGuildRankRegister(ctx, fmt.Sprintf("5-%d", i+1), fmt.Sprintf("0-%d", j+1))
			require.Equal(t, [types.PermissionBitCount]uint64{}, register)
		}
	}
}

func TestPermissionCleanup_EmptyQueueEntriesStillConsumeBudget(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	for i := 0; i < types.PermissionCleanupBudget*2; i++ {
		k.EnqueuePermissionCleanup(ctx, fmt.Sprintf("5-%08d", i+1))
	}
	k.ProcessPermissionCleanupQueue(ctx)
	require.GreaterOrEqual(t, len(k.GetPermissionCleanupQueue(ctx)), types.PermissionCleanupBudget)
	for pass := 0; pass < 4; pass++ {
		k.ProcessPermissionCleanupQueue(ctx)
	}
	require.Empty(t, k.GetPermissionCleanupQueue(ctx))
}

func TestPermissionCleanup_BacklogDoesNotIncreaseBatchCost(t *testing.T) {
	var costs []uint64
	for _, size := range []int{types.PermissionCleanupBudget * 2, types.PermissionCleanupBudget * 16} {
		k, ctx := keepertest.StructsKeeper(t)
		for i := 0; i < size; i++ {
			k.EnqueuePermissionCleanup(ctx, fmt.Sprintf("5-%08d", i))
		}
		grantMany(t, k, ctx, "5-00000000", types.PermissionCleanupBudget*2)
		metered := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		k.ProcessPermissionCleanupQueue(metered)
		costs = append(costs, metered.GasMeter().GasConsumed())
		require.Len(t, k.GetPermissionCleanupQueue(ctx), size)
	}
	require.Less(t, costs[1], costs[0]*5/4,
		"the worker must not load the entire permission cleanup queue: narrow=%d wide=%d", costs[0], costs[1])
}

/* TestPermissionCleanup_QueueDrainsAcrossBlocks is the regression itself: an
 * object with more grants than one pass must be queued, and must finish - the
 * remainder deferred, not dropped.
 */
func TestPermissionCleanup_QueueDrainsAcrossBlocks(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const objectId = "5-1"

	total := types.PermissionCleanupBudget*2 + 15
	grantMany(t, k, ctx, objectId, total)

	cc := k.NewCurrentContext(ctx)
	cc.ClearPermissionsForObject(objectId)
	cc.CommitAll()

	require.Equal(t, []string{objectId}, k.GetPermissionCleanupQueue(ctx),
		"an object larger than one pass must be queued")
	require.Greater(t, len(k.GetPermissionsByObject(ctx, objectId)), 0,
		"and must still have rows outstanding")

	blocks := 0
	for len(k.GetPermissionCleanupQueue(ctx)) > 0 {
		blocks++
		require.Less(t, blocks, 20, "the cleanup queue is not converging")
		k.ProcessPermissionCleanupQueue(ctx)
	}

	require.Empty(t, k.GetPermissionsByObject(ctx, objectId),
		"every row must be gone by the time the object leaves the queue")
	require.Greater(t, blocks, 1, "this object is supposed to need more than one block")
}

// An object small enough to finish in one pass must never touch the queue.
func TestPermissionCleanup_SmallObjectNeverQueues(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const objectId = "5-1"

	grantMany(t, k, ctx, objectId, 5)

	cc := k.NewCurrentContext(ctx)
	cc.ClearPermissionsForObject(objectId)
	cc.CommitAll()

	require.Empty(t, k.GetPermissionCleanupQueue(ctx))
	require.Empty(t, k.GetPermissionsByObject(ctx, objectId))
}

/* TestPermissionCleanup_CacheIsEvictedByPrefix pins why the cache eviction does
 * not follow the list of rows actually deleted.
 *
 * A cache entry marked Changed is written back at CommitAll. Evicting by the
 * deleted list only reaches entries that had a row on disk, so a permission
 * *created* in the same operation that destroys the object - present in the
 * cache, absent from the store, therefore absent from the list - survives the
 * clear and is written back afterwards. On an object small enough not to be
 * queued, nothing ever comes back for it: the grant outlives the object
 * permanently.
 *
 * Evicting by prefix covers everything for the object whether this pass deleted
 * its row or not.
 */
func TestPermissionCleanup_CacheIsEvictedByPrefix(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const objectId = "5-1"

	// Small enough that the clear finishes in one pass and the object is never
	// queued - which is what makes a survivor permanent.
	grantMany(t, k, ctx, objectId, 3)

	cc := k.NewCurrentContext(ctx)

	// A grant created in this same operation. It has no row on disk yet, so it
	// cannot appear in the list of rows the clear deleted.
	fresh := keeperlib.GetObjectPermissionIDBytes(objectId, "1-999")
	cc.SetPermissions(fresh, types.PermUpdate)

	cc.ClearPermissionsForObject(objectId)
	cc.CommitAll()

	require.Empty(t, k.GetPermissionCleanupQueue(ctx),
		"fixture sanity: this object must not be queued, or a later pass would hide the bug")
	require.Empty(t, k.GetPermissionsByObject(ctx, objectId),
		"a permission created in the destroying operation was written back and outlived its object")
}
