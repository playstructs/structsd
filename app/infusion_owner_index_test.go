package app_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestInfusionOwnerIndexMigrationRebuildsWithoutPublicChanges(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	store := ctx.KVStore(bApp.GetKey(types.StoreKey))
	row := types.Infusion{DestinationId: "3-999", Address: "address", OwnerId: "0-999", Defusing: 1, Commission: math.LegacyZeroDec()}
	store.Set([]byte(types.InfusionKey+row.DestinationId+"/"+row.Address), bApp.AppCodec().MustMarshal(&row))
	store.Set([]byte("Infusion/owner/1-999/3-999/address"), []byte{1})
	store.Set([]byte("Infusion/owner/0-999/malformed"), []byte{1})
	entries, more, rows := bApp.StructsKeeper.GetInfusionOwnerEntriesUpTo(ctx, row.OwnerId, 1)
	require.Empty(t, entries)
	require.False(t, more)
	require.Equal(t, 1, rows, "malformed rows consume the read allowance")
	before := bApp.StructsKeeper.GetAllInfusion(ctx)
	for pass := 0; pass < 2; pass++ {
		events := len(ctx.EventManager().Events())
		require.NoError(t, v22.MigrateInfusionOwnerIndex(ctx, &upgrades.Keepers{StructsKeeper: bApp.StructsKeeper}))
		require.Equal(t, events, len(ctx.EventManager().Events()))
		require.Equal(t, before, bApp.StructsKeeper.GetAllInfusion(ctx))
		entries, more, rows = bApp.StructsKeeper.GetInfusionOwnerEntriesUpTo(ctx, row.OwnerId, 2)
		require.Equal(t, []keeperlib.InfusionOwnerEntry{{DestinationId: row.DestinationId, Address: row.Address}}, entries)
		require.False(t, more)
		require.Equal(t, 1, rows)
		require.False(t, store.Has([]byte("Infusion/owner/1-999/3-999/address")))
		require.False(t, store.Has([]byte("Infusion/owner/0-999/malformed")))
	}
}

func TestInfusionOwnerInventoryRefusesStaleAndMalformedIndex(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	for _, suffix := range []string{"3-999/missing", "3-999/wrong-owner", "malformed"} {
		t.Run(suffix, func(t *testing.T) {
			probe, _ := ctx.CacheContext()
			row := types.Infusion{DestinationId: "3-999", Address: "wrong-owner", OwnerId: "0-999", Defusing: 1, Commission: math.LegacyZeroDec()}
			bApp.StructsKeeper.SetInfusion(probe, row)
			store := probe.KVStore(bApp.GetKey(types.StoreKey))
			store.Set([]byte("Infusion/owner/1-999/"+suffix), []byte{1})
			for _, warm := range []bool{false, true} {
				cc := bApp.StructsKeeper.NewCurrentContext(probe)
				if warm {
					cc.GetInfusion("3-999", "wrong-owner").CheckInfusion()
				}
				rows, complete, used := cc.GetInfusionsByOwnerUpTo("1-999", 4)
				require.False(t, complete)
				require.Empty(t, rows)
				require.LessOrEqual(t, used, 4)
			}
		})
	}
}
