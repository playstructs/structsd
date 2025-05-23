package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	"structs/testutil/nullify"
	"structs/x/structs/keeper"
	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func createNFleet(keeper keeper.Keeper, ctx sdk.Context, n int) []types.Fleet {
	items := make([]types.Fleet, n)
	for i := range items {
		// Create a player for each fleet
		player := types.Player{
			Id:      "player" + string(rune(i)),
			Creator: "creator" + string(rune(i)),
		}
		playerCache, _ := keeper.GetPlayerCacheFromId(ctx, player.Id)
		items[i] = keeper.AppendFleet(ctx, &playerCache)
	}
	return items
}

func TestFleetGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNFleet(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetFleet(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestFleetRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNFleet(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveFleet(ctx, item.Id)
		_, found := keeper.GetFleet(ctx, item.Id)
		require.False(t, found)
	}
}

func TestFleetGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNFleet(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllFleet(ctx)),
	)
}

func TestFleetCache(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create test player
	player := types.Player{
		Id:      "test-player",
		Creator: "test-creator",
	}
	playerCache, _ := keeper.GetPlayerCacheFromId(ctx, player.Id)

	// Create fleet
	fleet := keeper.AppendFleet(ctx, &playerCache)

	// Test FleetCache
	cache, err := keeper.GetFleetCacheFromId(ctx, fleet.Id)
	require.NoError(t, err)
	require.Equal(t, fleet.Id, cache.GetFleetId())

	// Test loading fleet data
	loadedFleet := cache.GetFleet()
	require.Equal(t, fleet.Id, loadedFleet.Id)
	require.Equal(t, player.Id, loadedFleet.Owner)

	// Test owner loading
	owner := cache.GetOwner()
	require.NotNil(t, owner)
	require.Equal(t, player.Id, owner.GetPlayerId())
}

func TestFleetLocationManagement(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create test player
	player := types.Player{
		Id:      "test-player",
		Creator: "test-creator",
	}
	playerCache, _ := keeper.GetPlayerCacheFromId(ctx, player.Id)

	// Create fleet
	fleet := keeper.AppendFleet(ctx, &playerCache)
	cache, _ := keeper.GetFleetCacheFromId(ctx, fleet.Id)

	// Create test planet
	planet := types.Planet{
		Id: "test-planet",
	}
	planetCache := keeper.GetPlanetCacheFromId(ctx, planet.Id)

	// Test setting location
	cache.SetLocationToPlanet(&planetCache)
	require.Equal(t, planet.Id, cache.GetLocationId())
	require.Equal(t, types.ObjectType_planet, cache.GetLocationType())

	// Test location status
	require.True(t, cache.IsAway())
	require.False(t, cache.IsOnStation())

	// Test moving back to home planet
	homePlanet := keeper.GetPlanetCacheFromId(ctx, playerCache.GetPlanetId())
	cache.SetLocationToPlanet(&homePlanet)
	require.True(t, cache.IsOnStation())
	require.False(t, cache.IsAway())
}

func TestFleetStructManagement(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create test player
	player := types.Player{
		Id:      "test-player",
		Creator: "test-creator",
	}
	playerCache, _ := keeper.GetPlayerCacheFromId(ctx, player.Id)

	// Create fleet
	fleet := keeper.AppendFleet(ctx, &playerCache)
	cache, _ := keeper.GetFleetCacheFromId(ctx, fleet.Id)

	// Create test struct
	structType := types.StructType{
		Id:       1,
		Type:     types.CommandStruct,
		Category: types.ObjectType_fleet,
	}
	structure := types.Struct{
		Id:             "test-struct",
		Owner:          player.Id,
		Type:           structType.Id,
		OperatingAmbit: types.Ambit_land,
		Slot:           0,
	}

	// Test setting command struct
	cache.SetCommandStruct(structure)
	require.True(t, cache.HasCommandStruct())
	require.Equal(t, structure.Id, cache.GetCommandStructId())

	// Test clearing command struct
	cache.ClearCommandStruct()
	require.False(t, cache.HasCommandStruct())
}

func TestFleetSlotManagement(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create test player
	player := types.Player{
		Id:      "test-player",
		Creator: "test-creator",
	}
	playerCache, _ := keeper.GetPlayerCacheFromId(ctx, player.Id)

	// Create fleet
	fleet := keeper.AppendFleet(ctx, &playerCache)
	cache, _ := keeper.GetFleetCacheFromId(ctx, fleet.Id)

	// Create test struct
	structure := types.Struct{
		Id:             "test-struct",
		Owner:          player.Id,
		OperatingAmbit: types.Ambit_land,
		Slot:           0,
	}

	// Test setting slot
	err := cache.SetSlot(structure)
	require.NoError(t, err)
	require.Equal(t, structure.Id, cache.GetFleet().Land[0])

	// Test clearing slot
	cache.ClearSlot(types.Ambit_land, 0)
	require.Equal(t, "", cache.GetFleet().Land[0])
}
