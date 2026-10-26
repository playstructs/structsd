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

func TestMigrateSubstationConnectionCapacity(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}
	want := map[string]uint64{}
	var changes []string
	for i := 0; i < 300; i++ {
		id := fmt.Sprintf("4-%d", i+1)
		k.SetSubstation(ctx, types.Substation{Id: id, Owner: "1-1"})
		capacity, load, count, old, share := uint64(1000), uint64(300), uint64(2), uint64(777), uint64(300)
		switch i % 4 {
		case 1:
			capacity, load, count, share = 100, 200, 3, 0 // brownout clamps before subtraction
		case 2:
			load, count, old, share = 333, 0, 667, 567 // no members uses denominator one
		case 3:
			capacity, load, count, old, share = 2, 0, 3, 0, 0 // integer rounding, no row required
		}
		for kind, value := range map[types.GridAttributeType]uint64{types.GridAttributeType_capacity: capacity, types.GridAttributeType_load: load, types.GridAttributeType_connectionCount: count} {
			attribute := keeperlib.GetGridAttributeIDByObjectId(kind, id)
			k.SetGridAttribute(ctx, attribute, value)
			want[attribute] = value
		}
		attribute := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, id)
		if i%4 != 3 {
			k.SetGridAttribute(ctx, attribute, old)
			want[attribute] = share
		}
		if old != share {
			changes = append(changes, attribute)
		}
	}
	k.SetGridAttribute(ctx, "5-6-999", 123)
	want["5-6-999"] = 123
	beforeStations := k.GetAllSubstation(ctx)
	ctx = ctx.WithEventManager(sdk.NewEventManager()).WithGasMeter(storetypes.NewInfiniteGasMeter())
	require.NoError(t, v22.MigrateSubstationConnectionCapacity(ctx, keepers))
	t.Logf("300 stations, 225 repaired: %d gas", ctx.GasMeter().GasConsumed())
	actual := map[string]uint64{}
	for _, row := range k.GetAllGridExport(ctx) {
		actual[row.AttributeId] = row.Value
	}
	require.Equal(t, want, actual)
	require.Equal(t, beforeStations, k.GetAllSubstation(ctx))
	require.Len(t, k.GetGridCascadeQueue(ctx, false), 75)
	sort.Strings(changes)
	var emitted []string
	for _, event := range ctx.EventManager().ABCIEvents() {
		require.Equal(t, "structs.structs.EventGrid", event.Type)
		parsed, err := sdk.ParseTypedEvent(event)
		require.NoError(t, err)
		row := parsed.(*types.EventGrid).GridRecord
		require.Equal(t, want[row.AttributeId], row.Value)
		emitted = append(emitted, row.AttributeId)
	}
	require.Equal(t, changes, emitted, "only changed shares get deterministic replacement events")
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, v22.MigrateSubstationConnectionCapacity(ctx, keepers))
	require.Empty(t, ctx.EventManager().Events())
}

// Overload must be queued even when its player share was already zero. Migration
// leaves allocations intact; ordinary cascade sheds the oldest one afterward.
func TestSubstationLossMigrationQueuesWithoutInlineTeardown(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}
	for _, id := range []string{"4-1", "4-2", "4-3"} {
		k.SetSubstation(ctx, types.Substation{Id: id, Owner: "1-1"})
	}
	set := func(kind types.GridAttributeType, id string, n uint64) {
		k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(kind, id), n)
	}
	set(types.GridAttributeType_capacity, "4-1", 100)
	set(types.GridAttributeType_load, "4-1", 95)
	set(types.GridAttributeType_connectionCount, "4-1", 2)
	set(types.GridAttributeType_capacity, "4-2", 45)
	// Exactly 90% is not an overload, even with no player headroom.
	set(types.GridAttributeType_capacity, "4-3", 100)
	set(types.GridAttributeType_load, "4-3", 90)
	for i, power := range []uint64{45, 50} {
		id := fmt.Sprintf("6-%d", i+1)
		destination := ""
		if i == 0 {
			destination = "4-2"
		}
		k.ImportAllocation(ctx, types.Allocation{Id: id, Index: uint64(i + 1), Type: types.AllocationType_static, SourceObjectId: "4-1", DestinationId: destination, Controller: "1-1"})
		set(types.GridAttributeType_power, id, power)
	}
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, v22.MigrateSubstationConnectionCapacity(ctx, keepers))
	require.Len(t, k.GetAllAllocation(ctx), 2)
	require.Equal(t, []string{"4-1"}, k.GetGridCascadeQueueExport(ctx))
	require.NoError(t, v22.MigrateSubstationConnectionCapacity(ctx, keepers))
	require.Equal(t, []string{"4-1"}, k.GetGridCascadeQueueExport(ctx), "queue is deduplicated")
	for _, event := range ctx.EventManager().ABCIEvents() {
		require.NotEqual(t, "structs.structs.EventDelete", event.Type)
	}
	cc := k.NewCurrentContext(ctx)
	cc.GridCascade()
	cc.CommitAll()
	_, found := k.GetAllocation(ctx, "6-1")
	require.False(t, found)
	_, found = k.GetAllocation(ctx, "6-2")
	require.True(t, found)
	cc = k.NewCurrentContext(ctx)
	require.Equal(t, uint64(100), cc.GetSubstation("4-1").GetCapacity(), "raw input is not rewritten")
	require.Equal(t, uint64(50), cc.GetSubstation("4-1").GetLoad())
	require.Equal(t, uint64(20), cc.GetSubstation("4-1").GetConnectionCapacity())
	require.Zero(t, cc.GetSubstation("4-2").GetCapacity())
	require.Empty(t, k.GetGridCascadeQueueExport(ctx))
}
