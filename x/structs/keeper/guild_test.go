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

func createNGuild(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Guild {
	items := make([]types.Guild, n)
	for i := range items {
		items[i].Id = keeper.AppendGuild(ctx, items[i])
	}
	return items
}

func TestGuildGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNGuild(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetGuild(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestGuildRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNGuild(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveGuild(ctx, item.Id)
		_, found := keeper.GetGuild(ctx, item.Id)
		require.False(t, found)
	}
}

func TestGuildGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNGuild(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllGuild(ctx)),
	)
}

func TestGuildCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNGuild(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetGuildCount(ctx))
}
