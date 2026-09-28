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
	ctxSDK := ctx
	wctx := sdk.WrapSDKContext(ctx)

	// Create test players
	player1 := types.Player{
		Creator:        sdk.AccAddress("queryplayer-primary1").String(),
		PrimaryAddress: sdk.AccAddress("queryplayer-primary1").String(),
	}
	player2 := types.Player{
		Creator:        sdk.AccAddress("queryplayer-primary2").String(),
		PrimaryAddress: sdk.AccAddress("queryplayer-primary2").String(),
	}

	created1 := testAppendPlayer(keeper, ctxSDK, player1)
	created2 := testAppendPlayer(keeper, ctxSDK, player2)

	// Set grid attributes and inventory for testing
	gridAttr1 := types.GridAttributes{
		Ore: 100, StructsLoad: 123, Capacity: 1000, Load: 50,
	}
	gridAttr2 := types.GridAttributes{
		Ore: 200, StructsLoad: 789, Capacity: 2000, Load: 75,
	}

	// This is a query fixture: seed distinct stored attributes explicitly rather
	// than depending on an obsolete automatic player-load reservation.
	for _, fixture := range []struct { player types.Player; grid types.GridAttributes; alpha int64 }{
		{created1, gridAttr1, 50}, {created2, gridAttr2, 75},
	} {
		for attribute, amount := range map[types.GridAttributeType]uint64{
			types.GridAttributeType_ore: fixture.grid.Ore,
			types.GridAttributeType_structsLoad: fixture.grid.StructsLoad,
			types.GridAttributeType_capacity: fixture.grid.Capacity,
			types.GridAttributeType_load: fixture.grid.Load,
		} {
			keeper.SetGridAttribute(ctxSDK, keeperlib.GetGridAttributeIDByObjectId(attribute, fixture.player.Id), amount)
		}
		vault, err := types.VaultAddress(fixture.player.Id)
		require.NoError(t, err)
		coins := sdk.NewCoins(sdk.NewInt64Coin("ualpha", fixture.alpha))
		require.NoError(t, keeper.BankKeeper().MintCoins(ctxSDK, types.ModuleName, coins))
		require.NoError(t, keeper.BankKeeper().SendCoinsFromModuleToAccount(ctxSDK, types.ModuleName, vault, coins))
	}

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
				PlayerInventory: &types.PlayerInventory{Rocks: sdk.NewInt64Coin("ualpha", 50)},
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
				PlayerInventory: &types.PlayerInventory{Rocks: sdk.NewInt64Coin("ualpha", 75)},
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
				// Check the queried player, each seeded grid attribute and vault inventory.
				require.Equal(t, tc.response.Player.Id, response.Player.Id)
				require.Equal(t, tc.response.Player.Creator, response.Player.Creator)
				require.Equal(t, tc.response.GridAttributes.Ore, response.GridAttributes.Ore)
				require.Equal(t, tc.response.GridAttributes.StructsLoad, response.GridAttributes.StructsLoad)
				require.Equal(t, tc.response.GridAttributes.Capacity, response.GridAttributes.Capacity)
				require.Equal(t, tc.response.GridAttributes.Load, response.GridAttributes.Load)
				require.Equal(t, tc.response.PlayerInventory, response.PlayerInventory)
			}
		})
	}
}

func TestPlayerAllQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	ctxSDK := ctx
	wctx := sdk.WrapSDKContext(ctx)

	// Create test players
	players := make([]types.Player, 5)
	for i := range players {
		player := types.Player{
			Creator:        "cosmos1creator" + string(rune(i)),
			PrimaryAddress: "cosmos1creator" + string(rune(i)),
		}
		created := testAppendPlayer(keeper, ctxSDK, player)
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
		// CountTotal is forced off by boundedPagination: it scans the whole
		// prefix however small the page, so capping the page alone would not
		// bound the work. The set stays fully reachable - see the assertion
		// below - only the count is gone.
		require.Zero(t, resp.Pagination.Total, "CountTotal must stay off; total is reported as 0")
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

/* TestPlayerHaltedAllQuery is disabled - PlayerHalt and QueryAllPlayerHaltedRequest no longer exist.
func TestPlayerHaltedAllQuery(t *testing.T) {
	...
}
*/
