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

func TestPlayerQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test players
	player1 := types.Player{
		Creator:        "cosmos1creator1",
		PrimaryAddress: "cosmos1creator1",
	}
	player2 := types.Player{
		Creator:        "cosmos1creator2",
		PrimaryAddress: "cosmos1creator2",
	}

	created1 := keeper.AppendPlayer(ctx, player1)
	created2 := keeper.AppendPlayer(ctx, player2)

	// Set grid attributes and inventory for testing
	gridAttr1 := types.GridAttributes{
		Ore: 100,
	}
	gridAttr2 := types.GridAttributes{
		Ore: 200,
	}

	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, created1.Id), gridAttr1.Ore)
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, created2.Id), gridAttr2.Ore)

	tests := []struct {
		desc     string
		request  *types.QueryGetPlayerRequest
		response *types.QueryGetPlayerResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetPlayerRequest{
				Id: created1.Id,
			},
			response: &types.QueryGetPlayerResponse{
				Player:          created1,
				GridAttributes:  &gridAttr1,
				PlayerInventory: &types.PlayerInventory{},
				Halted:          false,
			},
		},
		{
			desc: "Second",
			request: &types.QueryGetPlayerRequest{
				Id: created2.Id,
			},
			response: &types.QueryGetPlayerResponse{
				Player:          created2,
				GridAttributes:  &gridAttr2,
				PlayerInventory: &types.PlayerInventory{},
				Halted:          false,
			},
		},
		{
			desc: "NonExistent",
			request: &types.QueryGetPlayerRequest{
				Id: "non-existent",
			},
			err: types.ErrObjectNotFound,
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

func TestPlayerAllQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test players
	players := make([]types.Player, 5)
	for i := range players {
		player := types.Player{
			Creator:        "cosmos1creator" + string(rune(i)),
			PrimaryAddress: "cosmos1creator" + string(rune(i)),
		}
		created := keeper.AppendPlayer(ctx, player)
		players[i] = created
	}

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
		for i := 0; i < len(players); i += step {
			resp, err := keeper.PlayerAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Player), step)
			require.Subset(t,
				nullify.Fill(players),
				nullify.Fill(resp.Player),
			)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(players); i += step {
			resp, err := keeper.PlayerAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Player), step)
			require.Subset(t,
				nullify.Fill(players),
				nullify.Fill(resp.Player),
			)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.PlayerAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(players), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(players),
			nullify.Fill(resp.Player),
		)
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.PlayerAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func TestPlayerHaltedAllQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Create test players
	players := make([]types.Player, 3)
	for i := range players {
		player := types.Player{
			Creator:        "cosmos1creator" + string(rune(i)),
			PrimaryAddress: "cosmos1creator" + string(rune(i)),
		}
		created := keeper.AppendPlayer(ctx, player)
		players[i] = created
	}

	// Halt some players
	keeper.PlayerHalt(ctx, players[0].Id)
	keeper.PlayerHalt(ctx, players[2].Id)

	// Test the query
	response, err := keeper.PlayerHaltedAll(wctx, &types.QueryAllPlayerHaltedRequest{})
	require.NoError(t, err)
	require.Len(t, response.PlayerId, 2)
	require.Contains(t, response.PlayerId, players[0].Id)
	require.Contains(t, response.PlayerId, players[2].Id)
	require.NotContains(t, response.PlayerId, players[1].Id)
}
