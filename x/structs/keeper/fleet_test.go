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
		player := types.Player{
			Creator:        "structs" + strconv.Itoa(i),
			PrimaryAddress: "structs" + strconv.Itoa(i),
		}
		player = testAppendPlayer(keeper, ctx, player)
		fleet := types.Fleet{Owner: player.Id}
		items[i] = testAppendFleet(keeper, ctx, fleet)
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

/* TestFleetRemove is disabled because RemoveFleet no longer exists on Keeper.
func TestFleetRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNFleet(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveFleet(ctx, item.Id)
		_, found := keeper.GetFleet(ctx, item.Id)
		require.False(t, found)
	}
}
*/

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
	player = testAppendPlayer(keeper, ctx, player)

	// Create fleet
	fleet := testAppendFleet(keeper, ctx, types.Fleet{Owner: player.Id})

	// Test loading fleet data directly
	loadedFleet, found := keeper.GetFleet(ctx, fleet.Id)
	require.True(t, found)
	require.Equal(t, fleet.Id, loadedFleet.Id)
	require.Equal(t, player.Id, loadedFleet.Owner)

	// Test owner exists
	owner, ownerFound := keeper.GetPlayer(ctx, player.Id)
	require.True(t, ownerFound)
	require.Equal(t, player.Id, owner.Id)
}
