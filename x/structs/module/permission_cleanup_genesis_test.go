package structs_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	structs "structs/x/structs/module"
	"structs/x/structs/types"
)

func TestGenesis_PermissionCleanupSurvivesWireExportImport(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	structs.InitGenesis(ctx, k, *types.DefaultGenesis())
	const objectId = "6-999"
	k.SetAllocationCount(ctx, 1000)
	const grants = types.PermissionCleanupBudget*2 + 10
	for i := 0; i < grants; i++ {
		k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(objectId, fmt.Sprintf("1-%d", i+1)), types.PermPlay)
		k.SetGuildRankPermissionStoreOnly(ctx, objectId, fmt.Sprintf("0-%d", i+1), types.PermPlay, 3)
	}
	k.EnqueuePermissionCleanup(ctx, objectId)
	k.ProcessPermissionCleanupQueue(ctx)
	exported := structs.ExportGenesis(ctx, k)
	require.Equal(t, []string{objectId}, exported.PermissionCleanupQueue)
	require.NotEmpty(t, exported.PermissionList)
	require.NotEmpty(t, exported.GuildRankPermissionList)
	wire, err := proto.Marshal(exported)
	require.NoError(t, err)
	var decoded types.GenesisState
	require.NoError(t, proto.Unmarshal(wire, &decoded))
	k2, ctx2 := keepertest.StructsKeeper(t)
	structs.InitGenesis(ctx2, k2, decoded)
	require.Equal(t, []string{objectId}, k2.GetPermissionCleanupQueue(ctx2))
	require.Equal(t, k.GetPermissionsByObject(ctx, objectId), k2.GetPermissionsByObject(ctx2, objectId))
	for pass := 0; pass < 10 && len(k2.GetPermissionCleanupQueue(ctx2)) > 0; pass++ {
		k2.ProcessPermissionCleanupQueue(ctx2)
	}
	require.Empty(t, k2.GetPermissionCleanupQueue(ctx2))
	require.Empty(t, k2.GetPermissionsByObject(ctx2, objectId))
	for i := 0; i < grants; i++ {
		require.Equal(t, [types.PermissionBitCount]uint64{}, k2.ReadGuildRankRegister(ctx2, objectId, fmt.Sprintf("0-%d", i+1)))
	}
}

func TestGenesis_RejectsCleanupOfLiveObjectBeforeImport(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	gs := types.DefaultGenesis()
	gs.AllocationList = []types.Allocation{{Id: "6-1"}}
	gs.PermissionCleanupQueue = []string{"6-1"}
	require.Panics(t, func() { structs.InitGenesis(ctx, k, *gs) })
	require.Empty(t, k.GetPermissionCleanupQueue(ctx))
}
