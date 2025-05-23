package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

// GetObjectID returns a formatted object ID string
func GetObjectID(objectType types.ObjectType, index uint64) string {
	return string(objectType) + "-" + string(rune(index))
}

func TestAddressQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test nil request
	_, err := keeper.Address(ctx, nil)
	require.Error(t, err)

	// Test non-existent address
	req := &types.QueryGetAddressRequest{
		Address: "structs1qmhyqk",
	}
	resp, err := keeper.Address(ctx, req)
	require.NoError(t, err)
	require.Equal(t, req.Address, resp.Address)
	require.Equal(t, uint64(0), resp.Permissions)

	// Test existing address
	testAddress := "structs1qmhyqk"
	testPlayerIndex := uint64(42)
	keeper.SetPlayerIndexForAddress(ctx, testAddress, testPlayerIndex)

	req = &types.QueryGetAddressRequest{
		Address: testAddress,
	}
	resp, err = keeper.Address(ctx, req)
	require.NoError(t, err)
	require.Equal(t, testAddress, resp.Address)
	require.Equal(t, GetObjectID(types.ObjectType_player, testPlayerIndex), resp.PlayerId)
}

func TestAddressAllQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test nil request
	_, err := keeper.AddressAll(ctx, nil)
	require.Error(t, err)

	// Add multiple addresses
	addresses := []string{
		"structs1qmhyqk",
		"structs2t23sqk",
		"structs32hhlqk",
	}

	for _, addr := range addresses {
		keeper.SetPlayerIndexForAddress(ctx, addr, uint64(1))
	}

	// Test pagination
	req := &types.QueryAllAddressRequest{
		Pagination: &query.PageRequest{
			Limit: 2,
		},
	}
	resp, err := keeper.AddressAll(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Address, 2)
	require.NotNil(t, resp.Pagination)

	// Test without pagination
	req = &types.QueryAllAddressRequest{}
	resp, err = keeper.AddressAll(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Address, len(addresses))
	require.Nil(t, resp.Pagination)
}

func TestAddressAllByPlayerQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test nil request
	_, err := keeper.AddressAllByPlayer(ctx, nil)
	require.Error(t, err)

	// Add addresses for different players
	player1Addresses := []string{
		"structs1qmhyqk",
		"structs2t23sqk",
	}
	player2Addresses := []string{
		"structs32hhlqk",
		"structs4k9lqk",
	}

	// Set addresses for player 1
	for _, addr := range player1Addresses {
		keeper.SetPlayerIndexForAddress(ctx, addr, uint64(1))
	}

	// Set addresses for player 2
	for _, addr := range player2Addresses {
		keeper.SetPlayerIndexForAddress(ctx, addr, uint64(2))
	}

	// Test query for player 1
	player1Id := GetObjectID(types.ObjectType_player, 1)
	req := &types.QueryAllAddressByPlayerRequest{
		PlayerId: player1Id,
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	}
	resp, err := keeper.AddressAllByPlayer(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Address, len(player1Addresses))
	for _, addr := range resp.Address {
		require.Equal(t, player1Id, addr.PlayerId)
	}

	// Test query for player 2
	player2Id := GetObjectID(types.ObjectType_player, 2)
	req = &types.QueryAllAddressByPlayerRequest{
		PlayerId: player2Id,
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	}
	resp, err = keeper.AddressAllByPlayer(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Address, len(player2Addresses))
	for _, addr := range resp.Address {
		require.Equal(t, player2Id, addr.PlayerId)
	}

	// Test query for non-existent player
	req = &types.QueryAllAddressByPlayerRequest{
		PlayerId: "player-999",
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	}
	resp, err = keeper.AddressAllByPlayer(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Address, 0)
}
