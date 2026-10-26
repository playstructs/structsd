package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/testutil/nullify"
	"structs/x/structs/types"
)

func TestSubstationQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	// Query fixtures explicitly persist both healthy and brownout snapshots.
	// Querying them must report stored state, not recalculate or normalize it.
	address := sdk.AccAddress("substation-query-key")
	owner := testAppendPlayer(keeper, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
	reactor := keeper.AppendReactor(ctx, types.Reactor{RawAddress: sdk.ValAddress(address).Bytes()})
	guild := keeper.AppendGuild(ctx, "query-test", "", reactor, owner, "")
	msgs := []types.Substation{
		{Id: "4-1", Creator: address.String(), Owner: owner.Id},
		{Id: "4-10", Creator: address.String(), Owner: guild.Id},
	}
	grids := []types.GridAttributes{
		{Capacity: 100, Load: 25, ConnectionCount: 3, ConnectionCapacity: 25},
		{Capacity: 80, Load: 100, ConnectionCount: 4, ConnectionCapacity: 0},
	}
	for i, substation := range msgs {
		keeper.SetSubstation(ctx, substation)
		for _, attribute := range []struct {
			kind types.GridAttributeType
			value uint64
		}{
			{types.GridAttributeType_capacity, grids[i].Capacity},
			{types.GridAttributeType_load, grids[i].Load},
			{types.GridAttributeType_connectionCount, grids[i].ConnectionCount},
			{types.GridAttributeType_connectionCapacity, grids[i].ConnectionCapacity},
		} {
			keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(attribute.kind, substation.Id), attribute.value)
		}
	}
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetSubstationRequest
		response *types.QueryGetSubstationResponse
		err      error
	}{
		{
			desc:    "PlayerOwnedHealthy",
			request: &types.QueryGetSubstationRequest{Id: msgs[0].Id},
			response: &types.QueryGetSubstationResponse{
				Substation:     msgs[0],
				GridAttributes: &grids[0],
			},
		},
		{
			desc:    "GuildOwnedBrownout",
			request: &types.QueryGetSubstationRequest{Id: msgs[1].Id},
			response: &types.QueryGetSubstationResponse{
				Substation:     msgs[1],
				GridAttributes: &grids[1],
			},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetSubstationRequest{Id: "non-existent"},
			err:     types.ErrObjectNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			eventsBefore := len(ctx.EventManager().Events())
			response, err := keeper.Substation(wctx, tc.request)
			require.Len(t, ctx.EventManager().Events(), eventsBefore, "queries must not emit mutation events")
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

func TestSubstationQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNSubstation(t, keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllSubstationRequest {
		return &types.QueryAllSubstationRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		var collected []types.Substation
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.SubstationAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.Len(t, resp.Substation, min(step, len(msgs)-i))
			collected = append(collected, resp.Substation...)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Substation),
			)
		}
		require.ElementsMatch(t, msgs, collected, "offset pages must return every substation exactly once")
	})
	t.Run("ByKey", func(t *testing.T) {
		var collected []types.Substation
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.SubstationAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.Len(t, resp.Substation, min(step, len(msgs)-i))
			collected = append(collected, resp.Substation...)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Substation),
			)
			next = resp.Pagination.NextKey
		}
		require.Empty(t, next, "the final key page must terminate")
		require.ElementsMatch(t, msgs, collected, "key pages must return every substation exactly once")
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.SubstationAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		// CountTotal is forced off by boundedPagination: it scans the whole
		// prefix however small the page, so capping the page alone would not
		// bound the work. The set stays fully reachable - see the assertion
		// below - only the count is gone.
		require.Zero(t, resp.Pagination.Total, "CountTotal must stay off; total is reported as 0")
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Substation),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.SubstationAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
