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

func createNAllocation(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Allocation {
	items := make([]types.Allocation, n)
	for i := range items {
		items[i].Id = keeper.AppendAllocation(ctx, items[i])
	}
	return items
}

func TestAllocationGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAllocation(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetAllocation(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestAllocationRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAllocation(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveAllocation(ctx, item.Id)
		_, found := keeper.GetAllocation(ctx, item.Id)
		require.False(t, found)
	}
}

func TestAllocationGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAllocation(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllAllocation(ctx)),
	)
}

func TestAllocationCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAllocation(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetAllocationCount(ctx))
}
