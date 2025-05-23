package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "structs/testutil/keeper"
	"structs/testutil/nullify"
	"structs/x/structs/types"
)

func TestInfusionQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test infusions
	destinationId := "test-destination"
	infusions := createNInfusion(keeper, ctx, 2, destinationId)

	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetInfusionRequest
		response *types.QueryGetInfusionResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetInfusionRequest{
				DestinationId: destinationId,
				Address:       infusions[0].Address,
			},
			response: &types.QueryGetInfusionResponse{Infusion: infusions[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetInfusionRequest{
				DestinationId: destinationId,
				Address:       infusions[1].Address,
			},
			response: &types.QueryGetInfusionResponse{Infusion: infusions[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetInfusionRequest{
				DestinationId: destinationId,
				Address:       "non-existent",
			},
			err: types.ErrObjectNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Infusion(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(tc.response),
					nullify.Fill(response),
				)
			}
		})
	}
}

func TestInfusionAllQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test infusions
	destinationId := "test-destination"
	infusions := createNInfusion(keeper, ctx, 5, destinationId)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllInfusionRequest {
		return &types.QueryAllInfusionRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}

	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(infusions); i += step {
			resp, err := keeper.InfusionAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Infusion), step)
			require.Subset(t,
				nullify.Fill(infusions),
				nullify.Fill(resp.Infusion),
			)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(infusions); i += step {
			resp, err := keeper.InfusionAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Infusion), step)
			require.Subset(t,
				nullify.Fill(infusions),
				nullify.Fill(resp.Infusion),
			)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.InfusionAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(infusions), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(infusions),
			nullify.Fill(resp.Infusion),
		)
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.InfusionAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func TestInfusionAllByDestinationQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test infusions for different destinations
	destination1 := "dest1"
	destination2 := "dest2"
	infusions1 := createNInfusion(keeper, ctx, 3, destination1)
	infusions2 := createNInfusion(keeper, ctx, 2, destination2)

	request := func(destId string, next []byte, offset, limit uint64, total bool) *types.QueryAllInfusionByDestinationRequest {
		return &types.QueryAllInfusionByDestinationRequest{
			DestinationId: destId,
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}

	t.Run("QueryDestination1", func(t *testing.T) {
		resp, err := keeper.InfusionAllByDestination(wctx, request(destination1, nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(infusions1), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(infusions1),
			nullify.Fill(resp.Infusion),
		)
	})

	t.Run("QueryDestination2", func(t *testing.T) {
		resp, err := keeper.InfusionAllByDestination(wctx, request(destination2, nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(infusions2), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(infusions2),
			nullify.Fill(resp.Infusion),
		)
	})

	t.Run("QueryNonExistentDestination", func(t *testing.T) {
		resp, err := keeper.InfusionAllByDestination(wctx, request("non-existent", nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, 0, int(resp.Pagination.Total))
		require.Empty(t, resp.Infusion)
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.InfusionAllByDestination(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
