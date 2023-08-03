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

func createNAddress(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Address {
	items := make([]types.Address, n)
	for i := range items {
		items[i].Id = keeper.AppendAddress(ctx, items[i])
	}
	return items
}

func TestAddressGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAddress(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetAddress(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestAddressRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAddress(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveAddress(ctx, item.Id)
		_, found := keeper.GetAddress(ctx, item.Id)
		require.False(t, found)
	}
}

func TestAddressGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAddress(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllAddress(ctx)),
	)
}

func TestAddressCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAddress(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetAddressCount(ctx))
}
