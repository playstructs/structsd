package app_test

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

func TestPermissionPlayerIndexMigrationBackfillsLegacyRecords(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	store := ctx.KVStore(bApp.GetKey(types.StoreKey))
	id := "6-999@1-999"
	value := make([]byte, 8)
	binary.BigEndian.PutUint64(value, uint64(types.PermDelete))
	store.Set([]byte(types.PermissionKey+id), value)
	store.Set([]byte("Permission/player/1-999\x006-1000@1-999"), []byte{1})
	for pass := 0; pass < 2; pass++ {
		before := len(ctx.EventManager().Events())
		require.NoError(t, v22.MigratePermissionPlayerIndex(ctx, &upgrades.Keepers{StructsKeeper: bApp.StructsKeeper}))
		require.Len(t, ctx.EventManager().Events(), before)
		page, err := bApp.StructsKeeper.PermissionByPlayer(ctx, &types.QueryAllPermissionByPlayerRequest{PlayerId: "1-999"})
		require.NoError(t, err)
		require.Equal(t, []*types.PermissionRecord{{PermissionId: id, Value: uint64(types.PermDelete)}}, page.PermissionRecords)
		require.Equal(t, value, store.Get([]byte(types.PermissionKey+id)))
	}
}
