package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	"structs/testutil/nullify"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func createNSubstation(t *testing.T, keeper keeperlib.Keeper, ctx sdk.Context, n int) []types.Substation {
	items := make([]types.Substation, n)
	for i := range items {
		// Create test player first
		player := types.Player{
			Creator:        "creator" + string(rune(i)),
			PrimaryAddress: "creator" + string(rune(i)),
		}
		player = testAppendPlayer(keeper, ctx, player)

		// Set up source capacity for the allocation
		sourceId := "source" + string(rune(i))
		keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceId), uint64(200))

		// Create test allocation
		allocation := types.Allocation{
			SourceObjectId: sourceId,
			DestinationId:  "",
			Type:           types.AllocationType_static,
		}
		// Create the allocation first
		createdAllocation, err := testAppendAllocation(keeper, ctx, allocation, 100)
		require.NoError(t, err)

		// Append substation and handle returned values
		substation, _, err := testAppendSubstation(keeper, ctx, createdAllocation, player)
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
		keeper.ClearSubstation(ctx, item.Id)
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
	initialCount := keeper.GetSubstationCount(ctx)
	items := createNSubstation(t, keeper, ctx, 10)
	require.Equal(t, initialCount+uint64(len(items)), keeper.GetSubstationCount(ctx))
}

func TestSubstationPlayerConnection(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create test substation
	// First create a player for the substation owner
	ownerPlayer := types.Player{
		Creator:        "creator1",
		PrimaryAddress: "creator1",
	}
	ownerPlayer = testAppendPlayer(keeper, ctx, ownerPlayer)

	// Set up source capacity for the allocation
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, "source1"), uint64(200))

	allocation := types.Allocation{
		SourceObjectId: "source1",
		DestinationId:  "",
		Type:           types.AllocationType_static,
	}
	// Create the allocation first
	createdAllocation, err := testAppendAllocation(keeper, ctx, allocation, 100)
	require.NoError(t, err)

	player := types.Player{
		Id:      ownerPlayer.Id,
		Creator: ownerPlayer.Creator,
	}
	substation, _, err := testAppendSubstation(keeper, ctx, createdAllocation, player)
	require.NoError(t, err)

	// Create test player
	testPlayer := types.Player{
		Id:      "test-player",
		Creator: "test-creator",
	}
	testPlayer = testAppendPlayer(keeper, ctx, testPlayer)

	// Test connecting player by setting SubstationId directly
	testPlayer.SubstationId = substation.Id
	keeper.SetPlayer(ctx, testPlayer)

	// Verify connection
	updatedPlayer, found := keeper.GetPlayer(ctx, testPlayer.Id)
	require.True(t, found)
	require.Equal(t, substation.Id, updatedPlayer.SubstationId)

	// Test disconnecting player
	updatedPlayer.SubstationId = ""
	keeper.SetPlayer(ctx, updatedPlayer)

	disconnectedPlayer, found := keeper.GetPlayer(ctx, testPlayer.Id)
	require.True(t, found)
	require.Equal(t, "", disconnectedPlayer.SubstationId)
}
