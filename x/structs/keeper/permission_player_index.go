package keeper

import (
	"context"
	"strings"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
)

const permissionPlayerIndexPrefix = "Permission/player/"

func permissionPlayerStore(k Keeper, ctx context.Context, player string) prefix.Store {
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte(permissionPlayerIndexPrefix+player+"\x00"))
}

func permissionPlayer(permissionID []byte) (string, bool) {
	_, player, found := strings.Cut(string(permissionID), "@")
	return player, found && player != "" && !strings.ContainsAny(player, "@\x00")
}

func (k Keeper) setPermissionPlayerIndex(ctx context.Context, permissionID []byte) {
	if player, ok := permissionPlayer(permissionID); ok {
		permissionPlayerStore(k, ctx, player).Set(permissionID, []byte{1})
	}
}

func (k Keeper) removePermissionPlayerIndex(ctx context.Context, permissionID []byte) {
	if player, ok := permissionPlayer(permissionID); ok {
		permissionPlayerStore(k, ctx, player).Delete(permissionID)
	}
}

// RebuildPermissionPlayerIndex is an upgrade-only full scan. Genesis imports
// and every live permission write maintain the derived index through the setter.
func (k Keeper) RebuildPermissionPlayerIndex(ctx context.Context) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte(permissionPlayerIndexPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, nil)
	var keys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, append([]byte(nil), iterator.Key()...))
	}
	iterator.Close()
	for _, key := range keys {
		store.Delete(key)
	}
	for _, record := range k.GetAllPermissionExport(ctx) {
		k.setPermissionPlayerIndex(ctx, []byte(record.PermissionId))
	}
}
