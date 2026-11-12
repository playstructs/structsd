package keeper_test

import (
	"math"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestSubstationConnectionCapacityMaximumArithmetic(t *testing.T) {
	for _, tc := range []struct {
		name                                 string
		capacity, load, count, maximum, want uint64
	}{
		{"legacy unlimited", 1000, 100, 3, 0, 266},
		{"capped", 1000, 100, 3, 200, 200},
		{"maximum above share", 1000, 100, 3, 300, 266},
		{"zero connections", 1000, 100, 0, 200, 200},
		{"equal maximum", 1000, 100, 4, 200, 200},
		{"fully loaded", 1000, 900, 2, 200, 0},
		{"overloaded", 1000, 1000, 2, 200, 0},
		{"round to zero", 1, 0, 2, 200, 0},
		{"uint64 boundary", math.MaxUint64, 0, 1, math.MaxUint64, types.SubstationUsableCapacity(math.MaxUint64)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx := keepertest.StructsKeeper(t)
			const id = "4-1"
			k.SetSubstation(ctx, types.Substation{Id: id})
			cc := k.NewCurrentContext(ctx)
			station := cc.GetSubstation(id)
			cc.SetGridAttribute(station.CapacityAttributeId, tc.capacity)
			cc.SetGridAttribute(station.LoadAttributeId, tc.load)
			cc.SetGridAttribute(station.ConnectionCountAttributeId, tc.count)
			station.SetConnectionCapacityMaximum(tc.maximum)
			require.Equal(t, tc.want, station.GetConnectionCapacity())
			cc.CommitAll()
			fresh := k.NewCurrentContext(ctx).GetSubstation(id)
			require.Equal(t, tc.want, fresh.GetConnectionCapacity())
			require.Equal(t, tc.maximum, fresh.GetConnectionCapacityMaximum())
			require.Equal(t, tc.capacity, fresh.GetCapacity())
		})
	}
}

// Drive the actual allocation and player connection paths, rather than only
// calling the recalculation helper. Stable capped shares must emit no write.
func TestSubstationConnectionCapacityMaximumTracksGridChanges(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const id = "4-1"
	k.SetSubstation(ctx, types.Substation{Id: id, Owner: "1-1"})
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, "1-1"), 3000)
	cc := k.NewCurrentContext(ctx)
	incoming, err := cc.NewAllocation(types.AllocationType_dynamic, "1-1", id, "", "1-1", 1000)
	require.NoError(t, err)
	station := cc.GetSubstation(id)
	for _, playerID := range []string{"1-2", "1-3", "1-4", "1-5"} {
		k.SetPlayer(ctx, types.Player{Id: playerID})
	}
	cc.GetPlayer("1-2").MigrateSubstation(id)
	cc.GetPlayer("1-3").MigrateSubstation(id)
	station.SetConnectionCapacityMaximum(200)
	cc.CommitAll()

	var outgoing *keeperlib.AllocationCache
	refresh := func() {
		cc = k.NewCurrentContext(ctx)
		station = cc.GetSubstation(id)
		incoming, _ = cc.GetAllocation(incoming.ID())
		if outgoing != nil {
			var found bool
			outgoing, found = cc.GetAllocation(outgoing.ID())
			if !found {
				outgoing = nil
			}
		}
	}
	refresh()
	eventStart := len(ctx.EventManager().Events())
	assertShare := func(want uint64, writes int) {
		t.Helper()
		require.Equal(t, want, station.GetConnectionCapacity())
		require.Equal(t, want, cc.GetPlayer("1-2").GetCapacitySecondary())
		cc.CommitAll()
		require.Equal(t, want, k.GetGridAttribute(ctx, station.ConnectionCapacityAttributeId))
		count := 0
		for _, event := range ctx.EventManager().Events()[eventStart:] {
			if event.Type != "structs.structs.EventGrid" {
				continue
			}
			parsed, err := sdk.ParseTypedEvent(abci.Event(event))
			require.NoError(t, err)
			if parsed.(*types.EventGrid).GridRecord.AttributeId == station.ConnectionCapacityAttributeId {
				count++
			}
		}
		require.Equal(t, writes, count, "one grid event is emitted for each attribute store write")
		eventStart = len(ctx.EventManager().Events())
		refresh()
	}
	eventStart = len(ctx.EventManager().Events())
	_, err = incoming.SetDynamicPower(2000)
	require.NoError(t, err)
	assertShare(200, 0)
	outgoing, err = cc.NewAllocation(types.AllocationType_dynamic, id, "", "", "1-1", 1600)
	require.NoError(t, err)
	assertShare(100, 1)
	_, err = outgoing.SetDynamicPower(1200)
	require.NoError(t, err)
	assertShare(200, 1)
	cc.GetPlayer("1-4").MigrateSubstation(id)
	assertShare(200, 0)
	cc.GetPlayer("1-5").MigrateSubstation(id)
	assertShare(150, 1)
	cc.GetPlayer("1-5").MigrateSubstation("")
	assertShare(200, 1)
	station.SetConnectionCapacityMaximum(100)
	assertShare(100, 1)
	noOpStart := len(ctx.EventManager().Events())
	station.SetConnectionCapacityMaximum(100)
	assertShare(100, 0)
	require.Len(t, ctx.EventManager().Events(), noOpStart)
	station.SetConnectionCapacityMaximum(150)
	assertShare(150, 1)
	station.SetConnectionCapacityMaximum(0)
	assertShare(200, 1)
	_, err = incoming.SetDynamicPower(1000)
	require.NoError(t, err)
	assertShare(0, 1)
	require.NoError(t, outgoing.Destroy())
	assertShare(300, 1)
	station.SetConnectionCapacityMaximum(200)
	assertShare(200, 1)
	query, err := k.Substation(ctx, &types.QueryGetSubstationRequest{Id: id})
	require.NoError(t, err)
	require.Equal(t, uint64(200), query.GridAttributes.ConnectionCapacityMaximum)
	// Deletion must clear configuration as well as the derived share.
	station.Delete("")
	cc.CommitAll()
	_, found := k.GetSubstation(ctx, id)
	require.False(t, found)
	require.Zero(t, k.GetGridAttribute(ctx, station.ConnectionCapacityMaximumAttributeId))
}
