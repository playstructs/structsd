package keeper_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

func TestIncomingAllocationReadBoundsRawRowsAndReportsIncompleteSupply(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("6-%02d", i)
		require.NoError(t, k.SetAllocationDestinationIndex(ctx, "4-1", id))
		if i > 1 {
			_, err := k.SetAllocationOnly(ctx, types.Allocation{Id: id, DestinationId: "4-1"})
			require.NoError(t, err)
		}
	}
	require.NoError(t, k.SetAllocationDestinationIndex(ctx, "4-10", "6-other"))
	for _, limit := range []int{-1, 0, 1, 2, 3, 5, 6} {
		cc := k.NewCurrentContext(ctx)
		allocations, more, rows := cc.GetAllocationsByDestinationUpTo("4-1", limit)
		wantRows := max(0, min(limit, 5))
		require.Equal(t, wantRows, rows)
		require.Equal(t, wantRows < 5, more)
		require.Len(t, allocations, max(0, wantRows-2), "missing rows consume budget without being returned")
		for _, allocation := range allocations {
			require.Equal(t, "4-1", allocation.GetAllocation().DestinationId)
		}
	}
	ids, more := k.GetAllocationIdsByDestinationIndexUpTo(ctx, "4-missing", 0)
	require.Empty(t, ids)
	require.False(t, more)
}

func TestIncomingAllocationReadUsesCurrentContextAndRejectsStaleDestination(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	cc := k.NewCurrentContext(ctx)
	// GenesisImportAllocation populates the context and index before its record
	// is committed. A direct keeper reload would incorrectly miss this supply.
	cc.GenesisImportAllocation(types.Allocation{Id: "6-1", DestinationId: "4-1"}, 0)
	_, stored := k.GetAllocation(ctx, "6-1")
	require.False(t, stored)
	allocation, found := cc.GetAllocation("6-1")
	require.True(t, found)
	rows, more, visited := cc.GetAllocationsByDestinationUpTo("4-1", 1)
	require.Equal(t, 1, visited)
	require.False(t, more)
	require.Len(t, rows, 1)
	require.Same(t, allocation, rows[0])

	// Keep the old index deliberately, as can happen in stale state. Neither a
	// moved cache nor a deleted cache may be interpreted as current backing.
	allocation.SetAllocationDestinationId("4-2")
	rows, more, visited = cc.GetAllocationsByDestinationUpTo("4-1", 1)
	require.Empty(t, rows)
	require.False(t, more)
	require.Equal(t, 1, visited)
	require.NoError(t, k.SetAllocationDestinationIndex(ctx, "4-2", "6-1"))
	rows, _, _ = cc.GetAllocationsByDestinationUpTo("4-2", 1)
	require.Len(t, rows, 1)
	require.Same(t, allocation, rows[0])
	allocation.Deleted = true
	rows, more, visited = cc.GetAllocationsByDestinationUpTo("4-2", 1)
	require.Empty(t, rows)
	require.False(t, more)
	require.Equal(t, 1, visited)
}

func TestIncomingAllocationReadCostDoesNotGrowWithUnvisitedBacklog(t *testing.T) {
	measure := func(count int) uint64 {
		k, ctx := keepertest.StructsKeeper(t)
		for i := 0; i < count; i++ {
			require.NoError(t, k.SetAllocationDestinationIndex(ctx, "4-1", fmt.Sprintf("6-%08d", i)))
		}
		metered := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		allocations, more, rows := k.NewCurrentContext(metered).GetAllocationsByDestinationUpTo("4-1", 16)
		require.Empty(t, allocations)
		require.True(t, more)
		require.Equal(t, 16, rows)
		return metered.GasMeter().GasConsumed()
	}
	require.Equal(t, measure(1000), measure(10000))
}
