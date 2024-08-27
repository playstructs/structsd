package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"

    storetypes "cosmossdk.io/store/types"

	"context"
    "math"
    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
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

// AppendPlayer appends a player in the store with a new id and update the count
func (k Keeper) AppendPlayer(
	ctx context.Context,
	player types.Player,
) types.Player {
	ctxSDK := sdk.UnwrapSDKContext(ctx)

	// Create the player
	count := k.GetPlayerCount(ctx)
    player.Index = count

	// Set the ID of the appended value
	player.Id = GetObjectID(types.ObjectType_player, player.Index)

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	appendedValue := k.cdc.MustMarshal(&player)
	store.Set([]byte(player.Id), appendedValue)

	// Update player count
	k.SetPlayerCount(ctx, player.Index + 1)

	//Add Address records
	k.SetPlayerIndexForAddress(ctx, player.Creator, player.Index)

	addressPermissionId := GetAddressPermissionIDBytes(player.Creator)
	k.PermissionAdd(ctx, addressPermissionId, types.PermissionAll)

    // Add the initial Player Load
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id), types.PlayerPassiveDraw)

    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPlayer{Player: &player})

	return player
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
func (k Keeper) GetPlayer(ctx context.Context, playerId string, full bool) (val types.Player, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	b := store.Get([]byte(playerId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

    if (full) {
        val.Load      = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, val.Id))
        val.Capacity  = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, val.Id))

        val.StructsLoad           = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, val.Id))
        val.CapacitySecondary    = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, val.SubstationId))

    	playerAcc, _ := sdk.AccAddressFromBech32(val.PrimaryAddress)
    	val.Storage   = k.bankKeeper.SpendableCoin(ctx, playerAcc, "alpha")
    }

	return val, true
}

func (k Keeper) GetPlayerFromIndex(ctx context.Context, playerIndex uint64, full bool) (val types.Player, found bool) {
    val, found = k.GetPlayer(ctx, GetObjectID(types.ObjectType_player, playerIndex), full)
    return
}

// RemovePlayer removes a player from the store
func (k Keeper) RemovePlayer(ctx context.Context, playerId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	store.Delete([]byte(playerId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ ObjectId: playerId })
}

// GetAllPlayer returns all player
func (k Keeper) GetAllPlayer(ctx context.Context, full bool) (list []types.Player) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Player
		k.cdc.MustUnmarshal(iterator.Value(), &val)


        if (full) {
            val.Load      = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, val.Id))
            val.Capacity  = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, val.Id))

            val.StructsLoad          = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, val.Id))
            val.CapacitySecondary    = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, val.SubstationId))

            playerAcc, _ := sdk.AccAddressFromBech32(val.PrimaryAddress)
            val.Storage   = k.bankKeeper.SpendableCoin(ctx, playerAcc, "alpha")
        }

		list = append(list, val)
	}

	return
}

// GetAllPlayer returns all player
func (k Keeper) GetAllPlayerBySubstation(ctx context.Context, substationId string, full bool) (list []types.Player) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Player
		k.cdc.MustUnmarshal(iterator.Value(), &val)

        if (val.SubstationId == substationId) {

            if (full) {
                val.Load      = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, val.Id))
                val.Capacity  = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, val.Id))

                val.StructsLoad           = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, val.Id))
                val.CapacitySecondary    = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, val.SubstationId))


                playerAcc, _ := sdk.AccAddressFromBech32(val.PrimaryAddress)
                val.Storage   = k.bankKeeper.SpendableCoin(ctx, playerAcc, "alpha")
            }

            list = append(list, val)
		}
	}

	return
}

// Technically more of an InGet than an UpSert
func (k Keeper) UpsertPlayer(ctx context.Context, playerAddress string, full bool) (player types.Player) {
    playerIndex := k.GetPlayerIndexFromAddress(ctx, playerAddress)

    if (playerIndex == 0) {
        // No Player Found, Creating..
        player.Creator = playerAddress
        player.PrimaryAddress = playerAddress

        player = k.AppendPlayer(ctx, player)

    } else {
        player, _ = k.GetPlayerFromIndex(ctx, playerIndex, full)
    }

    return player
}



func (k Keeper) GetPlayerCharge(ctx context.Context, playerId string) (charge uint64) {
    ctxSDK := sdk.UnwrapSDKContext(ctx)

    // Volts*(1-power(exp(1),-(BlockSpan/(Resistance*Capacitance))))
    // Volts = 100000000
    // Resistance = 100
    // Capacitor (capacitance) = 10

    lastActionBlock := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, playerId))
    blockSpan := uint64(ctxSDK.BlockHeight()) - lastActionBlock

    // TODO / FIX
    // NGL, these floats freak me out a bit. Not sure if they'll be a source of consensus failures down the road
    result := types.Charge_Volts * (1 - math.Pow(math.Exp(1), -(float64(blockSpan)/(types.Charge_Resistance*types.Charge_Capacitance))))
    charge = uint64(result)

	return
}


func (k Keeper) DischargePlayer(ctx context.Context, playerId string) {
    ctxSDK := sdk.UnwrapSDKContext(ctx)
    k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, playerId), uint64(ctxSDK.BlockHeight()))
}




type PlayerCache struct {
    PlayerId string
    K *Keeper
    Ctx context.Context

    PlayerLoaded  bool
    PlayerChanged bool
    Player        types.Player

    StorageLoaded bool
    Storage       sdk.Coins

    NonceAttributeId string
    NonceLoaded     bool
    NonceChanged    bool
    Nonce           int64

    LoadAttributeId string
    LoadLoaded      bool
    LoadChanged     bool
    Load            uint64

    CapacityAttributeId string
    CapacityLoaded      bool
    CapacityChanged     bool
    Capacity            uint64

    StructsLoadAttributeId string
    StructsLoadLoaded      bool
    StructsLoadChanged     bool
    StructsLoad            uint64

    CapacitySecondaryAttributeId string
    CapacitySecondaryLoaded      bool
    CapacitySecondaryChanged     bool
    CapacitySecondary            uint64

}


func (k *Keeper) GetPlayerCacheFromId(ctx context.Context, playerId string) (PlayerCache, error) {
    return PlayerCache{
        PlayerId: playerId,
        K: k,
        Ctx: ctx,

        NonceAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_nonce, playerId),

        LoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_load, playerId),
        CapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, playerId),

        StructsLoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, playerId),
    }, nil
}

func (k *Keeper) GetPlayerCacheFromIndex(ctx context.Context, index uint64) (PlayerCache, error) {
    return k.GetPlayerCacheFromId(ctx, GetObjectID(types.ObjectType_player, index))
}

func (k *Keeper) GetPlayerCacheFromAddress(ctx context.Context, address string) (PlayerCache, error) {
    index := k.GetPlayerIndexFromAddress(ctx, address)

    if (index > 0) {
        return PlayerCache{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Player Account Not Found")
    }

    return k.GetPlayerCacheFromId(ctx, GetObjectID(types.ObjectType_player, index))
}

func (cache *PlayerCache) Commit() () {
    if (cache.PlayerChanged) { cache.K.SetPlayer(cache.Ctx, cache.Player) }

    if (cache.NonceChanged) { cache.K.SetGridAttributeIncrement(cache.Ctx, cache.NonceAttributeId, uint64(cache.Nonce)) }
}


func (cache *PlayerCache) LoadNonce() {
    cache.Nonce = int64(cache.K.GetGridAttribute(cache.Ctx, cache.NonceAttributeId))
    cache.NonceLoaded = true
}


func (cache *PlayerCache) LoadCapacity() {
    cache.Capacity = cache.K.GetGridAttribute(cache.Ctx, cache.CapacityAttributeId)
    cache.CapacityLoaded = true
}


func (cache *PlayerCache) LoadLoad() {
    cache.Load = cache.K.GetGridAttribute(cache.Ctx, cache.LoadAttributeId)
    cache.LoadLoaded = true
}


func (cache *PlayerCache) LoadPlayer() (found bool) {
    cache.Player, found = cache.K.GetPlayer(cache.Ctx, cache.PlayerId, true)

    if (found) {
        cache.PlayerLoaded = true
    }

    return found
}


func (cache *PlayerCache) LoadStorage() (error){
    if (!cache.PlayerLoaded) {
        return nil // TODO update to be an error
    }
    playerAcc, _ := sdk.AccAddressFromBech32(cache.Player.PrimaryAddress)
    cache.Storage = cache.K.bankKeeper.SpendableCoins(cache.Ctx, playerAcc)

    return nil
}

func (cache *PlayerCache) LoadStructsLoad() {
    cache.StructsLoad = cache.K.GetGridAttribute(cache.Ctx, cache.StructsLoadAttributeId)
    cache.StructsLoadLoaded = true
}



func (cache *PlayerCache) GetPlayer() (types.Player, error) {
    if (!cache.PlayerLoaded) {
        found := cache.LoadPlayer()
        if (!found) {
           return types.Player{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Could not load Player object for %s", cache.PlayerId )
        }
    }

    return cache.Player, nil
}





func (cache *PlayerCache) GetSubstationId() (string) {
    if (!cache.PlayerLoaded) { cache.LoadPlayer() }
    return cache.Player.SubstationId
}

func (cache *PlayerCache) GetNextNonce() (int64) {
    if (!cache.NonceLoaded) { cache.LoadNonce() }

    cache.Nonce = cache.Nonce + 1
    cache.NonceChanged = true
    return cache.Nonce
}


func (cache *PlayerCache) GetLoad() (uint64) {
    if (!cache.LoadLoaded) { cache.LoadLoad() }
    return cache.Load
}


func (cache *PlayerCache) GetStructsLoad() (uint64) {
    if (!cache.StructsLoadLoaded) { cache.LoadStructsLoad() }
    return cache.StructsLoad
}

func (cache *PlayerCache) GetCapacity() (uint64) {
    if (!cache.CapacityLoaded) { cache.LoadCapacity() }
    return cache.Capacity
}

func (cache *PlayerCache) LoadCapacitySecondary() {
    cache.CapacitySecondaryAttributeId = GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, cache.GetSubstationId())

    cache.CapacitySecondary = cache.K.GetGridAttribute(cache.Ctx, cache.CapacitySecondaryAttributeId)
    cache.CapacitySecondaryLoaded = true
}

func (cache *PlayerCache) GetCapacitySecondary() (uint64) {
    if (!cache.CapacitySecondaryLoaded) { cache.LoadCapacitySecondary() }
    return cache.CapacitySecondary
}

func (cache *PlayerCache) IsOnline() (online bool){
    if ((cache.GetLoad() + cache.GetStructsLoad()) <= (cache.GetCapacity() + cache.GetCapacitySecondary())) {
        online = true
    } else {
        online = false
    }
    return
}

func (cache *PlayerCache) IsOffline() (bool){
    return !cache.IsOnline()
}
