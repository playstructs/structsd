package keeper

import (
	"encoding/binary"
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"structs/x/structs/types"
)

// SubstationPlayerKeyPrefix returns the KV store key prefix for looking up
// all players connected to a given substation.
func SubstationPlayerKeyPrefix(substationId string) []byte {
	return []byte(types.SubstationPlayerKey + substationId + "/")
}

// --- Set / Remove ---

// SetSubstationPlayerIndex links a playerId under a substationId in the index.
func (k Keeper) SetSubstationPlayerIndex(ctx context.Context, substationId string, playerId string) (err error) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), SubstationPlayerKeyPrefix(substationId))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, 1)

	store.Set([]byte(playerId), bz)

	return err
}

// RemoveSubstationPlayerIndex removes a playerId from a substationId in the index.
func (k Keeper) RemoveSubstationPlayerIndex(ctx context.Context, substationId string, playerId string) (err error) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), SubstationPlayerKeyPrefix(substationId))
	store.Delete([]byte(playerId))

	return err
}

// ClearSubstationPlayerIndex removes all player entries for a given substationId.
func (k Keeper) ClearSubstationPlayerIndex(ctx context.Context, substationId string) (err error) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), SubstationPlayerKeyPrefix(substationId))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}

	return err
}

// --- Retrieval ---

// GetAllPlayerIdBySubstationIndex returns all player IDs connected to a given substationId.
func (k Keeper) GetAllPlayerIdBySubstationIndex(ctx context.Context, substationId string) (list []string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), SubstationPlayerKeyPrefix(substationId))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		list = append(list, string(iterator.Key()))
	}

	return
}
