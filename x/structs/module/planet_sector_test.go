package structs_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	structs "structs/x/structs/module"
	"structs/x/structs/types"
)

func TestPlanetSectorGenesisRoundTrip(t *testing.T) {
	gs := types.DefaultGenesis()
	gs.PlanetList = []types.Planet{{Id: "2-1"}, {Id: "2-2", Sector: 127}, {Id: "2-3", Sector: 999}, {Id: "2-4", Sector: 44, Status: types.PlanetStatus_complete}}
	require.NoError(t, gs.Validate())
	k, ctx := keepertest.StructsKeeper(t)
	structs.InitGenesis(ctx, k, *gs)
	exported := structs.ExportGenesis(ctx, k)
	require.True(t, types.IsPlanetSector(exported.PlanetList[0].Sector))
	for i := 1; i < len(gs.PlanetList); i++ {
		require.Equal(t, gs.PlanetList[i].Sector, exported.PlanetList[i].Sector)
	}
	// Exercise protobuf wire preservation, not just copying Go records.
	wire, err := exported.Marshal()
	require.NoError(t, err)
	var decoded types.GenesisState
	require.NoError(t, decoded.Unmarshal(wire))
	k2, ctx2 := keepertest.StructsKeeper(t)
	header := ctx2.BlockHeader()
	header.AppHash = []byte("different restart apphash")
	ctx2 = ctx2.WithBlockHeader(header)
	structs.InitGenesis(ctx2, k2, decoded)
	require.Equal(t, k.GetAllPlanet(ctx), k2.GetAllPlanet(ctx2))
	gs.PlanetList[0].Sector = 45
	require.Error(t, gs.Validate())
}
