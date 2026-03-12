package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	kpr "structs/x/structs/keeper"
	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func createNPlanet(keeper kpr.Keeper, ctx sdk.Context, n int) []types.Planet {
	items := make([]types.Planet, n)
	for i := range items {
		planet := testAppendPlanet(keeper, ctx, types.Planet{
			Creator: "address" + string(rune(i)),
			Owner:   "player" + string(rune(i)),
		})
		items[i] = planet
	}
	return items
}

func TestPlanetCRUD(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	planets := createNPlanet(keeper, ctx, 5)

	// Test GetPlanet
	for _, planet := range planets {
		got, found := keeper.GetPlanet(ctx, planet.Id)
		require.True(t, found)
		require.Equal(t, planet, got)
	}

	// Test GetAllPlanet
	allPlanets := keeper.GetAllPlanet(ctx)
	require.Len(t, allPlanets, 5)

	// Test RemovePlanet
	for _, planet := range planets {
		keeper.RemovePlanet(ctx, planet.Id)
		_, found := keeper.GetPlanet(ctx, planet.Id)
		require.False(t, found)
	}
}

func TestPlanetCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	initialCount := keeper.GetPlanetCount(ctx)
	require.Equal(t, types.KeeperStartValue, initialCount)

	// Create a planet and check count
	_ = testAppendPlanet(keeper, ctx, types.Planet{Creator: "address1", Owner: "player1"})
	newCount := keeper.GetPlanetCount(ctx)
	require.Equal(t, initialCount+1, newCount)
}

func TestPlanetAttributes(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	planet := testAppendPlanet(keeper, ctx, types.Planet{Creator: "address1", Owner: "player1"})

	// Test initial attributes
	attributes := keeper.GetPlanetAttributesByObject(ctx, planet.Id)
	require.Equal(t, uint64(types.PlanetaryShieldBase), attributes.PlanetaryShield)
	require.Equal(t, uint64(0), attributes.RepairNetworkQuantity)
	require.Equal(t, uint64(0), attributes.DefensiveCannonQuantity)

	// Test setting attributes
	attributeId := kpr.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetaryShield, planet.Id)
	keeper.SetPlanetAttribute(ctx, attributeId, 100)
	value := keeper.GetPlanetAttribute(ctx, attributeId)
	require.Equal(t, uint64(100), value)

	// Test increment (via CurrentContext)
	cc := keeper.NewCurrentContext(ctx)
	newValue := cc.SetPlanetAttributeIncrement(attributeId, 50)
	require.Equal(t, uint64(150), newValue)

	// Test decrement
	value = cc.SetPlanetAttributeDecrement(attributeId, 30)
	require.Equal(t, uint64(120), value)

	// Test clear
	keeper.ClearPlanetAttribute(ctx, attributeId)
	value = keeper.GetPlanetAttribute(ctx, attributeId)
	require.Equal(t, uint64(0), value)
}

func TestPlanetCache(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	planet := testAppendPlanet(keeper, ctx, types.Planet{Creator: "address1", Owner: "player1"})

	// Test cache creation and loading via CurrentContext
	cc := keeper.NewCurrentContext(ctx)
	cache := cc.GetPlanet(planet.Id)
	require.Equal(t, planet.Id, cache.GetPlanetId())

	// Test cache operations
	cache.LoadPlanet()
	loadedPlanet := cache.GetPlanet()
	require.Equal(t, planet.Id, loadedPlanet.Id)

	// Test attribute operations through cache
	initialShield := cache.GetPlanetaryShield()
	require.Equal(t, uint64(types.PlanetaryShieldBase), initialShield)

	cache.PlanetaryShieldIncrement(50)
	require.Equal(t, uint64(types.PlanetaryShieldBase+50), cache.GetPlanetaryShield())

	cache.PlanetaryShieldDecrement(20)
	require.Equal(t, uint64(types.PlanetaryShieldBase+30), cache.GetPlanetaryShield())

	// Test commit
	cache.Commit()
	attributes := keeper.GetPlanetAttributesByObject(ctx, planet.Id)
	require.Equal(t, uint64(types.PlanetaryShieldBase+30), attributes.PlanetaryShield)
}

func TestPlanetStatus(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	planet := testAppendPlanet(keeper, ctx, types.Planet{Creator: "address1", Owner: "player1"})

	// Test cache creation via CurrentContext
	cc := keeper.NewCurrentContext(ctx)
	cache := cc.GetPlanet(planet.Id)
	cache.LoadPlanet()

	// Test initial status
	require.True(t, cache.IsActive())

	// Test status change
	cache.SetStatus(types.PlanetStatus_complete)
	require.True(t, cache.IsComplete())
	require.False(t, cache.IsActive())

	// Test commit
	cache.Commit()
	updatedPlanet, _ := keeper.GetPlanet(ctx, planet.Id)
	require.Equal(t, types.PlanetStatus_complete, updatedPlanet.Status)
}
