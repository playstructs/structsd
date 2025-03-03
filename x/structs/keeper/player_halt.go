package keeper

import (
	"encoding/binary"
    "context"

    "github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"

	"structs/x/structs/types"
)

func (k Keeper) PlayerHalt(ctx context.Context, playerId string) (err error) {
    playerHaltStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerHaltKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, 1)

	playerHaltStore.Set([]byte(playerId), bz)

	return err
}

func (k Keeper) PlayerResume(ctx context.Context, playerId string) (err error) {
    playerHaltStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerHaltKey))
	playerHaltStore.Delete([]byte(playerId))

	return err
}


func (k Keeper) IsPlayerHalted(ctx context.Context, playerId string) (bool) {
    playerHaltStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerHaltKey))
	bz := playerHaltStore.Get(types.KeyPrefix(playerId))

	return bz != nil
}

