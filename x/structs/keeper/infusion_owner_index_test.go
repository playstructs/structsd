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

func TestInfusionOwnerIndexFollowsDirectAndCachedOwnership(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	row := types.Infusion{DestinationId: "3-1", Address: "address", PlayerId: "1-1", Defusing: 1, Commission: math.LegacyZeroDec()}
	entry := keeperlib.InfusionOwnerEntry{DestinationId: row.DestinationId, Address: row.Address}
	check := func(owner string, present bool) {
		entries, more, visited := k.GetInfusionOwnerEntriesUpTo(ctx, owner, 10)
		require.False(t, more)
		if present {
			require.Equal(t, []keeperlib.InfusionOwnerEntry{entry}, entries)
			require.Equal(t, 1, visited)
		} else {
			require.Empty(t, entries)
			require.Zero(t, visited)
		}
	}
	k.SetInfusion(ctx, row)
	check("1-1", true)
	cc := k.NewCurrentContext(ctx)
	cache := cc.GetInfusion(row.DestinationId, row.Address)
	require.NoError(t, cache.CheckInfusion())
	cache.SetPlayerId("1-2")
	cache.SetPlayerId("1-3")
	cc.CommitAll()
	check("1-1", false)
	check("1-2", false)
	check("1-3", true)
	cc = k.NewCurrentContext(ctx)
	cache = cc.GetInfusion(row.DestinationId, row.Address)
	cache.SetDefusing(2)
	cc.CommitAll()
	check("1-3", true)
	cc = k.NewCurrentContext(ctx)
	cache = cc.GetInfusion(row.DestinationId, row.Address)
	cache.Destroy()
	cc.CommitAll()
	check("1-3", false)

	row.PlayerId, row.OwnerId = "", "0-1"
	k.SetInfusion(ctx, row)
	check("0-1", true)
	row.OwnerId = "0-10"
	k.SetInfusion(ctx, row)
	check("0-1", false)
	check("0-10", true)
	k.RemoveInfusion(ctx, row.DestinationId, row.Address)
	check("0-10", false)
}

func TestInfusionOwnerIndexBoundsOwnerPrefixReads(t *testing.T) {
	measure := func(count int) uint64 {
		k, ctx := keepertest.StructsKeeper(t)
		for i := 0; i < count; i++ {
			k.SetInfusion(ctx, types.Infusion{DestinationId: "3-1", Address: fmt.Sprintf("address%08d", i), PlayerId: "1-1", Commission: math.LegacyZeroDec()})
		}
		k.SetInfusion(ctx, types.Infusion{DestinationId: "3-1", Address: "other", PlayerId: "1-10", Commission: math.LegacyZeroDec()})
		for _, limit := range []int{-1, 0, 1, 16} {
			entries, more, visited := k.GetInfusionOwnerEntriesUpTo(ctx, "1-1", limit)
			require.Equal(t, max(0, limit), visited)
			require.Len(t, entries, visited)
			require.True(t, more)
		}
		metered := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		entries, more, visited := k.GetInfusionOwnerEntriesUpTo(metered, "1-1", 16)
		require.Len(t, entries, 16)
		require.True(t, more)
		require.Equal(t, 16, visited)
		return metered.GasMeter().GasConsumed()
	}
	require.Equal(t, measure(1000), measure(10000))
}
