package keeper_test

import (
	"strconv"
	"testing"

	keepertest "structs/testutil/keeper"
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
			Creator:        "structs" + strconv.Itoa(i),
			PrimaryAddress: "structs" + strconv.Itoa(i),
		}
		player = keeper.AppendPlayer(ctx, player)
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
		require.Equal(t, item.Id, got.Id)
		require.Equal(t, item.Owner, got.Owner)
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
	allFleets := keeper.GetAllFleet(ctx)
	require.Len(t, allFleets, len(items))
	// Verify all created fleets are in the result
	for _, item := range items {
		found := false
		for _, fleet := range allFleets {
			if fleet.Id == item.Id {
				found = true
				break
			}
		}
		require.True(t, found, "Fleet %s should be in GetAllFleet result", item.Id)
	}
}

func TestFleetCache(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create test player
	player := types.Player{
		Creator:        "test-creator",
		PrimaryAddress: "test-creator",
	}
	player = keeper.AppendPlayer(ctx, player)
	playerCache, err := keeper.GetPlayerCacheFromId(ctx, player.Id)
	require.NoError(t, err)

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
