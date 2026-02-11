package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"

    storetypes "cosmossdk.io/store/types"

	"context"
    //"math"
    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

)


// GetPlayerCount get the total number of player
func (k Keeper) GetPlayerCount(ctx context.Context) uint64 {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.PlayerCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0 {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetPlayerCount set the total number of player
func (k Keeper) SetPlayerCount(ctx context.Context, count uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.PlayerCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}


// SetPlayer set a specific player in the store
func (k Keeper) SetPlayer(ctx context.Context, player types.Player) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	b := k.cdc.MustMarshal(&player)

	store.Set([]byte(player.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPlayer{Player: &player})
}

// GetPlayer returns a player from its id
func (k Keeper) GetPlayer(ctx context.Context, playerId string) (val types.Player, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	b := store.Get([]byte(playerId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

func (k Keeper) GetPlayerFromIndex(ctx context.Context, playerIndex uint64) (val types.Player, found bool) {
    val, found = k.GetPlayer(ctx, GetObjectID(types.ObjectType_player, playerIndex))
    return
}


// GetAllPlayer returns all player
func (k Keeper) GetAllPlayer(ctx context.Context) (list []types.Player) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Player
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		list = append(list, val)
	}

	return
}

// GetAllPlayer returns all player
func (k Keeper) GetAllPlayerBySubstation(ctx context.Context, substationId string) (list []types.Player) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Player
		k.cdc.MustUnmarshal(iterator.Value(), &val)

        if (val.SubstationId == substationId) {
            list = append(list, val)
		}
	}

	return
}


/*
    message PlayerInventory {
      cosmos.base.v1beta1.Coin rocks = 13
      [(gogoproto.nullable) = false, (gogoproto.castrepeated) = "github.com/cosmos/cosmos-sdk/types.Coins"];
    }
*/

func (k Keeper) GetPlayerInventory(ctx context.Context, primaryAddress string) (types.PlayerInventory){

     playerAcc, _ := sdk.AccAddressFromBech32(primaryAddress)
     storage := k.bankKeeper.SpendableCoin(ctx, playerAcc, "ualpha")

     return types.PlayerInventory{Rocks: storage}

}