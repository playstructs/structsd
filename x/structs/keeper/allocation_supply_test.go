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

func TestSubstationHeadroomCostIndependentOfRootInventory(t *testing.T) {
	measure := func(rows int) (uint64, uint64) {
		k, ctx := keepertest.StructsKeeper(t)
		owner := testAppendPlayer(k, ctx, types.Player{})
		k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner.Id), 2)
		k.SetSubstation(ctx, types.Substation{Id: "4-1", Owner: owner.Id})
		cc := k.NewCurrentContext(ctx)
		cc.GenesisImportAllocation(types.Allocation{Id: "6-1", SourceObjectId: owner.Id, DestinationId: "4-1", Type: types.AllocationType_dynamic}, 2)
		cc.CommitAll()
		// Zero-output generator rows isolate inventory cost from live staking.
		// The capacity is fixture accounting, not a claim of generator backing.
		for i := 0; i < rows; i++ {
			k.SetInfusion(ctx, types.Infusion{DestinationId: "5-1", DestinationType: types.ObjectType_struct,
				Address: fmt.Sprintf("address%08d", i), PlayerId: owner.Id, Defusing: 1, Commission: math.LegacyZeroDec()})
		}

		metered := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		available := k.NewCurrentContext(metered).GetSubstation("4-1").GetAvailableCapacity()
		return available, metered.GasMeter().GasConsumed()
	}
	// One station, its incoming row and the player root already used 3 units.
	available, _ := measure(2048 - 3)
	require.Equal(t, uint64(1), available)
	available, boundedGas := measure(2048 - 2)
	require.Equal(t, uint64(1), available, "diagnostic exhaustion must not disable stored supply")
	available, largerGas := measure(10000)
	require.Equal(t, uint64(1), available)
	require.Equal(t, boundedGas, largerGas)
}

func TestSubstationHeadroomCostIndependentOfAncestry(t *testing.T) {
	measure := func(links int) (uint64, uint64) {
		k, ctx := keepertest.StructsKeeper(t)
		root := testAppendPlayer(k, ctx, types.Player{})
		k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, root.Id), 2)
		cc := k.NewCurrentContext(ctx)
		source := root.Id
		for i := 1; i <= links; i++ {
			station := fmt.Sprintf("4-%08d", i)
			k.SetSubstation(ctx, types.Substation{Id: station, Owner: root.Id})
			cc.GenesisImportAllocation(types.Allocation{Id: fmt.Sprintf("6-%08d", i), SourceObjectId: source,
				DestinationId: station, Controller: root.Id, Type: types.AllocationType_dynamic}, 2)
			source = station
		}
		cc.CommitAll()
		metered := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		available := k.NewCurrentContext(metered).GetSubstation(source).GetAvailableCapacity()
		return available, metered.GasMeter().GasConsumed()
	}
	available, gas := measure(2048/2 - 1)
	require.Equal(t, uint64(1), available, "a complete proof within the allowance stays usable")
	require.Less(t, gas, uint64(20_000_000))
	available, boundedGas := measure(2048/2 + 1)
	require.Equal(t, uint64(1), available, "large legacy graphs retain local headroom until cascade")
	available, largerGas := measure(10000)
	require.Equal(t, uint64(1), available)
	require.Equal(t, boundedGas, largerGas, "unvisited ancestry cannot grow read cost")
	require.Less(t, boundedGas, uint64(20_000_000))
	t.Logf("within_budget_gas=%d exhausted_budget_gas=%d", gas, boundedGas)
}

