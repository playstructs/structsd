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

func TestPlayerQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNPlayer(keeper, ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetPlayerRequest
		response *types.QueryGetPlayerResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetPlayerRequest{Id: msgs[0].Id},
			response: &types.QueryGetPlayerResponse{Player: msgs[0]},
		},
		{
			desc:     "Second",
			request:  &types.QueryGetPlayerRequest{Id: msgs[1].Id},
			response: &types.QueryGetPlayerResponse{Player: msgs[1]},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetPlayerRequest{Id: uint64(len(msgs))},
			err:     sdkerrors.ErrKeyNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Player(wctx, tc.request)
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

func TestPlayerQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNPlayer(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllPlayerRequest {
		return &types.QueryAllPlayerRequest{
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
			resp, err := keeper.PlayerAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Player), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Player),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.PlayerAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Player), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Player),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.PlayerAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Player),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.PlayerAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
