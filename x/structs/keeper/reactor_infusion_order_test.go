package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestReactorInfusionOrderPreservesCutoffAcrossUpdatesAndRecreation(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	row := types.Infusion{DestinationType: types.ObjectType_reactor, DestinationId: "3-1", Address: "z", PlayerId: "1-1", Defusing: 1, Commission: math.LegacyZeroDec()}
	k.SetInfusion(ctx, row)
	row.Address = "a"
	k.SetInfusion(ctx, row)
	cutoff := k.ReactorInfusionOrderEnd(ctx, "3-1")
	require.Equal(t, uint64(2), cutoff)
	cc := k.NewCurrentContext(ctx)
	cc.GetInfusion("3-1", "a").SetPlayerId("1-2")
	cc.CommitAll()
	require.Equal(t, cutoff, k.ReactorInfusionOrderEnd(ctx, "3-1"))
	cc = k.NewCurrentContext(ctx)
	cc.GetInfusion("3-1", "z").Destroy()
	cc.CommitAll()
	row.Address = "z"
	k.SetInfusion(ctx, row)
	require.Equal(t, uint64(3), k.ReactorInfusionOrderEnd(ctx, "3-1"))
	entries, more, err := k.ReactorInfusionOrderBatch(ctx, "3-1", 0, cutoff, 10)
	require.NoError(t, err)
	require.False(t, more)
	require.Equal(t, []keeperlib.ReactorInfusionOrderEntry{{Ordinal: 2, Address: "a"}}, entries)
	entries, more, err = k.ReactorInfusionOrderBatch(ctx, "3-1", cutoff, ^uint64(0), 1)
	require.NoError(t, err)
	require.False(t, more)
	require.Equal(t, []keeperlib.ReactorInfusionOrderEntry{{Ordinal: 3, Address: "z"}}, entries)
	k.RemoveInfusion(ctx, "3-1", "z")
	k.RemoveInfusion(ctx, "3-1", "z")
	require.Equal(t, uint64(3), k.ReactorInfusionOrderEnd(ctx, "3-1"))
	row.DestinationId, row.DestinationType = "5-1", types.ObjectType_struct
	k.SetInfusion(ctx, row)
	require.Equal(t, uint64(1), k.ReactorInfusionOrderEnd(ctx, "5-1"), "generators share the destination-scoped order index")
	k.RemoveInfusion(ctx, "5-1", "z")
	entries, more, err = k.ReactorInfusionOrderBatch(ctx, "5-1", 0, 1, 1)
	require.NoError(t, err)
	require.False(t, more)
	require.Empty(t, entries, "generator deletion must clear the same index it created")
	k.SetInfusion(ctx, row)
	require.Equal(t, uint64(2), k.ReactorInfusionOrderEnd(ctx, "5-1"), "recreation cannot reuse a captured ordinal")
}

func TestReactorInfusionOrderUsesSortedCacheCommits(t *testing.T) {
	for _, order := range [][]string{{"z", "a", "m"}, {"m", "a", "z"}} {
		k, ctx := keepertest.StructsKeeper(t)
		cc := k.NewCurrentContext(ctx)
		for _, address := range order {
			cc.UpsertInfusion(types.ObjectType_reactor, "3-1", address, "1-1").SetDefusing(1)
		}
		cc.CommitAll()
		entries, more, err := k.ReactorInfusionOrderBatch(ctx, "3-1", 0, 3, 3)
		require.NoError(t, err)
		require.False(t, more)
		require.Equal(t, []keeperlib.ReactorInfusionOrderEntry{{1, "a"}, {2, "m"}, {3, "z"}}, entries)
	}
}

func TestReactorInfusionOrderBoundsReadsAndIgnoresLaterInsertions(t *testing.T) {
	measure := func(count int) uint64 {
		k, ctx := keepertest.StructsKeeper(t)
		for i := 0; i < count; i++ {
			k.SetInfusion(ctx, types.Infusion{DestinationType: types.ObjectType_reactor, DestinationId: "3-1", Address: fmt.Sprintf("address%08d", i), Commission: math.LegacyZeroDec()})
		}
		k.SetInfusion(ctx, types.Infusion{DestinationType: types.ObjectType_reactor, DestinationId: "3-10", Address: "other", Commission: math.LegacyZeroDec()})
		for _, limit := range []int{-1, 0, 1, 16} {
			entries, more, err := k.ReactorInfusionOrderBatch(ctx, "3-1", 0, 20, limit)
			require.NoError(t, err)
			require.Len(t, entries, max(0, limit))
			require.True(t, more)
		}
		metered := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		entries, more, err := k.ReactorInfusionOrderBatch(metered, "3-1", 0, 20, 20)
		require.NoError(t, err)
		require.False(t, more)
		require.Len(t, entries, 20)
		require.Equal(t, uint64(20), entries[19].Ordinal)
		return metered.GasMeter().GasConsumed()
	}
	require.Equal(t, measure(1000), measure(10000))
}
