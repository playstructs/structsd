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

func createNSubstation(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Substation {
	items := make([]types.Substation, n)
	for i := range items {
		items[i].Id = keeper.AppendSubstation(ctx, items[i])
	}
	return items
}

func TestSubstationGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNSubstation(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetSubstation(ctx, item.Id, true)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestSubstationRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNSubstation(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveSubstation(ctx, item.Id)
		_, found := keeper.GetSubstation(ctx, item.Id, true)
		require.False(t, found)
	}
}

func TestSubstationGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNSubstation(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllSubstation(ctx, true)),
	)
}

func TestSubstationCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNSubstation(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetSubstationCount(ctx))
}
