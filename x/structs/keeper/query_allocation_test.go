package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
)

func TestQueryAllocation(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	sourceId := "source1"
	destId := "dest1"
	alloc := createTestAllocation(sourceId, destId, types.AllocationType_static)

	resp, err := keeper.Allocation(wctx, &types.QueryGetAllocationRequest{Id: alloc.Id})
	require.NoError(t, err)
	require.Equal(t, alloc.Id, resp.Allocation.Id)
	require.NotNil(t, resp.GridAttributes)

	_, err = keeper.Allocation(wctx, &types.QueryGetAllocationRequest{Id: "nonexistent"})
	require.Error(t, err)

	_, err = keeper.Allocation(wctx, nil)
	require.Error(t, err)
}

func TestQueryAllocationAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	sourceId := "source2"
	for i := 0; i < 5; i++ {
		destId := "dest" + string(rune(i))
		createTestAllocation(sourceId, destId, types.AllocationType_static)
	}

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllAllocationRequest {
		return &types.QueryAllAllocationRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}

	resp, err := keeper.AllocationAll(wctx, request(nil, 0, 3, false))
	require.NoError(t, err)
	require.LessOrEqual(t, len(resp.Allocation), 3)

	resp, err = keeper.AllocationAll(wctx, request(nil, 0, 0, true))
	require.NoError(t, err)
	require.GreaterOrEqual(t, int(resp.Pagination.Total), 5)

	_, err = keeper.AllocationAll(wctx, nil)
	require.Error(t, err)
}

func TestQueryAllocationAllBySource(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	sourceId := "source3"
	for i := 0; i < 3; i++ {
		destId := "dest" + string(rune(i))
		createTestAllocation(sourceId, destId, types.AllocationType_static)
	}

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllAllocationBySourceRequest {
		return &types.QueryAllAllocationBySourceRequest{
			SourceId: sourceId,
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}

	resp, err := keeper.AllocationAllBySource(wctx, request(nil, 0, 2, false))
	require.NoError(t, err)
	require.LessOrEqual(t, len(resp.Allocation), 2)

	_, err = keeper.AllocationAllBySource(wctx, nil)
	require.Error(t, err)
}

func TestQueryAllocationAllByDestination(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	destId := "destX"
	for i := 0; i < 2; i++ {
		sourceId := "sourceX" + string(rune(i))
		createTestAllocation(sourceId, destId, types.AllocationType_static)
	}

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllAllocationByDestinationRequest {
		return &types.QueryAllAllocationByDestinationRequest{
			DestinationId: destId,
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}

	resp, err := keeper.AllocationAllByDestination(wctx, request(nil, 0, 1, false))
	require.NoError(t, err)
	require.LessOrEqual(t, len(resp.Allocation), 1)

	_, err = keeper.AllocationAllByDestination(wctx, nil)
	require.Error(t, err)
}
