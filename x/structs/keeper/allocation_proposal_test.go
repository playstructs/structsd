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

func createNAllocationProposal(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.AllocationProposal {
	items := make([]types.AllocationProposal, n)
	for i := range items {
		items[i].Id = keeper.AppendAllocationProposal(ctx, items[i])
	}
	return items
}

func TestAllocationProposalGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAllocationProposal(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetAllocationProposal(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestAllocationProposalRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAllocationProposal(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveAllocationProposal(ctx, item.Id)
		_, found := keeper.GetAllocationProposal(ctx, item.Id)
		require.False(t, found)
	}
}

func TestAllocationProposalGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAllocationProposal(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllAllocationProposal(ctx)),
	)
}

func TestAllocationProposalCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAllocationProposal(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetAllocationProposalCount(ctx))
}
