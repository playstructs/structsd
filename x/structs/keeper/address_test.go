package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"

	"github.com/stretchr/testify/require"
)

func TestGetPlayerIndexFromAddress(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test non-existent address
	playerIndex := keeper.GetPlayerIndexFromAddress(ctx, "structs1qmhyqk")
	require.Equal(t, uint64(0), playerIndex)

	// Test setting and getting an address
	testAddress := "structs1qmhyqk"
	testPlayerIndex := uint64(42)
	keeper.SetPlayerIndexForAddress(ctx, testAddress, testPlayerIndex)

	playerIndex = keeper.GetPlayerIndexFromAddress(ctx, testAddress)
	require.Equal(t, testPlayerIndex, playerIndex)
}

func TestSetAndRevokePlayerIndexForAddress(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	testAddress := "structs1qmhyqk"
	testPlayerIndex := uint64(42)

	// Test setting player index
	keeper.SetPlayerIndexForAddress(ctx, testAddress, testPlayerIndex)
	playerIndex := keeper.GetPlayerIndexFromAddress(ctx, testAddress)
	require.Equal(t, testPlayerIndex, playerIndex)

	// Test revoking player index
	keeper.RevokePlayerIndexForAddress(ctx, testAddress, testPlayerIndex)
	playerIndex = keeper.GetPlayerIndexFromAddress(ctx, testAddress)
	require.Equal(t, uint64(0), playerIndex)
}

func TestGetAllAddressExport(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Add multiple addresses
	addresses := []string{
		"structs1qmhyqk",
		"structs2t23sqk",
		"structs32hhlqk",
	}

	for i, addr := range addresses {
		keeper.SetPlayerIndexForAddress(ctx, addr, uint64(i+1))
	}

	// Get all addresses
	addressRecords := keeper.GetAllAddressExport(ctx)
	require.Len(t, addressRecords, len(addresses))

	// Verify each address is present with correct player index
	for i, record := range addressRecords {
		require.Equal(t, addresses[i], record.Address)
		require.Equal(t, uint64(i+1), record.PlayerIndex)
	}
}

func TestAddressEmitActivity(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	testAddress := "structs1qmhyqk"

	// This test is mainly to ensure the function doesn't panic
	// The actual event emission would need to be verified through the event manager
	require.NotPanics(t, func() {
		keeper.AddressEmitActivity(ctx, testAddress)
	})
}
