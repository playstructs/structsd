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

    Ready bool

    PlayerLoaded  bool
    PlayerChanged bool
    Player        types.Player

    PlanetLoaded bool
    Planet *PlanetCache

    FleetLoaded bool
    Fleet *FleetCache

    StorageLoaded bool
    Storage       sdk.Coins

    NonceAttributeId string
    NonceLoaded     bool
    NonceChanged    bool
    Nonce           int64

    LastActionAttributeId   string
    LastActionLoaded        bool
    LastActionChanged       bool
    LastAction              uint64

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

        LastActionAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, playerId),

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

    if (cache.NonceChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.NonceAttributeId, uint64(cache.Nonce))
        cache.NonceChanged = false
    }

    if (cache.LastActionChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.LastActionAttributeId, cache.LastAction)
        cache.LastActionChanged = false
    }

    if (cache.StoredOreChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.StoredOreAttributeId, cache.StoredOre)
        cache.StoredOreChanged = false
    }

    if (cache.StructsLoadChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.StructsLoadAttributeId, cache.StructsLoad)
        cache.StructsLoadChanged = false
    }

}


func (cache *PlayerCache) LoadNonce() {
    cache.Nonce = int64(cache.K.GetGridAttribute(cache.Ctx, cache.NonceAttributeId))
    cache.NonceLoaded = true
}

func (cache *PlayerCache) LoadLastAction() {
    cache.LastAction = cache.K.GetGridAttribute(cache.Ctx, cache.LastActionAttributeId)
    cache.LastActionLoaded = true
}

func (cache *PlayerCache) LoadCapacity() {
    cache.Capacity = cache.K.GetGridAttribute(cache.Ctx, cache.CapacityAttributeId)
    cache.CapacityLoaded = true
}

func (cache *PlayerCache) LoadCapacitySecondary() {
    cache.CapacitySecondaryAttributeId = GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, cache.GetSubstationId())

    cache.CapacitySecondary = cache.K.GetGridAttribute(cache.Ctx, cache.CapacitySecondaryAttributeId)
    cache.CapacitySecondaryLoaded = true
}

func (cache *PlayerCache) LoadLoad() {
    cache.Load = cache.K.GetGridAttribute(cache.Ctx, cache.LoadAttributeId)
    cache.LoadLoaded = true
}


func (cache *PlayerCache) LoadPlayer() (found bool) {
    cache.Player, found = cache.K.GetPlayer(cache.Ctx, cache.PlayerId)

    if (found) {
        cache.PlayerLoaded = true
    }

    return found
}

// Load the Planet data
func (cache *PlayerCache) LoadPlanet() (bool) {
    if (cache.HasPlanet()) {
        newPlanet := cache.K.GetPlanetCacheFromId(cache.Ctx, cache.GetPlanetId())
        cache.Planet = &newPlanet
        cache.PlanetLoaded = true
    }
    return cache.PlanetLoaded
}

// Load the Planet data
func (cache *PlayerCache) LoadFleet() (bool) {
    newFleet, _ := cache.K.GetFleetCacheFromId(cache.Ctx, cache.GetFleetId())
    cache.Fleet = &newFleet
    cache.FleetLoaded = true

    return cache.FleetLoaded
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


func (cache *PlayerCache) GetPlayerId()         (string) { return cache.PlayerId }
func (cache *PlayerCache) GetPrimaryAddress()   (string) { if (!cache.PlayerLoaded) { cache.LoadPlayer() }; return cache.Player.PrimaryAddress }
func (cache *PlayerCache) GetSubstationId()     (string) { if (!cache.PlayerLoaded) { cache.LoadPlayer() }; return cache.Player.SubstationId }

func (cache *PlayerCache) GetFleet()    (*FleetCache)   { if (!cache.FleetLoaded) { cache.LoadFleet() }; return cache.Fleet }
func (cache *PlayerCache) GetFleetId()  (string)        { if (!cache.PlayerLoaded) { cache.LoadPlayer() }; return cache.Player.FleetId }

func (cache *PlayerCache) GetPlanet()   (*PlanetCache)  { if (!cache.PlanetLoaded) { cache.LoadPlanet() }; return cache.Planet }
func (cache *PlayerCache) GetPlanetId() (string)        { if (!cache.PlayerLoaded) { cache.LoadPlayer() }; return cache.Player.PlanetId }

func (cache *PlayerCache) GetStoredOre()            (uint64) { if (!cache.StoredOreLoaded) { cache.LoadStoredOre() }; return cache.StoredOre }
func (cache *PlayerCache) GetLoad()                 (uint64) { if (!cache.LoadLoaded) { cache.LoadLoad() }; return cache.Load }
func (cache *PlayerCache) GetStructsLoad()          (uint64) { if (!cache.StructsLoadLoaded) { cache.LoadStructsLoad() }; return cache.StructsLoad }
func (cache *PlayerCache) GetCapacity()             (uint64) { if (!cache.CapacityLoaded) { cache.LoadCapacity() }; return cache.Capacity }
func (cache *PlayerCache) GetCapacitySecondary()    (uint64) { if (!cache.CapacitySecondaryLoaded) { cache.LoadCapacitySecondary() }; return cache.CapacitySecondary }
func (cache *PlayerCache) GetLastAction()           (uint64) { if (!cache.LastActionLoaded) { cache.LoadLastAction() }; return cache.LastAction }
func (cache *PlayerCache) GetCharge()               (uint64) { ctxSDK := sdk.UnwrapSDKContext(cache.Ctx); return uint64(ctxSDK.BlockHeight()) - cache.GetLastAction() }
func (cache *PlayerCache) GetAllocatableCapacity()  (uint64) {return cache.GetCapacity() - cache.GetLoad()}

func (cache *PlayerCache) GetAvailableCapacity() (uint64) {
    if (cache.GetLoad() + cache.GetStructsLoad()) > (cache.GetCapacity() + cache.GetCapacitySecondary()) {
        return 0
    } else {
        return (cache.GetCapacity() + cache.GetCapacitySecondary()) - (cache.GetLoad() + cache.GetStructsLoad())
    }
}


func (cache *PlayerCache) GetNextNonce() (int64) {
    if (!cache.NonceLoaded) { cache.LoadNonce() }

    cache.Nonce = cache.Nonce + 1
    cache.NonceChanged = true
    return cache.Nonce
}

func (cache *PlayerCache) GetBuiltQuantity(structTypeId uint64) (uint64) {
    return cache.K.GetStructAttribute(cache.Ctx, GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, cache.GetPlayerId(), structTypeId))
}

func (cache *PlayerCache) BuildQuantityIncrement(structTypeId uint64) {
    cache.K.SetStructAttributeIncrement(cache.Ctx, GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, cache.GetPlayerId(), structTypeId), uint64(1))
}

func (cache *PlayerCache) BuildQuantityDecrement(structTypeId uint64) {
    cache.K.SetStructAttributeDecrement(cache.Ctx, GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, cache.GetPlayerId(), structTypeId), uint64(1))
}


func (cache *PlayerCache) StoredOreEmpty() {
    cache.StoredOre = 0
    cache.StoredOreChanged = true
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


func (cache *PlayerCache) StructsLoadDecrement(amount uint64) {

    if (cache.GetStructsLoad() > amount) {
        cache.StructsLoad = cache.StructsLoad - amount
    } else {
        cache.StructsLoad = 0
    }

    cache.StructsLoadChanged = true
}

func (cache *PlayerCache) StructsLoadIncrement(amount uint64) {
    cache.StructsLoad = cache.GetStructsLoad() + amount
    cache.StructsLoadChanged = true
}


func (cache *PlayerCache) Discharge() {
    ctxSDK := sdk.UnwrapSDKContext(cache.Ctx)
    cache.LastAction = uint64(ctxSDK.BlockHeight())
    cache.LastActionChanged = true
    cache.LastActionLoaded = true
}

func (cache *PlayerCache) SetPlanetId(planetId string) {
    cache.Player.PlanetId = planetId
    cache.PlayerChanged = true
}

// DepositRefinedAlpha() - Immediately Commits
// Turn this into a delayed commit like the rest
func (cache *PlayerCache) DepositRefinedAlpha() {
    // Got this far, let's reward the player with some fresh Alpha
    // Mint the new Alpha to the module
    newAlpha, _ := sdk.ParseCoinsNormalized("1alpha")
    cache.K.bankKeeper.MintCoins(cache.Ctx, types.ModuleName, newAlpha)
    // Transfer the refined Alpha to the player
    playerAcc, _ := sdk.AccAddressFromBech32(cache.GetPrimaryAddress())
    cache.K.bankKeeper.SendCoinsFromModuleToAccount(cache.Ctx, types.ModuleName, playerAcc, newAlpha)
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

func (cache *PlayerCache) HasPlanet() (bool){
    return (cache.GetPlanetId() != "")
}

func (cache *PlayerCache) HasStoredOre() (bool) {
    return (cache.GetStoredOre() > 0)
}

/* Permissions */
func (cache *PlayerCache) CanBePlayedBy(address string) (err error) {

    // Make sure the address calling this has Play permissions
    if (!cache.K.PermissionHasOneOf(cache.Ctx, GetAddressPermissionIDBytes(address), types.PermissionPlay)) {
        err = sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", address)

    } else {
        if (cache.GetPrimaryAddress() != address) {
            callingPlayer, err := cache.K.GetPlayerCacheFromAddress(cache.Ctx, address)
            if (err != nil) {
                if (callingPlayer.GetPlayerId() != cache.GetPlayerId()) {
                    if (!cache.K.PermissionHasOneOf(cache.Ctx, GetObjectPermissionIDBytes(cache.GetPlayerId(), callingPlayer.GetPlayerId()), types.PermissionPlay)) {
                       err = sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling account (%s) has no play permissions on target player (%s)", callingPlayer.GetPlayerId(), cache.GetPlayerId())
                    }
                }
            }
        }
    }

    return
}

func (cache *PlayerCache) ReadinessCheck() (err error) {
    if (cache.IsOffline()) {
        err = sdkerrors.Wrapf(types.ErrGridMalfunction, "Player (%s) is offline. Activate it", cache.PlayerId)
    }
    cache.Ready = true
    return
}


func (cache *PlayerCache) AttemptPlanetExplore() (err error) {
    newPlanetId := cache.K.AppendPlanet(cache.Ctx, cache.Player)
    cache.SetPlanetId(newPlanetId)
    cache.PlanetLoaded = false

    // TODO move fleet to new planet (if it's not elsewhere I guess?)

    return nil
}

func (cache *PlayerCache) CanSupportLoadAddition(additionalLoad uint64) (bool) {
    // Check player Load for the passiveDraw capacity
    totalLoad := cache.GetLoad() + cache.GetStructsLoad()
    totalCapacity := cache.GetCapacity() + cache.GetCapacitySecondary()

    if (totalLoad > totalCapacity) {
        return false
    }

    return ((totalCapacity - totalLoad) >= additionalLoad)
}