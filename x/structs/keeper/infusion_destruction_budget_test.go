package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

func TestInfusionDestructionBoundsEveryQueuedAttempt(t *testing.T) {
	for _, shape := range []string{"missing", "malformed", "empty"} {
		t.Run(shape, func(t *testing.T) {
			k, ctx := keepertest.StructsKeeper(t)
			count := types.InfusionDestructionBudget*2 + 1
			for i := 0; i < count; i++ {
				address := fmt.Sprintf("address%06d", i)
				id := "3-1-" + address
				if shape == "malformed" { id = fmt.Sprintf("!%06d", i) }
				if shape == "empty" {
					k.SetInfusion(ctx, types.Infusion{DestinationId: "3-1", Address: address, Commission: math.LegacyZeroDec()})
				}
				require.NoError(t, k.AppendInfusionDestructionQueue(ctx, id))
			}
			for pass := 0; pass < 3; pass++ {
				cc := k.NewCurrentContext(ctx)
				cc.ProcessInfusionDestructionQueue()
				cc.CommitAll()
				removed := types.InfusionDestructionBudget
				if pass == 2 { removed = 1 }
				count -= removed
				require.Len(t, k.GetInfusionDestructionQueueExport(ctx), count)
				if shape == "empty" { require.Len(t, k.GetAllInfusion(ctx), count) }
			}
		})
	}
}

func TestInfusionDestructionRotatesPastReplenishedEarlierKeys(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	for i := 0; i < types.InfusionDestructionBudget; i++ {
		require.NoError(t, k.AppendInfusionDestructionQueue(ctx, fmt.Sprintf("!early%06d", i)))
	}
	require.NoError(t, k.AppendInfusionDestructionQueue(ctx, "3-1-later"))
	cc := k.NewCurrentContext(ctx)
	cc.ProcessInfusionDestructionQueue()
	cc.CommitAll()
	require.Contains(t, k.GetInfusionDestructionQueueExport(ctx), "3-1-later")
	for i := 0; i < types.InfusionDestructionBudget; i++ {
		require.NoError(t, k.AppendInfusionDestructionQueue(ctx, fmt.Sprintf("!early%06d", i)))
	}
	cc = k.NewCurrentContext(ctx)
	cc.ProcessInfusionDestructionQueue()
	cc.CommitAll()
	require.NotContains(t, k.GetInfusionDestructionQueueExport(ctx), "3-1-later", "new earlier keys cannot starve pending cleanup")
}

func TestInfusionDestructionReadCostDoesNotGrowWithBacklog(t *testing.T) {
	cost := func(count int) uint64 {
		k, ctx := keepertest.StructsKeeper(t)
		for i := 0; i < count; i++ {
			require.NoError(t, k.AppendInfusionDestructionQueue(ctx, fmt.Sprintf("3-1-address%06d", i)))
		}
		meter := storetypes.NewGasMeter(20000000)
		cc := k.NewCurrentContext(ctx.WithGasMeter(meter))
		cc.ProcessInfusionDestructionQueue()
		cc.CommitAll()
		used := meter.GasConsumed()
		require.Len(t, k.GetInfusionDestructionQueueExport(ctx), count-types.InfusionDestructionBudget)
		return used
	}
	require.Equal(t, cost(1000), cost(10000))
}

func TestInfusionDestructionRechecksRowsThatRegainedBacking(t *testing.T) {
	for _, defusing := range []bool{false, true} {
		k, ctx := keepertest.StructsKeeper(t)
		row := types.Infusion{DestinationId: "3-1", Address: "address", Commission: math.LegacyZeroDec()}
		k.SetInfusion(ctx, row)
		require.NoError(t, k.AppendInfusionDestructionQueue(ctx, "3-1-address"))
		// Simulate a later staking hook refilling a record before its queued
		// cleanup is serviced. A queue entry is not permission to delete it.
		if defusing { row.Defusing = 10 } else { row.Fuel = 10 }
		k.SetInfusion(ctx, row)
		cc := k.NewCurrentContext(ctx)
		cc.ProcessInfusionDestructionQueue()
		cc.CommitAll()
		after, found := k.GetInfusion(ctx, row.DestinationId, row.Address)
		require.True(t, found)
		require.Equal(t, row, after)
		require.Empty(t, k.GetInfusionDestructionQueueExport(ctx))
	}
}
