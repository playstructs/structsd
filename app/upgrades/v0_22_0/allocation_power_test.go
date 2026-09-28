package v0_22_0_test

import (
	"fmt"
	"sort"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMigrateOrphanedAllocationPower(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}
	want := map[string]uint64{}
	var removed []string
	// More than two batches, with live and dead records interleaved in key
	// order. A zero-valued orphan is still a row that must be reclaimed.
	for i := 0; i < 270; i++ {
		id := fmt.Sprintf("6-%d", i)
		attribute := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, id)
		k.SetGridAttribute(ctx, attribute, uint64(i))
		if i%3 == 1 {
			_, err := k.SetAllocationOnly(ctx, types.Allocation{Id: id, Index: uint64(i), Controller: "1-1", SourceObjectId: "1-2", Type: types.AllocationType_dynamic})
			require.NoError(t, err)
			want[attribute] = uint64(i)
		} else {
			removed = append(removed, attribute)
		}
	}
	for _, id := range []string{"5-5-999", "5-60-999", "2-6-999", "3-1-2", "2-4-3"} {
		k.SetGridAttribute(ctx, id, 17)
		want[id] = 17
	}
	beforeAllocations := k.GetAllAllocation(ctx)
	beforeCount := k.GetAllocationCount(ctx)
	ctx = ctx.WithEventManager(sdk.NewEventManager()).WithGasMeter(storetypes.NewInfiniteGasMeter())
	require.NoError(t, v22.MigrateOrphanedAllocationPower(ctx, keepers))
	t.Logf("270 allocation-power rows, 180 removed: %d gas", ctx.GasMeter().GasConsumed())
	actual := map[string]uint64{}
	for _, row := range k.GetAllGridExport(ctx) {
		actual[row.AttributeId] = row.Value
	}
	require.Equal(t, want, actual)
	require.Equal(t, beforeAllocations, k.GetAllAllocation(ctx))
	require.Equal(t, beforeCount, k.GetAllocationCount(ctx))
	require.Empty(t, k.GetGridCascadeQueue(ctx, false), "historical teardown already reduced aggregates; cleanup must not shed more load")
	sort.Strings(removed)
	var events []string
	for _, event := range ctx.EventManager().ABCIEvents() {
		require.Equal(t, "structs.structs.EventGrid", event.Type)
		parsed, err := sdk.ParseTypedEvent(event)
		require.NoError(t, err)
		row := parsed.(*types.EventGrid).GridRecord
		require.Zero(t, row.Value)
		events = append(events, row.AttributeId)
	}
	require.Equal(t, removed, events, "one exact zero event per removed row, in deterministic key order")
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, v22.MigrateOrphanedAllocationPower(ctx, keepers))
	require.Empty(t, ctx.EventManager().Events(), "repeating cleanup neither rewrites live rows nor emits duplicate clears")
}
