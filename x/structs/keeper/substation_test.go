package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	"structs/testutil/nullify"
	"structs/x/structs/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func createNSubstation(t *testing.T, keeper keeper.Keeper, ctx sdk.Context, n int) []types.Substation {
	items := make([]types.Substation, n)
	for i := range items {
		// Create test allocation and player
		allocation := types.Allocation{
			SourceObjectId: "source" + string(rune(i)),
			DestinationId:  "",
			Type:           types.AllocationType_static,
		}
		player := types.Player{
			Id:      "player" + string(rune(i)),
			Creator: "creator" + string(rune(i)),
		}

		// Append substation and handle returned values
		substation, _, err := keeper.AppendSubstation(ctx, allocation, player)
		require.NoError(t, err)
		items[i] = substation
	}
	return items
}

func TestSubstationGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNSubstation(t, keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetSubstation(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestSubstationRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNSubstation(t, keeper, ctx, 10)
	for _, item := range items {
		// Test removal with empty migration ID
		keeper.RemoveSubstation(ctx, item.Id, "")
		_, found := keeper.GetSubstation(ctx, item.Id)
		require.False(t, found)
	}
}

func TestSubstationGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNSubstation(t, keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllSubstation(ctx)),
	)
}

func TestSubstationCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNSubstation(t, keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetSubstationCount(ctx))
}

func TestSubstationPlayerConnection(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create test substation
	allocation := types.Allocation{
		SourceObjectId: "source1",
		DestinationId:  "",
		Type:           types.AllocationType_static,
	}
	player := types.Player{
		Id:      "player1",
		Creator: "creator1",
	}
	substation, _, err := keeper.AppendSubstation(ctx, allocation, player)
	require.NoError(t, err)

	// Create test player
	testPlayer := types.Player{
		Id:      "test-player",
		Creator: "test-creator",
	}
	testPlayer = keeper.AppendPlayer(ctx, testPlayer)

	// Test connecting player
	updatedPlayer, err := keeper.SubstationConnectPlayer(ctx, substation, testPlayer)
	require.NoError(t, err)
	require.Equal(t, substation.Id, updatedPlayer.SubstationId)

	// Verify connection count
	connectionCount := keeper.GetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, substation.Id))
	require.Equal(t, uint64(1), connectionCount)

	// Test disconnecting player
	updatedPlayer, err = keeper.SubstationDisconnectPlayer(ctx, updatedPlayer)
	require.NoError(t, err)
	require.Equal(t, "", updatedPlayer.SubstationId)

	// Verify connection count is decremented
	connectionCount = keeper.GetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, substation.Id))
	require.Equal(t, uint64(0), connectionCount)
}
