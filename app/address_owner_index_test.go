package app_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestAddressOwnerIndexLifecycleAndUpgradeRebuild(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	k := bApp.StructsKeeper
	first, _, _ := newTransferPlayer(t, bApp, ctx)
	second, _, _ := newTransferPlayer(t, bApp, ctx)
	firstIndex := k.GetPlayerIndexFromAddress(ctx, first.String())
	secondIndex := k.GetPlayerIndexFromAddress(ctx, second.String())
	address := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	key := func(index uint64) []byte {
		return []byte(fmt.Sprintf("Address/owner/%d/%s", index, address))
	}
	store := ctx.KVStore(bApp.GetKey(types.StoreKey))
	check := func(owner uint64) {
		t.Helper()
		require.Equal(t, owner, k.GetPlayerIndexFromAddress(ctx, address))
		require.Equal(t, owner == firstIndex, store.Has(key(firstIndex)))
		require.Equal(t, owner == secondIndex, store.Has(key(secondIndex)))
	}
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, address, firstIndex))
	check(firstIndex)
	cc := k.NewCurrentContext(ctx)
	cc.SetPlayerIndexForAddress(address, secondIndex)
	check(firstIndex) // staged ownership must not leak to other operations
	cc.CommitAll()
	check(secondIndex)
	// A cached revoke may carry an obsolete owner argument. Clear the key
	// under the actual association, or the old owner keeps a stale inventory.
	cc = k.NewCurrentContext(ctx)
	cc.RevokePlayerIndexForAddress(address, firstIndex)
	cc.CommitAll()
	check(0)
	cc = k.NewCurrentContext(ctx)
	cc.GenesisImportAddress(address, firstIndex)
	cc.CommitAll()
	check(firstIndex)
	// Multiple rewrites in one operation publish only the final association.
	cc = k.NewCurrentContext(ctx)
	cc.SetPlayerIndexForAddress(address, secondIndex)
	cc.RevokePlayerIndexForAddress(address, secondIndex)
	cc.SetPlayerIndexForAddress(address, firstIndex)
	cc.CommitAll()
	check(firstIndex)
	branch, _ := ctx.CacheContext()
	cc = k.NewCurrentContext(branch)
	cc.SetPlayerIndexForAddress(address, secondIndex)
	cc.CommitAll()
	check(firstIndex) // discarded transaction includes its secondary indexes
	// Simulate pre-upgrade state with canonical associations but no reverse
	// index, plus an obsolete row. Rebuild must restore only live associations.
	reverse := prefix.NewStore(store, []byte("Address/owner/"))
	iterator := reverse.Iterator(nil, nil)
	var keys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, append([]byte{}, iterator.Key()...))
	}
	iterator.Close()
	for _, key := range keys {
		reverse.Delete(key)
	}
	store.Set(key(secondIndex), []byte{1})
	k.RebuildAddressOwnerIndex(ctx)
	check(firstIndex)
	for _, association := range k.GetAllAddressExport(ctx) {
		require.True(t, store.Has([]byte(fmt.Sprintf("Address/owner/%d/%s", association.PlayerIndex, association.Address))))
	}
	k.RebuildAddressOwnerIndex(ctx)
	check(firstIndex)
}
