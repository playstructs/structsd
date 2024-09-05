package keeper

import (

	"context"
    //"math"
    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"

)

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

    StoredOreAttributeId string
    StoredOreLoaded      bool
    StoredOreChanged     bool
    StoredOre            uint64

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

        StoredOreAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_ore, playerId),
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

    if (cache.StoredOreChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.StoredOreAttributeId, cache.StoredOre)
        cache.StoredOreChanged = false
    }

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

func (cache *PlayerCache) LoadStoredOre() {
    cache.StoredOre = cache.K.GetGridAttribute(cache.Ctx, cache.StoredOreAttributeId)
    cache.StoredOreLoaded = true
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

func (cache *PlayerCache) GetStoredOre() (uint64) {
    if (!cache.StoredOreLoaded) { cache.LoadStoredOre() }
    return cache.StoredOre
}

func (cache *PlayerCache) StoredOreDecrement(amount uint64) {

    if (cache.GetStoredOre() > amount) {
        cache.StoredOre = cache.StoredOre - amount
    } else {
        cache.StoredOre = 0
    }

    cache.StoredOreChanged = true
}

func (cache *PlayerCache) StoredOreIncrement(amount uint64) {
    cache.StoredOre = cache.GetStoredOre() + amount
    cache.StoredOreChanged = true
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
