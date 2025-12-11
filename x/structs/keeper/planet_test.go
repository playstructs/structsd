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
		player := types.Player{
			Id:      "player" + string(rune(i)),
			Creator: "address" + string(rune(i)),
		}
		planetId := keeper.AppendPlanet(ctx, player)
		planet, _ := keeper.GetPlanet(ctx, planetId)
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
	player := types.Player{
		Id:      "player1",
		Creator: "address1",
	}
	_ = keeper.AppendPlanet(ctx, player)
	newCount := keeper.GetPlanetCount(ctx)
	require.Equal(t, initialCount+1, newCount)
}

func TestPlanetAttributes(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	player := types.Player{
		Id:      "player1",
		Creator: "address1",
	}
	planetId := keeper.AppendPlanet(ctx, player)

	// Test initial attributes
	attributes := keeper.GetPlanetAttributesByObject(ctx, planetId)
	require.Equal(t, uint64(types.PlanetaryShieldBase), attributes.PlanetaryShield)
	require.Equal(t, uint64(0), attributes.RepairNetworkQuantity)
	require.Equal(t, uint64(0), attributes.DefensiveCannonQuantity)

	// Test setting attributes
	attributeId := kpr.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetaryShield, planetId)
	keeper.SetPlanetAttribute(ctx, attributeId, 100)
	value := keeper.GetPlanetAttribute(ctx, attributeId)
	require.Equal(t, uint64(100), value)

	// Test increment
	newValue := keeper.SetPlanetAttributeIncrement(ctx, attributeId, 50)
	require.Equal(t, uint64(150), newValue)

	// Test decrement
	value, _ = keeper.SetPlanetAttributeDecrement(ctx, attributeId, 30)
	require.Equal(t, uint64(120), value)

	// Test clear
	keeper.ClearPlanetAttribute(ctx, attributeId)
	value = keeper.GetPlanetAttribute(ctx, attributeId)
	require.Equal(t, uint64(0), value)
}

func TestPlanetCache(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	player := types.Player{
		Id:      "player1",
		Creator: "address1",
	}
	planetId := keeper.AppendPlanet(ctx, player)

	// Test cache creation and loading
	cache := keeper.GetPlanetCacheFromId(ctx, planetId)
	require.Equal(t, planetId, cache.GetPlanetId())

	// Test cache operations
	cache.LoadPlanet()
	planet := cache.GetPlanet()
	require.Equal(t, planetId, planet.Id)

	// Test attribute operations through cache
	cache.LoadPlanetaryShield()
	initialShield := cache.GetPlanetaryShield()
	require.Equal(t, uint64(types.PlanetaryShieldBase), initialShield)

	cache.PlanetaryShieldIncrement(50)
	require.Equal(t, uint64(types.PlanetaryShieldBase+50), cache.GetPlanetaryShield())

	cache.PlanetaryShieldDecrement(20)
	require.Equal(t, uint64(types.PlanetaryShieldBase+30), cache.GetPlanetaryShield())

	// Test commit
	cache.Commit()
	attributes := keeper.GetPlanetAttributesByObject(ctx, planetId)
	require.Equal(t, uint64(types.PlanetaryShieldBase+30), attributes.PlanetaryShield)
}

func TestPlanetStatus(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	player := types.Player{
		Id:      "player1",
		Creator: "address1",
	}
	planetId := keeper.AppendPlanet(ctx, player)
	cache := keeper.GetPlanetCacheFromId(ctx, planetId)

	// Test initial status
	require.True(t, cache.IsActive())

	// Test status change
	cache.SetStatus(types.PlanetStatus_complete)
	require.True(t, cache.IsComplete())
	require.False(t, cache.IsActive())

	// Test commit
	cache.Commit()
	planet, _ := keeper.GetPlanet(ctx, planetId)
	require.Equal(t, types.PlanetStatus_complete, planet.Status)
}
