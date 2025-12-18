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
	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Helper function to create N structs for testing (to avoid conflict with struct_test.go)
func createNStructForQuery(keeper keeper.Keeper, ctx sdk.Context, n int) []types.Struct {
	items := make([]types.Struct, n)
	for i := range items {
		items[i] = types.Struct{
			Creator: "cosmos1creator" + string(rune(i)),
			Owner:   "cosmos1owner" + string(rune(i)),
			Type:    uint64(i % 3), // Different types for variety
		}
		items[i] = keeper.AppendStruct(ctx, items[i])
	}
	return items
}

func TestStructQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNStructForQuery(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetStructRequest
		response *types.QueryGetStructResponse
		err      error
	}{
		{
			desc:    "First",
			request: &types.QueryGetStructRequest{Id: msgs[0].Id},
			response: &types.QueryGetStructResponse{
				Struct:           msgs[0],
				GridAttributes:   &types.GridAttributes{},
				StructAttributes: &types.StructAttributes{},
				StructDefenders:  []string{},
			},
		},
		{
			desc:    "Second",
			request: &types.QueryGetStructRequest{Id: msgs[1].Id},
			response: &types.QueryGetStructResponse{
				Struct:           msgs[1],
				GridAttributes:   &types.GridAttributes{},
				StructAttributes: &types.StructAttributes{},
				StructDefenders:  []string{},
			},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetStructRequest{Id: "non-existent"},
			err:     types.ErrObjectNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Struct(wctx, tc.request)
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

func TestStructQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNStructForQuery(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllStructRequest {
		return &types.QueryAllStructRequest{
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
			resp, err := keeper.StructAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Struct), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Struct),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.StructAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Struct), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Struct),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.StructAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Struct),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.StructAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func TestStructAttributeQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNStructForQuery(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetStructAttributeRequest
		response *types.QueryGetStructAttributeResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetStructAttributeRequest{
				StructId:      msgs[0].Id,
				AttributeType: "health",
			},
			response: &types.QueryGetStructAttributeResponse{
				Attribute: 0,
			},
		},
		{
			desc: "Second",
			request: &types.QueryGetStructAttributeRequest{
				StructId:      msgs[1].Id,
				AttributeType: "status",
			},
			response: &types.QueryGetStructAttributeResponse{
				Attribute: 0,
			},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetStructAttributeRequest{
				StructId:      "non-existent",
				AttributeType: "health",
			},
			response: &types.QueryGetStructAttributeResponse{
				Attribute: 0,
			},
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.StructAttribute(wctx, tc.request)
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

func TestStructAttributeQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNStructForQuery(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllStructAttributeRequest {
		return &types.QueryAllStructAttributeRequest{
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
			resp, err := keeper.StructAttributeAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.StructAttributeRecords), step)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.StructAttributeAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.StructAttributeRecords), step)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.StructAttributeAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.StructAttributeAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
