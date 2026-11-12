package app_test

import (
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/stretchr/testify/require"

	v23 "structs/app/upgrades/v0_23_0"
	"structs/x/structs/types"
)

// Exercise the registered production handler through x/upgrade with the current
// module version map. A new release must not replay the old economy migration.
func TestV23RegisteredUpgradePreservesCurrentState(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	header := ctx.HeaderInfo()
	header.Height = ctx.BlockHeight()
	ctx = ctx.WithHeaderInfo(header)
	require.True(t, bApp.UpgradeKeeper.HasHandler(v23.UpgradeName))
	// The genesis helper's runtime version map omits manually wired IBC
	// modules. Seed the completed v0.22.0 migration map for this upgrade test.
	beforeVM := bApp.ModuleManager.GetVersionMap()
	require.NoError(t, bApp.UpgradeKeeper.SetModuleVersionMap(ctx, beforeVM))
	for _, planet := range []types.Planet{
		{Id: "2-991", Name: "Legacy", MaxOre: 123},
		{Id: "2-992", Status: types.PlanetStatus_complete},
		{Id: "2-993", Sector: 44},
	} {
		bApp.StructsKeeper.SetPlanet(ctx, planet)
	}
	before, err := bApp.ModuleManager.ExportGenesisForModules(ctx, bApp.AppCodec(), []string{types.ModuleName})
	require.NoError(t, err)

	plan := upgradetypes.Plan{Name: v23.UpgradeName, Height: ctx.BlockHeight()}
	require.NoError(t, bApp.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, bApp.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	afterVM, err := bApp.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.Equal(t, beforeVM, afterVM)
	after, err := bApp.ModuleManager.ExportGenesisForModules(ctx, bApp.AppCodec(), []string{types.ModuleName})
	require.NoError(t, err)
	var expected, actual types.GenesisState
	bApp.AppCodec().MustUnmarshalJSON(before[types.ModuleName], &expected)
	bApp.AppCodec().MustUnmarshalJSON(after[types.ModuleName], &actual)
	require.Len(t, actual.PlanetList, len(expected.PlanetList))
	for i := range expected.PlanetList {
		require.True(t, types.IsPlanetSector(actual.PlanetList[i].Sector))
		if expected.PlanetList[i].Sector != 0 {
			require.Equal(t, expected.PlanetList[i].Sector, actual.PlanetList[i].Sector)
		}
		expected.PlanetList[i].Sector = actual.PlanetList[i].Sector
	}
	require.JSONEq(t, string(bApp.AppCodec().MustMarshalJSON(&expected)), string(after[types.ModuleName]))
	doneHeight, err := bApp.UpgradeKeeper.GetDoneHeight(ctx, v23.UpgradeName)
	require.NoError(t, err)
	require.Equal(t, plan.Height, doneHeight)
	require.Empty(t, v23.NewUpgrade().StoreUpgrades)
}

func TestIBCV2RouterRejectsUnregisteredApplications(t *testing.T) {
	bApp, _ := setupJailGateApp(t)
	router := bApp.IBCKeeper.ChannelKeeperV2.Router
	require.NotNil(t, router)
	for _, port := range []string{"transfer", "icahost", "icacontroller", "unknown"} {
		require.False(t, router.HasRoute(port))
		// Upstream v10.7.0 uses an explicit panic for missing routes. BaseApp
		// recovers it on message delivery; it is not a nil router dereference.
		require.PanicsWithValue(t, "no route for "+port, func() { router.Route(port) })
	}
}
