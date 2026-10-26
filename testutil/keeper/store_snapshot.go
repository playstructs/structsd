package keeper

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// StructsStoreRows snapshots the complete module store for migration/rollback
// assertions without exposing a production keeper store accessor.
func StructsStoreRows(t testing.TB, ctx sdk.Context) map[string]string {
	t.Helper()
	key, ok := ctx.Value(testStoreKeyCtx{}).(*storetypes.KVStoreKey)
	require.True(t, ok, "context must come from keepertest.StructsKeeper")
	it := ctx.KVStore(key).Iterator(nil, nil)
	defer it.Close()
	rows := map[string]string{}
	for ; it.Valid(); it.Next() {
		rows[string(it.Key())] = string(it.Value())
	}
	return rows
}
