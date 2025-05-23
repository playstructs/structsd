package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	"structs/testutil/nullify"
	"structs/x/structs/types"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
)

func TestFleetQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test nil request
	_, err := keeper.Fleet(ctx, nil)
	require.Error(t, err)

	// Test non-existent fleet
	req := &types.QueryGetFleetRequest{
		Id: "non-existent",
	}
	_, err = keeper.Fleet(ctx, req)
	require.Error(t, err)

	// Test existing fleet
	player := types.Player{
		Id:      "test-player",
		Creator: "test-creator",
	}
	playerCache, _ := keeper.GetPlayerCacheFromId(ctx, player.Id)
	fleet := keeper.AppendFleet(ctx, &playerCache)

	req = &types.QueryGetFleetRequest{
		Id: fleet.Id,
	}
	resp, err := keeper.Fleet(ctx, req)
	require.NoError(t, err)
	require.Equal(t,
		nullify.Fill(&fleet),
		nullify.Fill(&resp.Fleet),
	)
}

func TestFleetAllQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test nil request
	_, err := keeper.FleetAll(ctx, nil)
	require.Error(t, err)

	// Create multiple fleets
	player := types.Player{
		Id:      "test-player",
		Creator: "test-creator",
	}
	playerCache, _ := keeper.GetPlayerCacheFromId(ctx, player.Id)
	fleets := make([]types.Fleet, 5)
	for i := range fleets {
		fleets[i] = keeper.AppendFleet(ctx, &playerCache)
	}

	// Test pagination
	req := &types.QueryAllFleetRequest{
		Pagination: &query.PageRequest{
			Limit: 2,
		},
	}
	resp, err := keeper.FleetAll(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Fleet, 2)
	require.NotNil(t, resp.Pagination)

	// Test without pagination
	req = &types.QueryAllFleetRequest{}
	resp, err = keeper.FleetAll(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Fleet, len(fleets))
	require.Nil(t, resp.Pagination)

	// Verify all fleets are present
	for _, fleet := range fleets {
		found := false
		for _, respFleet := range resp.Fleet {
			if fleet.Id == respFleet.Id {
				found = true
				require.Equal(t,
					nullify.Fill(&fleet),
					nullify.Fill(&respFleet),
				)
				break
			}
		}
		require.True(t, found)
	}
}

func TestFleetByIndexQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test nil request
	_, err := keeper.FleetByIndex(ctx, nil)
	require.Error(t, err)

	// Create a fleet
	player := types.Player{
		Id:      "test-player",
		Creator: "test-creator",
	}
	playerCache, _ := keeper.GetPlayerCacheFromId(ctx, player.Id)
	fleet := keeper.AppendFleet(ctx, &playerCache)

	// Extract index from fleet ID (assuming format "fleet-{index}")
	fleetIndex := uint64(1) // This should match the index used in AppendFleet

	// Test existing fleet by index
	req := &types.QueryGetFleetByIndexRequest{
		Index: fleetIndex,
	}
	resp, err := keeper.FleetByIndex(ctx, req)
	require.NoError(t, err)
	require.Equal(t,
		nullify.Fill(&fleet),
		nullify.Fill(&resp.Fleet),
	)

	// Test non-existent fleet index
	req = &types.QueryGetFleetByIndexRequest{
		Index: 999, // Use a high index that shouldn't exist
	}
	_, err = keeper.FleetByIndex(ctx, req)
	require.Error(t, err)
}
