package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "structs/testutil/keeper"
	"structs/testutil/nullify"
	"structs/x/structs/types"
)

func TestAllocationProposalQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNAllocationProposal(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetAllocationProposalRequest
		response *types.QueryGetAllocationProposalResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetAllocationProposalRequest{Id: msgs[0].Id},
			response: &types.QueryGetAllocationProposalResponse{AllocationProposal: msgs[0]},
		},
		{
			desc:     "Second",
			request:  &types.QueryGetAllocationProposalRequest{Id: msgs[1].Id},
			response: &types.QueryGetAllocationProposalResponse{AllocationProposal: msgs[1]},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetAllocationProposalRequest{Id: uint64(len(msgs))},
			err:     sdkerrors.ErrKeyNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.AllocationProposal(wctx, tc.request)
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

func TestAllocationProposalQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNAllocationProposal(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllAllocationProposalRequest {
		return &types.QueryAllAllocationProposalRequest{
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
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.AllocationProposalAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.AllocationProposal), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.AllocationProposal),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.AllocationProposalAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.AllocationProposal), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.AllocationProposal),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.AllocationProposalAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.AllocationProposal),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.AllocationProposalAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
