package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

func TestPermissionPlayerIndexFollowsWritesAndCleanup(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	page := func(player string) []*types.PermissionRecord {
		resp, err := k.PermissionByPlayer(ctx, &types.QueryAllPermissionByPlayerRequest{PlayerId: player})
		require.NoError(t, err)
		return resp.PermissionRecords
	}
	id := []byte("6-1@1-1")
	cc := k.NewCurrentContext(ctx)
	cc.PermissionAdd(id, types.PermUpdate)
	cc.CommitAll()
	require.Equal(t, []*types.PermissionRecord{{PermissionId: string(id), Value: uint64(types.PermUpdate)}}, page("1-1"))
	cc = k.NewCurrentContext(ctx)
	cc.PermissionAdd(id, types.PermDelete)
	cc.CommitAll()
	require.Equal(t, []*types.PermissionRecord{{PermissionId: string(id), Value: uint64(types.PermUpdate | types.PermDelete)}}, page("1-1"))
	cc = k.NewCurrentContext(ctx)
	cc.PermissionRemove(id, types.PermUpdate|types.PermDelete)
	cc.CommitAll()
	require.Empty(t, page("1-1"), "single-row deletion clears its index")

	const count = types.PermissionCleanupBudget + 10
	for i := 1; i <= count; i++ {
		k.SetPermissionsByBytes(ctx, []byte(fmt.Sprintf("6-2@1-%d", i)), types.PermPlay)
	}
	k.SetPermissionsByBytes(ctx, []byte("6-3@1-1"), types.PermDelete)
	k.EnqueuePermissionCleanup(ctx, "6-2")
	for pass := 0; pass < 3; pass++ {
		k.ProcessPermissionCleanupQueue(ctx)
	}
	require.Empty(t, k.GetPermissionCleanupQueue(ctx))
	for i := 1; i <= count; i++ {
		if i == 1 {
			require.Equal(t, []*types.PermissionRecord{{PermissionId: "6-3@1-1", Value: uint64(types.PermDelete)}}, page("1-1"))
		} else {
			require.Empty(t, page(fmt.Sprintf("1-%d", i)), "batched deletion clears each player's index")
		}
	}
}
