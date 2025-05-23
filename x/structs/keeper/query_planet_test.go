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

	// Create test planet
	player := types.Player{
		Id:      "test-player",
		Creator: "test-creator",
	}
	planetId := keeper.AppendPlanet(ctx, player)
	planet, _ := keeper.GetPlanet(ctx, planetId)

	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetPlanetRequest
		response *types.QueryGetPlanetResponse
		err      error
	}{
		{
			desc: "ValidPlanet",
			request: &types.QueryGetPlanetRequest{
				Id: planetId,
			},
			response: &types.QueryGetPlanetResponse{
				Planet: planet,
				GridAttributes: &types.GridAttributes{
					Ore: types.PlanetStartingOre,
				},
				PlanetAttributes: &types.PlanetAttributes{
					PlanetaryShield: types.PlanetaryShieldBase,
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
			response, err := keeper.Planet(wctx, tc.request)
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
		player := types.Player{
			Id:      "player" + string(rune(i)),
			Creator: "creator" + string(rune(i)),
		}
		planetId := keeper.AppendPlanet(ctx, player)
		planet, _ := keeper.GetPlanet(ctx, planetId)
		planets[i] = planet
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
		step := 2
		for i := 0; i < len(planets); i += step {
			resp, err := keeper.PlanetAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Planet), step)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(planets); i += step {
			resp, err := keeper.PlanetAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Planet), step)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.PlanetAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(planets), int(resp.Pagination.Total))
		require.Len(t, resp.Planet, len(planets))
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.PlanetAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func TestPlanetAllByPlayerQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test planets for different players
	player1 := types.Player{
		Id:      "player1",
		Creator: "creator1",
	}
	player2 := types.Player{
		Id:      "player2",
		Creator: "creator2",
	}

	// Create 2 planets for player1 and 1 for player2
	keeper.AppendPlanet(ctx, player1)
	keeper.AppendPlanet(ctx, player1)
	keeper.AppendPlanet(ctx, player2)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllPlanetByPlayerRequest {
		return &types.QueryAllPlanetByPlayerRequest{
			PlayerId: player1.Id,
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
		require.Equal(t, 2, int(resp.Pagination.Total))
		require.Len(t, resp.Planet, 2)
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.PlanetAllByPlayer(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func TestPlanetAttributeQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test planet
	player := types.Player{
		Id:      "test-player",
		Creator: "test-creator",
	}
	planetId := keeper.AppendPlanet(ctx, player)

	// Set test attribute
	attributeType := types.PlanetAttributeType_planetaryShield
	attributeId := keeperlib.GetPlanetAttributeIDByObjectId(attributeType, planetId)
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
				PlanetId:      planetId,
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
	player := types.Player{
		Id:      "test-player",
		Creator: "test-creator",
	}
	planetId := keeper.AppendPlanet(ctx, player)

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
		attributeId := keeperlib.GetPlanetAttributeIDByObjectId(attr.attributeType, planetId)
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
		require.Equal(t, len(attributes), int(resp.Pagination.Total))
		require.Len(t, resp.PlanetAttributeRecords, len(attributes))
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.PlanetAttributeAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
