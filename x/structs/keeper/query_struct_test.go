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
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Helper function to create N structs for testing (to avoid conflict with struct_test.go)
func createNStructForQuery(k keeperlib.Keeper, ctx sdk.Context, n int) []types.Struct {
	items := make([]types.Struct, n)
	for i := range items {
		items[i] = types.Struct{
			Creator: "cosmos1creator" + string(rune('a'+i)),
			Owner:   "cosmos1owner" + string(rune('a'+i)),
			Type:    uint64(i % 3),
		}
		items[i] = testAppendStruct(k, ctx, items[i])

		healthAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, items[i].Id)
		k.SetStructAttribute(ctx, healthAttrId, uint64(100+i))
	}
	return items
}

func TestStructQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNStructForQuery(keeper, ctx, 2)
	address := sdk.AccAddress("struct-query-primary")
	owner := testAppendPlayer(keeper, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
	states := []uint64{
		uint64(types.StructStateMaterialized | types.StructStateBuilt | types.StructStateOnline),
		uint64(types.StructStateMaterialized | types.StructStateBuilt | types.StructStateDestroyed),
	}
	for i := range msgs {
		msgs[i].Creator, msgs[i].Owner, msgs[i].Type = address.String(), owner.Id, 20
		keeper.SetStruct(ctx, msgs[i])
		keeper.SetStructAttribute(ctx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, msgs[i].Id), states[i])
	}
	keeper.SetStructAttribute(ctx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, msgs[1].Id), 0)
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, msgs[0].Id), 14)
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, msgs[0].Id), 3)
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, msgs[0].Id), 7)
	keeper.SetStructDefender(ctx, msgs[0].Id, msgs[0].Index, "5-999")

	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetStructRequest
		response *types.QueryGetStructResponse
		err      error
	}{
		{
			desc:    "OnlineGenerator",
			request: &types.QueryGetStructRequest{Id: msgs[0].Id},
			response: &types.QueryGetStructResponse{
				Struct:           msgs[0],
				GridAttributes:   &types.GridAttributes{Capacity: 14, Load: 3, Fuel: 7},
				StructAttributes: &types.StructAttributes{Health: 100, Status: states[0], IsMaterialized: true, IsBuilt: true, IsOnline: true},
				StructDefenders:  []string{"5-999"},
			},
		},
		{
			desc:    "DestroyedGeneratorBeforeSweep",
			request: &types.QueryGetStructRequest{Id: msgs[1].Id},
			response: &types.QueryGetStructResponse{
				Struct:           msgs[1],
				GridAttributes:   &types.GridAttributes{},
				StructAttributes: &types.StructAttributes{Status: states[1], IsMaterialized: true, IsBuilt: true, IsDestroyed: true},
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
			eventsBefore := len(ctx.EventManager().Events())
			response, err := keeper.Struct(wctx, tc.request)
			require.Len(t, ctx.EventManager().Events(), eventsBefore)
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
		var collected []types.Struct
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.StructAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.Len(t, resp.Struct, min(step, len(msgs)-i))
			collected = append(collected, resp.Struct...)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Struct),
			)
		}
		require.ElementsMatch(t, msgs, collected)
	})
	t.Run("ByKey", func(t *testing.T) {
		var collected []types.Struct
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.StructAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.Len(t, resp.Struct, min(step, len(msgs)-i))
			collected = append(collected, resp.Struct...)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Struct),
			)
			next = resp.Pagination.NextKey
		}
		require.Empty(t, next)
		require.ElementsMatch(t, msgs, collected)
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.StructAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		// CountTotal is forced off by boundedPagination: it scans the whole
		// prefix however small the page, so capping the page alone would not
		// bound the work. The set stays fully reachable - see the assertion
		// below - only the count is gone.
		require.Zero(t, resp.Pagination.Total, "CountTotal must stay off; total is reported as 0")
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
				Attribute: 100,
			},
		},
		{
			desc: "Second",
			request: &types.QueryGetStructAttributeRequest{
				StructId:      msgs[1].Id,
				AttributeType: "health",
			},
			response: &types.QueryGetStructAttributeResponse{
				Attribute: 101,
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
	var expected []*types.StructAttributeRecord
	for i, structure := range msgs {
		expected = append(expected, &types.StructAttributeRecord{
			AttributeId: keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, structure.Id),
			Value: uint64(100+i),
		})
	}


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
		var collected []*types.StructAttributeRecord
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.StructAttributeAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.Len(t, resp.StructAttributeRecords, min(step, len(expected)-i))
			collected = append(collected, resp.StructAttributeRecords...)
		}
		require.ElementsMatch(t, expected, collected)
	})
	t.Run("ByKey", func(t *testing.T) {
		var collected []*types.StructAttributeRecord
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.StructAttributeAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.Len(t, resp.StructAttributeRecords, min(step, len(expected)-i))
			collected = append(collected, resp.StructAttributeRecords...)
			next = resp.Pagination.NextKey
		}
		require.Empty(t, next)
		require.ElementsMatch(t, expected, collected)
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.StructAttributeAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		// CountTotal is forced off by boundedPagination: it scans the whole
		// prefix however small the page, so capping the page alone would not
		// bound the work. The set stays fully reachable - see the assertion
		// below - only the count is gone.
		require.Zero(t, resp.Pagination.Total, "CountTotal must stay off; total is reported as 0")
		require.ElementsMatch(t, expected, resp.StructAttributeRecords)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.StructAttributeAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
