package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

func TestPlanetOwnerIndexMigrationBackfillsLegacyRecords(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	store := ctx.KVStore(bApp.GetKey(types.StoreKey))
	planet := types.Planet{Id: "2-999", Owner: "1-1", Status: types.PlanetStatus_active}
	// Legacy storage has no owner index. A stale row also verifies that the
	// rebuild is derived from planet records rather than trusting old indexes.
	store.Set([]byte(types.PlanetKey+planet.Id), bApp.AppCodec().MustMarshal(&planet))
	store.Set([]byte("Planet/owner/1-1/2-1000"), []byte{1})
	for pass := 0; pass < 2; pass++ {
		beforeEvents := len(ctx.EventManager().Events())
		require.NoError(t, v22.MigratePlanetOwnerIndex(ctx, &upgrades.Keepers{StructsKeeper: bApp.StructsKeeper}))
		require.Len(t, ctx.EventManager().Events(), beforeEvents, "derived-index rebuild does not change public records")
		page, err := bApp.StructsKeeper.PlanetAllByPlayer(ctx, &types.QueryAllPlanetByPlayerRequest{PlayerId: "1-1"})
		require.NoError(t, err)
		require.Equal(t, []types.Planet{planet}, page.Planet)
		stored, found := bApp.StructsKeeper.GetPlanet(ctx, planet.Id)
		require.True(t, found)
		require.Equal(t, planet, stored)
	}
}
