package app_test

import (
	"encoding/binary"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestReactorInfusionOrderMigrationRebuildsCanonicalRows(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	_, _, ownerID := newTransferPlayer(t, bApp, ctx)
	store := ctx.KVStore(bApp.GetKey(types.StoreKey))
	for _, address := range []string{"z", "a"} {
		row := types.Infusion{DestinationId: "3-999", DestinationType: types.ObjectType_reactor, Address: address, PlayerId: ownerID, Defusing: 1, Commission: math.LegacyZeroDec()}
		store.Set([]byte(types.InfusionKey+row.DestinationId+"/"+address), bApp.AppCodec().MustMarshal(&row))
	}
	// More than two clear chunks, plus corrupt metadata and an invalid ordinal
	// key inside the batch's range. None is part of canonical public state.
	for i := 0; i < 600; i++ {
		store.Set([]byte(fmt.Sprintf("Reactor/infusionOrder/stale/%08d", i)), []byte{1})
	}
	store.Set([]byte("Reactor/infusionOrder/end/3-999"), []byte{1})
	bad := append([]byte("Reactor/infusionOrder/entry/3-999/"), make([]byte, 9)...)
	binary.BigEndian.PutUint64(bad[len(bad)-9:], 1)
	store.Set(bad, []byte("bad"))
	_, _, err := bApp.StructsKeeper.ReactorInfusionOrderBatch(ctx, "3-999", 0, 2, 10)
	require.ErrorContains(t, err, "invalid reactor infusion order entry")
	store.Set([]byte(types.ReactorSlashReconcileQueue+"3-999"), []byte("old-address-cursor"))
	staleSnapshot := make([]byte, 16)
	binary.BigEndian.PutUint64(staleSnapshot[:8], 99)
	binary.BigEndian.PutUint64(staleSnapshot[8:], 100)
	store.Set([]byte("Reactor/repairSnapshot/3-999"), staleSnapshot)
	before := bApp.StructsKeeper.GetAllInfusion(ctx)
	events := len(ctx.EventManager().Events())
	for pass := 0; pass < 2; pass++ {
		require.NoError(t, v22.MigrateReactorInfusionOrder(ctx, &upgrades.Keepers{StructsKeeper: bApp.StructsKeeper}))
		require.Equal(t, uint64(2), bApp.StructsKeeper.ReactorInfusionOrderEnd(ctx, "3-999"))
		snapshot, found, err := bApp.StructsKeeper.GetReactorRepairSnapshot(ctx, "3-999")
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, keeperlib.ReactorRepairSnapshot{After: 0, Through: 2}, snapshot)
		require.Empty(t, store.Get([]byte(types.ReactorSlashReconcileQueue+"3-999")))
		entries, more, err := bApp.StructsKeeper.ReactorInfusionOrderBatch(ctx, "3-999", 0, 2, 2)
		require.NoError(t, err)
		require.False(t, more)
		require.Equal(t, []keeperlib.ReactorInfusionOrderEntry{{Ordinal: 1, Address: "a"}, {Ordinal: 2, Address: "z"}}, entries)
		require.Equal(t, before, bApp.StructsKeeper.GetAllInfusion(ctx))
		require.Equal(t, events, len(ctx.EventManager().Events()))
		require.False(t, store.Has(bad))
		for i := 0; i < 600; i++ {
			require.False(t, store.Has([]byte(fmt.Sprintf("Reactor/infusionOrder/stale/%08d", i))))
		}
	}
}
