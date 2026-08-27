package keeper_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

/* Regression suite for unbounded query pagination.
 *
 * The query endpoints take no authorization and passed req.Pagination straight
 * to query.Paginate. Limit is a uint64 and is not capped, so one request could
 * ask a node to decode and retain an entire collection; and a request with no
 * Limit switched CountTotal on, which walks the whole prefix after the page has
 * been collected - so capping the page alone would not have bounded the work.
 *
 * This bounds the node serving the query rather than consensus. Nothing here
 * touches chain state; what it protects is a node that exposes its query
 * endpoints, against the largest collection it happens to hold.
 */

type paginationFixture struct {
	k   keeperlib.Keeper
	ctx sdk.Context
}

func seedPlayers(t *testing.T, count int) (paginationFixture, []types.Player) {
	t.Helper()

	k, ctx := keepertest.StructsKeeper(t)
	players := make([]types.Player, 0, count)
	for i := 0; i < count; i++ {
		player := types.Player{Id: fmt.Sprintf("1-%d", i+1), Index: uint64(i + 1)}
		k.SetPlayer(ctx, player)
		players = append(players, player)
	}

	return paginationFixture{k: k, ctx: ctx}, players
}

// TestQueryPagination_LimitIsCapped is the direct regression: asking for
// everything must return a page, not everything.
func TestQueryPagination_LimitIsCapped(t *testing.T) {
	f, players := seedPlayers(t, types.QueryPageLimitMaximum+50)

	resp, err := f.k.PlayerAll(f.ctx, &types.QueryAllPlayerRequest{
		Pagination: &query.PageRequest{Limit: ^uint64(0)},
	})
	require.NoError(t, err)
	require.Len(t, resp.Player, types.QueryPageLimitMaximum,
		"a request for the whole collection must be cut to the maximum page")
	require.Less(t, len(resp.Player), len(players))
	require.NotEmpty(t, resp.Pagination.NextKey, "and must hand back a way to continue")
}

// TestQueryPagination_CountTotalIsOff pins the half that actually bounds the
// work. A page can be small while CountTotal still scans the entire prefix.
func TestQueryPagination_CountTotalIsOff(t *testing.T) {
	f, _ := seedPlayers(t, 20)

	resp, err := f.k.PlayerAll(f.ctx, &types.QueryAllPlayerRequest{
		Pagination: &query.PageRequest{Limit: 5, CountTotal: true},
	})
	require.NoError(t, err)
	require.Len(t, resp.Player, 5)
	require.Zero(t, resp.Pagination.Total,
		"CountTotal must be refused; it walks the whole prefix regardless of page size")
}

// A nil pagination must still be bounded, and must not turn CountTotal back on
// through the SDK's own defaults.
func TestQueryPagination_NilRequestIsBounded(t *testing.T) {
	f, _ := seedPlayers(t, types.QueryPageLimitDefault+25)

	resp, err := f.k.PlayerAll(f.ctx, &types.QueryAllPlayerRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Player, types.QueryPageLimitDefault)
	require.Zero(t, resp.Pagination.Total)
}

/* TestQueryPagination_WholeSetStaysReachable is the other half of the contract.
 * The cap bounds one response, not what a client can retrieve: following
 * next_key must still walk the entire collection, or the clamp would be data
 * loss rather than a bound.
 */
func TestQueryPagination_WholeSetStaysReachable(t *testing.T) {
	const total = 250
	f, _ := seedPlayers(t, total)

	seen := map[string]bool{}
	var next []byte
	for pages := 0; ; pages++ {
		require.Less(t, pages, 20, "pagination is not converging")

		resp, err := f.k.PlayerAll(f.ctx, &types.QueryAllPlayerRequest{
			Pagination: &query.PageRequest{Key: next, Limit: 40},
		})
		require.NoError(t, err)
		for _, player := range resp.Player {
			seen[player.Id] = true
		}
		if len(resp.Pagination.NextKey) == 0 {
			break
		}
		next = resp.Pagination.NextKey
	}

	require.Len(t, seen, total, "every player must still be reachable by paging")
}

// A limit under the cap is honoured untouched, so the clamp cannot quietly
// change ordinary requests.
func TestQueryPagination_SmallLimitUnchanged(t *testing.T) {
	f, _ := seedPlayers(t, 50)

	resp, err := f.k.PlayerAll(f.ctx, &types.QueryAllPlayerRequest{
		Pagination: &query.PageRequest{Limit: 7},
	})
	require.NoError(t, err)
	require.Len(t, resp.Player, 7)
}
