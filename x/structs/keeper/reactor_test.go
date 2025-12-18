package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

func createNReactor(keeper keeper.Keeper, ctx sdk.Context, n int) []types.Reactor {
	items := make([]types.Reactor, n)
	for i := range items {
		items[i] = types.Reactor{
			RawAddress: []byte("validator" + string(rune(i))),
		}
		items[i] = keeper.AppendReactor(ctx, items[i])
	}
	return items
}

func TestReactorGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNReactor(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetReactor(ctx, item.Id)
		require.True(t, found)
		require.Equal(t, item.Id, got.Id)
		require.Equal(t, item.Validator, got.Validator)
		require.Equal(t, item.GuildId, got.GuildId)
		require.Equal(t, item.RawAddress, got.RawAddress)
		// DefaultCommission is a LegacyDec, check if it's zero
		require.True(t, got.DefaultCommission.IsZero() || item.DefaultCommission.Equal(got.DefaultCommission))
	}
}

func TestReactorRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNReactor(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveReactor(ctx, item.Id)
		_, found := keeper.GetReactor(ctx, item.Id)
		require.False(t, found)
	}
}

func TestReactorGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNReactor(keeper, ctx, 10)
	allReactors := keeper.GetAllReactor(ctx)
	require.Len(t, allReactors, len(items))
	// Verify all created reactors are in the result
	for _, item := range items {
		found := false
		for _, reactor := range allReactors {
			if reactor.Id == item.Id {
				found = true
				break
			}
		}
		require.True(t, found, "Reactor %s should be in GetAllReactor result", item.Id)
	}
}

func TestReactorCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	initialCount := keeper.GetReactorCount(ctx)
	items := createNReactor(keeper, ctx, 10)
	expectedCount := initialCount + uint64(len(items))
	actualCount := keeper.GetReactorCount(ctx)
	require.Equal(t, expectedCount, actualCount)
}

func TestReactorGetByBytes(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNReactor(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetReactorByBytes(ctx, []byte(item.Id))
		require.True(t, found)
		require.Equal(t, item.Id, got.Id)
		require.Equal(t, item.RawAddress, got.RawAddress)
	}

	// Test with nil bytes
	_, found := keeper.GetReactorByBytes(ctx, nil)
	require.False(t, found)
}

func TestReactorValidatorOperations(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create a test reactor
	reactor := types.Reactor{
		RawAddress: []byte("test-validator"),
	}
	reactor = keeper.AppendReactor(ctx, reactor)

	// Test getting reactor bytes from validator
	reactorBytes, found := keeper.GetReactorBytesFromValidator(ctx, []byte("test-validator"))
	require.True(t, found)
	require.Equal(t, reactor.Id, string(reactorBytes))

	// Test getting reactor bytes from non-existent validator
	_, found = keeper.GetReactorBytesFromValidator(ctx, []byte("non-existent"))
	require.False(t, found)

	// Test getting reactor bytes with nil validator address
	_, found = keeper.GetReactorBytesFromValidator(ctx, nil)
	require.False(t, found)
}
