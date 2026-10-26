package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestInfusionOwnerInventoryIncludesUncommittedChanges(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	for _, address := range []string{"a", "b", "c"} {
		k.SetInfusion(ctx, types.Infusion{DestinationId: "3-1", Address: address, PlayerId: "1-1", Defusing: 1, Commission: math.LegacyZeroDec()})
	}
	before := k.GetAllInfusion(ctx)
	cc := k.NewCurrentContext(ctx)
	check := func(owner string, addresses ...string) {
		t.Helper()
		rows, complete, work := cc.GetInfusionsByOwnerUpTo(owner, 32)
		require.True(t, complete)
		require.LessOrEqual(t, work, 32)
		var actual []string
		for _, row := range rows {
			actual = append(actual, row.Address)
		}
		require.Equal(t, addresses, actual)
	}
	check("1-1", "a", "b", "c")
	cc.GetInfusion("3-1", "a").SetPlayerId("1-2")
	cc.GetInfusion("3-1", "b").Destroy()
	created := cc.UpsertInfusion(types.ObjectType_reactor, "3-1", "d", "1-2")
	created.SetDefusing(1)
	cc.GetInfusion("3-1", "missing") // cache allocation is not existence
	check("1-1", "c")
	check("1-2", "a", "d")
	// A second re-home invalidates the previous positive inventory immediately.
	cc.GetInfusion("3-1", "a").SetPlayerId("1-3")
	check("1-2", "d")
	check("1-3", "a")
	check("1-10")
	require.Equal(t, before, k.GetAllInfusion(ctx), "inventory reads must not commit cached mutations")
	cc.CommitAll()
	cc = k.NewCurrentContext(ctx)
	check("1-1", "c")
	check("1-2", "d")
	check("1-3", "a")
}

func TestInfusionOwnerInventoryExcludesUnrelatedCacheProbes(t *testing.T) {
	for _, cached := range []int{0, 1, 16, 10000} {
		t.Run(fmt.Sprint(cached), func(t *testing.T) {
			k, ctx := keepertest.StructsKeeper(t)
			k.SetInfusion(ctx, types.Infusion{DestinationId: "3-1", Address: "a", PlayerId: "1-1", Defusing: 1, Commission: math.LegacyZeroDec()})
			makeContext := func() *keeperlib.CurrentContext {
				cc := k.NewCurrentContext(ctx)
				for i := 0; i < cached; i++ {
					cc.GetInfusion("3-1", fmt.Sprintf("probe%08d", i))
				}
				return cc
			}
			for _, limit := range []int{-1, 0, 1, 16} {
				cc := makeContext()
				events := len(ctx.EventManager().Events())
				rows, complete, work := cc.GetInfusionsByOwnerUpTo("1-1", limit)
				require.LessOrEqual(t, work, max(0, limit))
				require.Equal(t, 1 <= limit, complete)
				if complete {
					require.Len(t, rows, 1)
					require.Equal(t, "a", rows[0].Address)
					require.Equal(t, 1, work)
				} else {
					require.Empty(t, rows, "an incomplete proof must not expose a partial inventory")
				}
				require.Equal(t, events, len(ctx.EventManager().Events()))
			}
		})
	}
}

func TestInfusionOwnerInventoryDoesNotCountLoadedRowsTwice(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	for i := 0; i < keeperlib.OwnerSupplyReadBudget; i++ {
		for _, owner := range []string{"1-1", "1-2"} {
			k.SetInfusion(ctx, types.Infusion{DestinationId: "3-1", Address: fmt.Sprintf("%s-%08d", owner, i), PlayerId: owner, Defusing: 1, Commission: math.LegacyZeroDec()})
		}
	}
	cc := k.NewCurrentContext(ctx)
	for i := 0; i < 3; i++ {
		for _, owner := range []string{"1-1", "1-2"} {
			rows, complete, work := cc.GetInfusionsByOwnerUpTo(owner, keeperlib.OwnerSupplyReadBudget)
			require.True(t, complete)
			require.Len(t, rows, keeperlib.OwnerSupplyReadBudget)
			require.Equal(t, keeperlib.OwnerSupplyReadBudget, work)
		}
	}
	created := cc.UpsertInfusion(types.ObjectType_reactor, "3-1", "new", "1-1")
	created.SetDefusing(1)
	_, complete, _ := cc.GetInfusionsByOwnerUpTo("1-1", keeperlib.OwnerSupplyReadBudget)
	require.False(t, complete, "uncommitted additions still consume this owner's allowance")
	created.Destroy()
	rows, complete, _ := cc.GetInfusionsByOwnerUpTo("1-1", keeperlib.OwnerSupplyReadBudget)
	require.True(t, complete)
	require.Len(t, rows, keeperlib.OwnerSupplyReadBudget)
}
