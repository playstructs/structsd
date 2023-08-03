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

func createNPlayer(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Player {
	items := make([]types.Player, n)
	for i := range items {
		items[i].Id = keeper.AppendPlayer(ctx, items[i])
	}
	return items
}

func TestPlayerGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNPlayer(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetPlayer(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestPlayerRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNPlayer(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemovePlayer(ctx, item.Id)
		_, found := keeper.GetPlayer(ctx, item.Id)
		require.False(t, found)
	}
}

func TestPlayerGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNPlayer(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllPlayer(ctx)),
	)
}

func TestPlayerCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNPlayer(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetPlayerCount(ctx))
}
