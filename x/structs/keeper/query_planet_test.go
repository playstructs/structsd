package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPlanetQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Explicit post-migration planet clocks and active quantities are query
	// state; the old per-struct ore clocks are not their source.
	address := sdk.AccAddress("planet-query-primary")
	owner := testAppendPlayer(keeper, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
	planet := testAppendPlanet(keeper, ctx, types.Planet{Creator: address.String(), Owner: owner.Id})
	for _, attribute := range []struct {
		kind types.PlanetAttributeType
		value uint64
	}{
		{types.PlanetAttributeType_planetBlockStartOreMine, 17},
		{types.PlanetAttributeType_planetBlockStartOreRefine, 23},
		{types.PlanetAttributeType_oreMiningActiveQuantity, 2},
		{types.PlanetAttributeType_oreRefiningActiveQuantity, 1},
		{types.PlanetAttributeType_defensiveCannonQuantity, 3},
	} {
		keeper.SetPlanetAttribute(ctx, keeperlib.GetPlanetAttributeIDByObjectId(attribute.kind, planet.Id), attribute.value)
	}

	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetPlanetRequest
		response *types.QueryGetPlanetResponse
		err      error
	}{
		{
			desc: "ValidPlanet",
			request: &types.QueryGetPlanetRequest{
				Id: planet.Id,
			},
			response: &types.QueryGetPlanetResponse{
				Planet: planet,
				GridAttributes: &types.GridAttributes{
					Ore: types.PlanetStartingOre,
				},
				PlanetAttributes: &types.PlanetAttributes{
					PlanetaryShield: types.PlanetaryShieldBase,
					BlockStartOreMine: 17, BlockStartOreRefine: 23,
					OreMiningActiveQuantity: 2, OreRefiningActiveQuantity: 1,
					DefensiveCannonQuantity: 3,
				},
			},
		},
		{
			desc: "NonExistentPlanet",
			request: &types.QueryGetPlanetRequest{
				Id: "non-existent",
			},
			err: types.ErrObjectNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			eventsBefore := len(ctx.EventManager().Events())
			response, err := keeper.Planet(wctx, tc.request)
			require.Len(t, ctx.EventManager().Events(), eventsBefore)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.response.Planet, response.Planet)
				require.Equal(t, tc.response.GridAttributes, response.GridAttributes)
				require.Equal(t, tc.response.PlanetAttributes, response.PlanetAttributes)
			}
		})
	}
}

func TestPlanetAllQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test planets
	planets := make([]types.Planet, 4)
	for i := range planets {
		planets[i] = testAppendPlanet(keeper, ctx, types.Planet{
			Creator: "creator" + string(rune(i)),
			Owner:   "player" + string(rune(i)),
		})
	}

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllPlanetRequest {
		return &types.QueryAllPlanetRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}

	t.Run("ByOffset", func(t *testing.T) {
		var collected []types.Planet
		step := 2
		for i := 0; i < len(planets); i += step {
			resp, err := keeper.PlanetAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.Len(t, resp.Planet, min(step, len(planets)-i))
			collected = append(collected, resp.Planet...)
		}
		require.ElementsMatch(t, planets, collected)
	})

	t.Run("ByKey", func(t *testing.T) {
		var collected []types.Planet
		step := 2
		var next []byte
		for i := 0; i < len(planets); i += step {
			resp, err := keeper.PlanetAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.Len(t, resp.Planet, min(step, len(planets)-i))
			collected = append(collected, resp.Planet...)
			next = resp.Pagination.NextKey
		}
		require.Empty(t, next)
		require.ElementsMatch(t, planets, collected)
	})

	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.PlanetAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		// CountTotal is forced off by boundedPagination: it scans the whole
		// prefix however small the page, so capping the page alone would not
		// bound the work. The set stays fully reachable - see the assertion
		// below - only the count is gone.
		require.Zero(t, resp.Pagination.Total, "CountTotal must stay off; total is reported as 0")
		require.ElementsMatch(t, planets, resp.Planet)
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.PlanetAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func TestPlanetAllByPlayerQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create 2 planets for player1 and 1 for player2
	testAppendPlanet(keeper, ctx, types.Planet{Creator: "creator1", Owner: "player1"})
	testAppendPlanet(keeper, ctx, types.Planet{Creator: "creator1", Owner: "player1"})
	testAppendPlanet(keeper, ctx, types.Planet{Creator: "creator2", Owner: "player2"})

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllPlanetByPlayerRequest {
		return &types.QueryAllPlanetByPlayerRequest{
			PlayerId: "player1",
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}

	t.Run("ByOffset", func(t *testing.T) {
		step := 1
		for i := 0; i < 2; i += step {
			resp, err := keeper.PlanetAllByPlayer(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Planet), step)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 1
		var next []byte
		for i := 0; i < 2; i += step {
			resp, err := keeper.PlanetAllByPlayer(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Planet), step)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.PlanetAllByPlayer(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		// Pagination.Total counts all planets (not just filtered), since filtering is done in-memory
		// CountTotal is forced off by boundedPagination: it scans the whole
		// prefix however small the page, so capping the page alone would not
		// bound the work. The set stays fully reachable - see the assertion
		// below - only the count is gone.
		require.Zero(t, resp.Pagination.Total, "CountTotal must stay off; total is reported as 0")
		require.Len(t, resp.Planet, 2)
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.PlanetAllByPlayer(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func TestPlanetOwnerPagesIgnoreUnrelatedPlanets(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	want := []types.Planet{}
	for i := 0; i < 7; i++ {
		owner := "1-2"
		if i == 2 || i == 5 { owner = "1-1" }
		planet := testAppendPlanet(k, ctx, types.Planet{Owner: owner})
		if owner == "1-1" { want = append(want, planet) }
	}
	var next []byte
	for i := range want {
		page, err := k.PlanetAllByPlayer(ctx, &types.QueryAllPlanetByPlayerRequest{PlayerId: "1-1", Pagination: &query.PageRequest{Key: next, Limit: 1}})
		require.NoError(t, err)
		require.Equal(t, []types.Planet{want[i]}, page.Planet)
		next = page.Pagination.NextKey
	}
	require.Empty(t, next)
	page, err := k.PlanetAllByPlayer(ctx, &types.QueryAllPlanetByPlayerRequest{PlayerId: "1-1", Pagination: &query.PageRequest{Offset: 1, Limit: 1}})
	require.NoError(t, err)
	require.Equal(t, []types.Planet{want[1]}, page.Planet)
	// Updating ownership must remove the exact old index row; deletion must
	// remove the new one rather than leaving an unreadable query entry.
	moved := want[0]
	moved.Owner = "1-10"
	k.SetPlanet(ctx, moved)
	page, err = k.PlanetAllByPlayer(ctx, &types.QueryAllPlanetByPlayerRequest{PlayerId: "1-1"})
	require.NoError(t, err)
	require.Equal(t, []types.Planet{want[1]}, page.Planet)
	page, err = k.PlanetAllByPlayer(ctx, &types.QueryAllPlanetByPlayerRequest{PlayerId: "1-10"})
	require.NoError(t, err)
	require.Equal(t, []types.Planet{moved}, page.Planet)
	k.RemovePlanet(ctx, moved.Id)
	page, err = k.PlanetAllByPlayer(ctx, &types.QueryAllPlanetByPlayerRequest{PlayerId: "1-10"})
	require.NoError(t, err)
	require.Empty(t, page.Planet)
}

func TestPlanetAttributeQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test planet
	planet := testAppendPlanet(keeper, ctx, types.Planet{Creator: "test-creator", Owner: "test-player"})

	// Set test attribute
	attributeType := types.PlanetAttributeType_planetaryShield
	attributeId := keeperlib.GetPlanetAttributeIDByObjectId(attributeType, planet.Id)
	keeper.SetPlanetAttribute(ctx, attributeId, 100)

	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetPlanetAttributeRequest
		response *types.QueryGetPlanetAttributeResponse
		err      error
	}{
		{
			desc: "ValidAttribute",
			request: &types.QueryGetPlanetAttributeRequest{
				PlanetId:      planet.Id,
				AttributeType: attributeType.String(),
			},
			response: &types.QueryGetPlanetAttributeResponse{
				Attribute: 100,
			},
		},
		{
			desc: "NonExistentAttribute",
			request: &types.QueryGetPlanetAttributeRequest{
				PlanetId:      "non-existent",
				AttributeType: attributeType.String(),
			},
			response: &types.QueryGetPlanetAttributeResponse{
				Attribute: 0,
			},
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.PlanetAttribute(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.response, response)
			}
		})
	}
}

func TestPlanetAttributeAllQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test planet
	planet := testAppendPlanet(keeper, ctx, types.Planet{Creator: "test-creator", Owner: "test-player"})

	// Set multiple attributes
	attributes := []struct {
		attributeType types.PlanetAttributeType
		value         uint64
	}{
		{types.PlanetAttributeType_planetaryShield, 100},
		{types.PlanetAttributeType_repairNetworkQuantity, 50},
		{types.PlanetAttributeType_defensiveCannonQuantity, 25},
	}

	for _, attr := range attributes {
		attributeId := keeperlib.GetPlanetAttributeIDByObjectId(attr.attributeType, planet.Id)
		keeper.SetPlanetAttribute(ctx, attributeId, attr.value)
	}

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllPlanetAttributeRequest {
		return &types.QueryAllPlanetAttributeRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}

	t.Run("ByOffset", func(t *testing.T) {
		step := 1
		for i := 0; i < len(attributes); i += step {
			resp, err := keeper.PlanetAttributeAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.PlanetAttributeRecords), step)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 1
		var next []byte
		for i := 0; i < len(attributes); i += step {
			resp, err := keeper.PlanetAttributeAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.PlanetAttributeRecords), step)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.PlanetAttributeAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		// CountTotal is forced off by boundedPagination: it scans the whole
		// prefix however small the page, so capping the page alone would not
		// bound the work. The set stays fully reachable - see the assertion
		// below - only the count is gone.
		require.Zero(t, resp.Pagination.Total, "CountTotal must stay off; total is reported as 0")
		require.Len(t, resp.PlanetAttributeRecords, len(attributes))
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.PlanetAttributeAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
