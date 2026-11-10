package keeper_test

import (
	"encoding/binary"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

func TestPlanetSectorCreationDeterministicAndPersistent(t *testing.T) {
	var baseline []uint64
	for pass := 0; pass < 3; pass++ {
		k, ctx := keepertest.StructsKeeper(t)
		header := ctx.BlockHeader()
		header.AppHash = make([]byte, 32)
		seed := uint64(123456)
		if pass == 2 {
			seed++
		}
		binary.BigEndian.PutUint64(header.AppHash, seed)
		ctx = ctx.WithBlockHeader(header)
		seen := map[uint64]bool{}
		var sectors []uint64
		for i := 0; i < 120; i++ {
			cc := k.NewCurrentContext(ctx)
			planet := cc.NewPlanet("creator", "1-1")
			sector := planet.GetSector()
			require.True(t, types.IsPlanetSector(sector))
			seen[sector] = true
			cc.CommitAll()
			stored, found := k.GetPlanet(ctx, planet.ID())
			require.True(t, found)
			require.Equal(t, sector, stored.Sector)
			query, err := k.Planet(ctx, &types.QueryGetPlanetRequest{Id: planet.ID()})
			require.NoError(t, err)
			require.Equal(t, sector, query.Planet.Sector)
			sectors = append(sectors, sector)
		}
		require.Len(t, seen, 3, "same-block planets must not all use the same draw")
		if pass == 0 {
			baseline = sectors
		} else if pass == 1 {
			require.Equal(t, baseline, sectors, "validators must agree")
		} else {
			require.NotEqual(t, baseline, sectors, "block AppHash must participate in the draw")
		}
	}
}

func TestPlanetSectorBackfillPreservesEverythingElse(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	rows := []types.Planet{
		{Id: "2-1", Owner: "1-1", Name: "Legacy", MaxOre: 100, LocationListCount: 2, LocationListExtra: 3},
		{Id: "2-2", Status: types.PlanetStatus_complete},
		{Id: "2-3", Owner: "1-2", Sector: 999},
	}
	for _, row := range rows {
		k.SetPlanet(ctx, row)
	}
	count := k.GetPlanetCount(ctx)
	beforeEvents := len(ctx.EventManager().Events())
	k.BackfillPlanetSectors(ctx)
	require.Len(t, ctx.EventManager().Events(), beforeEvents+2)
	for _, event := range ctx.EventManager().ABCIEvents()[beforeEvents:] {
		parsed, err := sdk.ParseTypedEvent(event)
		require.NoError(t, err)
		planetEvent, ok := parsed.(*types.EventPlanet)
		require.True(t, ok)
		stored, found := k.GetPlanet(ctx, planetEvent.Planet.Id)
		require.True(t, found)
		require.True(t, proto.Equal(&stored, planetEvent.Planet), "migration must publish the assigned sector")
	}

	for _, row := range rows {
		stored, found := k.GetPlanet(ctx, row.Id)
		require.True(t, found)
		require.True(t, types.IsPlanetSector(stored.Sector))
		if row.Sector != 0 {
			require.Equal(t, row.Sector, stored.Sector)
		}
		stored.Sector = row.Sector
		require.Equal(t, row, stored)
	}
	require.Equal(t, count, k.GetPlanetCount(ctx))
	before := k.GetAllPlanet(ctx)
	header := ctx.BlockHeader()
	header.AppHash = []byte("different app hash for rerun")
	ctx = ctx.WithBlockHeader(header)
	k.BackfillPlanetSectors(ctx)
	require.Equal(t, before, k.GetAllPlanet(ctx))
	require.Len(t, ctx.EventManager().Events(), beforeEvents+2)
	// All statuses are assigned without needing their former owner to exist.
	for _, planet := range before {
		require.True(t, types.IsPlanetSector(planet.Sector), fmt.Sprint(planet))
	}
}
