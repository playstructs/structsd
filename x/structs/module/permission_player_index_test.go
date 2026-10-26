package structs_test

import (
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	structs "structs/x/structs/module"
	"structs/x/structs/types"
)

func TestGenesis_RebuildsPermissionPlayerIndex(t *testing.T) {
	genesis := types.DefaultGenesis()
	genesis.PermissionList = []*types.PermissionRecord{
		{PermissionId: "6-1@1-1", Value: uint64(types.PermUpdate)},
		{PermissionId: "6-2@1-10", Value: uint64(types.PermDelete)},
	}
	for pass := 0; pass < 2; pass++ {
		k, ctx := keepertest.StructsKeeper(t)
		structs.InitGenesis(ctx, k, *genesis)
		for _, expected := range genesis.PermissionList {
			player := "1-1"
			if expected.PermissionId == "6-2@1-10" {
				player = "1-10"
			}
			page, err := k.PermissionByPlayer(ctx, &types.QueryAllPermissionByPlayerRequest{PlayerId: player})
			require.NoError(t, err)
			require.Equal(t, []*types.PermissionRecord{expected}, page.PermissionRecords)
		}
		wire, err := proto.Marshal(structs.ExportGenesis(ctx, k))
		require.NoError(t, err)
		genesis = new(types.GenesisState)
		require.NoError(t, proto.Unmarshal(wire, genesis))
	}
}
