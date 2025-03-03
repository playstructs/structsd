package keeper

import (
	"encoding/binary"
    "context"

    "github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
    sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k Keeper) PlayerHalt(ctx context.Context, playerId string) (err error) {
    playerHaltStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerHaltKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, 1)

	playerHaltStore.Set([]byte(playerId), bz)

    ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPlayerHalted{PlayerId: playerId})

	return err
}

func (k Keeper) PlayerResume(ctx context.Context, playerId string) (err error) {
    playerHaltStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerHaltKey))
	playerHaltStore.Delete([]byte(playerId))

    ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPlayerResumed{PlayerId: playerId})

	return err
}


func (k Keeper) IsPlayerHalted(ctx context.Context, playerId string) (bool) {
    playerHaltStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerHaltKey))
	bz := playerHaltStore.Get(types.KeyPrefix(playerId))

	return bz != nil
}


func (k Keeper) GetAllHaltedPlayerId(ctx context.Context) (list []string) {
	playerHaltStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerHaltKey))
	iterator := storetypes.KVStorePrefixIterator(playerHaltStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		list = append(list, string(iterator.Key()))
	}

    return
}

func (k Keeper) SetAllHaltedPlayerId(ctx context.Context, list []string) {
    for _, element := range list {
        k.PlayerHalt(ctx, element)
    }
}