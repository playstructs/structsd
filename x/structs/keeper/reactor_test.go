package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	"structs/testutil/nullify"
	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

func createNReactor(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Reactor {
	items := make([]types.Reactor, n)
	for i := range items {
		items[i].Id = keeper.AppendReactor(ctx, items[i])
	}
	return items
}

func TestReactorGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNReactor(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetReactor(ctx, item.Id, false)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestReactorRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNReactor(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveReactor(ctx, item.Id)
		_, found := keeper.GetReactor(ctx, item.Id, false)
		require.False(t, found)
	}
}

func TestReactorGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNReactor(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllReactor(ctx, false)),
	)
}

func TestReactorCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNReactor(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetReactorCount(ctx))
}
