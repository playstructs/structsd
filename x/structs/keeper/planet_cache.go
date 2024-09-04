package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
    //sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	"github.com/nethruster/go-fraction"


    // Used in Randomness Orb
	"math/rand"
    "bytes"
    "encoding/binary"

)


type PlanetCache struct {
    PlanetId string
    K *Keeper
    Ctx context.Context

    Ready bool

    PlanetLoaded  bool
    PlanetChanged bool
    Planet  types.Planet

    OwnerLoaded bool
    Owner *PlayerCache

    BuriedOreAttributeId string
    BuriedOreLoaded bool
    BuriedOreChanged bool
    BuriedOre  uint64

    PlanetaryShieldAttributeId string
    PlanetaryShieldLoaded bool
    PlanetaryShieldChanged bool
    PlanetaryShield  uint64

    RepairNetworkQuantityAttributeId string
    RepairNetworkQuantityLoaded bool
    RepairNetworkQuantityChanged bool
    RepairNetworkQuantity uint64

    DefensiveCannonQuantityAttributeId string
    DefensiveCannonQuantityLoaded bool
    DefensiveCannonQuantityChanged bool
    DefensiveCannonQuantity uint64

    CoordinatedGlobalShieldNetworkQuantityAttributeId string
    CoordinatedGlobalShieldNetworkQuantityLoaded bool
    CoordinatedGlobalShieldNetworkQuantityChanged bool
    CoordinatedGlobalShieldNetworkQuantity uint64

    LowOrbitBallisticsInterceptorNetworkQuantityAttributeId string
    LowOrbitBallisticsInterceptorNetworkQuantityLoaded bool
    LowOrbitBallisticsInterceptorNetworkQuantityChanged bool
    LowOrbitBallisticsInterceptorNetworkQuantity uint64

    AdvancedLowOrbitBallisticsInterceptorNetworkQuantityAttributeId string
    AdvancedLowOrbitBallisticsInterceptorNetworkQuantityLoaded bool
    AdvancedLowOrbitBallisticsInterceptorNetworkQuantityChanged bool
    AdvancedLowOrbitBallisticsInterceptorNetworkQuantity uint64

    OrbitalJammingStationQuantityAttributeId string
    OrbitalJammingStationQuantityLoaded bool
    OrbitalJammingStationQuantityChanged bool
    OrbitalJammingStationQuantity uint64

    AdvancedOrbitalJammingStationQuantityAttributeId string
    AdvancedOrbitalJammingStationQuantityLoaded bool
    AdvancedOrbitalJammingStationQuantityChanged bool
    AdvancedOrbitalJammingStationQuantity uint64


    // Event Tracking
    EventAttackDetailLoaded bool
    EventAttackDetail *types.EventAttackDetail

    EventAttackShotDetailLoaded bool
    EventAttackShotDetail *types.EventAttackShotDetail

}

// Build this initial Struct Cache object
// This does no validation on the provided structId
func (k *Keeper) GetPlanetCacheFromId(ctx context.Context, planetId string) (PlanetCache) {
    return PlanetCache{
        PlanetId: planetId,
        K: k,
        Ctx: ctx,

        BuriedOreAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_ore, planetId),

        PlanetaryShieldAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetaryShield, planetId),
        RepairNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_repairNetworkQuantity, planetId),
        DefensiveCannonQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_defensiveCannonQuantity, planetId),

        CoordinatedGlobalShieldNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_coordinatedGlobalShieldNetworkQuantity, planetId),
        LowOrbitBallisticsInterceptorNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkQuantity, planetId),

        AdvancedLowOrbitBallisticsInterceptorNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_advancedLowOrbitBallisticsInterceptorNetworkQuantity, planetId),

        OrbitalJammingStationQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_orbitalJammingStationQuantity, planetId),
        AdvancedOrbitalJammingStationQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_advancedOrbitalJammingStationQuantity, planetId),


    }
}

func (cache *PlanetCache) Commit() () {

    if (cache.PlanetChanged) {
        cache.K.SetPlanet(cache.Ctx, cache.Planet)
        cache.PlanetChanged = false
    }

    if (cache.BuriedOreChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.BuriedOreAttributeId, cache.BuriedOre)
        cache.BuriedOreChanged = false
    }


    if (cache.PlanetaryShieldChanged) {
        cache.K.SetPlanetAttribute(cache.Ctx, cache.PlanetaryShieldAttributeId, cache.PlanetaryShield)
        cache.PlanetaryShieldChanged = false
    }

    if (cache.RepairNetworkQuantityChanged) {
        cache.K.SetPlanetAttribute(cache.Ctx, cache.RepairNetworkQuantityAttributeId, cache.RepairNetworkQuantity)
        cache.RepairNetworkQuantityChanged = false
    }

    if (cache.DefensiveCannonQuantityChanged) {
        cache.K.SetPlanetAttribute(cache.Ctx, cache.DefensiveCannonQuantityAttributeId, cache.DefensiveCannonQuantity)
        cache.DefensiveCannonQuantityChanged = false
    }

    if (cache.CoordinatedGlobalShieldNetworkQuantityChanged) {
        cache.K.SetPlanetAttribute(cache.Ctx, cache.CoordinatedGlobalShieldNetworkQuantityAttributeId, cache.CoordinatedGlobalShieldNetworkQuantity)
        cache.CoordinatedGlobalShieldNetworkQuantityChanged = false
    }

    if (cache.LowOrbitBallisticsInterceptorNetworkQuantityChanged) {
        cache.K.SetPlanetAttribute(cache.Ctx, cache.LowOrbitBallisticsInterceptorNetworkQuantityAttributeId, cache.LowOrbitBallisticsInterceptorNetworkQuantity)
        cache.LowOrbitBallisticsInterceptorNetworkQuantityChanged = false
    }

    if (cache.AdvancedLowOrbitBallisticsInterceptorNetworkQuantityChanged) {
        cache.K.SetPlanetAttribute(cache.Ctx, cache.AdvancedLowOrbitBallisticsInterceptorNetworkQuantityAttributeId, cache.AdvancedLowOrbitBallisticsInterceptorNetworkQuantity)
        cache.AdvancedLowOrbitBallisticsInterceptorNetworkQuantityChanged = false
    }

    if (cache.OrbitalJammingStationQuantityChanged) {
        cache.K.SetPlanetAttribute(cache.Ctx, cache.OrbitalJammingStationQuantityAttributeId, cache.OrbitalJammingStationQuantity)
        cache.OrbitalJammingStationQuantityChanged = false
    }

    if (cache.AdvancedOrbitalJammingStationQuantityChanged) {
        cache.K.SetPlanetAttribute(cache.Ctx, cache.AdvancedOrbitalJammingStationQuantityAttributeId, cache.AdvancedOrbitalJammingStationQuantity)
        cache.AdvancedOrbitalJammingStationQuantityChanged = false
    }

}

/* Separate Loading functions for each of the underlying containers */

// Load the core Planet data
func (cache *PlanetCache) LoadPlanet() (bool) {
    cache.Planet, cache.PlanetLoaded = cache.K.GetPlanet(cache.Ctx, cache.PlanetId)
    return cache.PlanetLoaded
}


// Load the Player data
func (cache *PlanetCache) LoadOwner() (bool) {
    newOwner, _ := cache.K.GetPlayerCacheFromId(cache.Ctx, cache.GetOwnerId())
    cache.Owner = &newOwner
    cache.OwnerLoaded = true
    return cache.OwnerLoaded
}

// Load the BuriedOre record
func (cache *PlanetCache) LoadBuriedOre() {
    cache.BuriedOre = cache.K.GetGridAttribute(cache.Ctx, cache.BuriedOreAttributeId)
    cache.BuriedOreLoaded = true
}

// Load the Planet PlanetaryShield record
func (cache *PlanetCache) LoadPlanetaryShield() {
    cache.PlanetaryShield = cache.K.GetPlanetAttribute(cache.Ctx, cache.PlanetaryShieldAttributeId)
    cache.PlanetaryShieldLoaded = true
}

// Load the Planet RepairNetworkQuantity record
func (cache *PlanetCache) LoadRepairNetworkQuantity() {
    cache.RepairNetworkQuantity = cache.K.GetPlanetAttribute(cache.Ctx, cache.RepairNetworkQuantityAttributeId)
    cache.RepairNetworkQuantityLoaded = true
}

// Load the Planet DefensiveCannonQuantity record
func (cache *PlanetCache) LoadDefensiveCannonQuantity() {
    cache.DefensiveCannonQuantity = cache.K.GetPlanetAttribute(cache.Ctx, cache.DefensiveCannonQuantityAttributeId)
    cache.DefensiveCannonQuantityLoaded = true
}

// Load the Planet CoordinatedGlobalShieldNetworkQuantity record
func (cache *PlanetCache) LoadCoordinatedGlobalShieldNetworkQuantity() {
    cache.CoordinatedGlobalShieldNetworkQuantity = cache.K.GetPlanetAttribute(cache.Ctx, cache.CoordinatedGlobalShieldNetworkQuantityAttributeId)
    cache.CoordinatedGlobalShieldNetworkQuantityLoaded = true
}

// Load the Planet LowOrbitBallisticsInterceptorNetworkQuantity record
func (cache *PlanetCache) LoadLowOrbitBallisticsInterceptorNetworkQuantity() {
    cache.LowOrbitBallisticsInterceptorNetworkQuantity = cache.K.GetPlanetAttribute(cache.Ctx, cache.LowOrbitBallisticsInterceptorNetworkQuantityAttributeId)
    cache.LowOrbitBallisticsInterceptorNetworkQuantityLoaded = true
}

// Load the Planet AdvancedLowOrbitBallisticsInterceptorNetworkQuantity record
func (cache *PlanetCache) LoadAdvancedLowOrbitBallisticsInterceptorNetworkQuantity() {
    cache.AdvancedLowOrbitBallisticsInterceptorNetworkQuantity = cache.K.GetPlanetAttribute(cache.Ctx, cache.AdvancedLowOrbitBallisticsInterceptorNetworkQuantityAttributeId)
    cache.AdvancedLowOrbitBallisticsInterceptorNetworkQuantityLoaded = true
}

// Load the Planet OrbitalJammingStationQuantity record
func (cache *PlanetCache) LoadOrbitalJammingStationQuantity() {
    cache.OrbitalJammingStationQuantity = cache.K.GetPlanetAttribute(cache.Ctx, cache.OrbitalJammingStationQuantityAttributeId)
    cache.OrbitalJammingStationQuantityLoaded = true
}

// Load the Planet AdvancedOrbitalJammingStationQuantity record
func (cache *PlanetCache) LoadAdvancedOrbitalJammingStationQuantity() {
    cache.AdvancedOrbitalJammingStationQuantity = cache.K.GetPlanetAttribute(cache.Ctx, cache.AdvancedOrbitalJammingStationQuantityAttributeId)
    cache.AdvancedOrbitalJammingStationQuantityLoaded = true
}


/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

// Get the Owner ID data
func (cache *PlanetCache) GetOwnerId() (string) {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }
    return cache.Planet.Owner
}

// Get the Owner data
func (cache *PlanetCache) GetOwner() (*PlayerCache) {
    if (!cache.OwnerLoaded) { cache.LoadOwner() }
    return cache.Owner
}

func (cache *PlanetCache) GetPlanet() (types.Planet) {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }
    return cache.Planet
}

func (cache *PlanetCache) GetBuriedOre() (uint64) {
    if (!cache.BuriedOreLoaded) { cache.LoadBuriedOre() }
    return cache.BuriedOre
}

func (cache *PlanetCache) GetPlanetaryShield() (uint64) {
    if (!cache.PlanetaryShieldLoaded) { cache.LoadPlanetaryShield() }
    return cache.PlanetaryShield
}

func (cache *PlanetCache) GetRepairNetworkQuantity() (uint64) {
    if (!cache.RepairNetworkQuantityLoaded) { cache.LoadRepairNetworkQuantity() }
    return cache.RepairNetworkQuantity
}

func (cache *PlanetCache) GetDefensiveCannonQuantity() (uint64) {
    if (!cache.DefensiveCannonQuantityLoaded) { cache.LoadDefensiveCannonQuantity() }
    return cache.DefensiveCannonQuantity
}

func (cache *PlanetCache) GetCoordinatedGlobalShieldNetworkQuantity() (uint64) {
    if (!cache.CoordinatedGlobalShieldNetworkQuantityLoaded) { cache.LoadCoordinatedGlobalShieldNetworkQuantity() }
    return cache.CoordinatedGlobalShieldNetworkQuantity
}

func (cache *PlanetCache) GetLowOrbitBallisticsInterceptorNetworkQuantity() (uint64) {
    if (!cache.LowOrbitBallisticsInterceptorNetworkQuantityLoaded) { cache.LoadLowOrbitBallisticsInterceptorNetworkQuantity() }
    return cache.LowOrbitBallisticsInterceptorNetworkQuantity
}

func (cache *PlanetCache) GetAdvancedLowOrbitBallisticsInterceptorNetworkQuantity() (uint64) {
    if (!cache.AdvancedLowOrbitBallisticsInterceptorNetworkQuantityLoaded) { cache.LoadAdvancedLowOrbitBallisticsInterceptorNetworkQuantity() }
    return cache.AdvancedLowOrbitBallisticsInterceptorNetworkQuantity
}

func (cache *PlanetCache) GetOrbitalJammingStationQuantity() (uint64) {
    if (!cache.OrbitalJammingStationQuantityLoaded) { cache.LoadOrbitalJammingStationQuantity() }
    return cache.OrbitalJammingStationQuantity
}

func (cache *PlanetCache) GetAdvancedOrbitalJammingStationQuantity() (uint64) {
    if (!cache.AdvancedOrbitalJammingStationQuantityLoaded) { cache.LoadAdvancedOrbitalJammingStationQuantity() }
    return cache.AdvancedOrbitalJammingStationQuantity
}


func (cache *PlanetCache) GetEventAttackDetail() (*types.EventAttackDetail) {
    if (!cache.EventAttackDetailLoaded) { cache.EventAttackDetail = types.CreateEventAttackDetail() }
    return cache.EventAttackDetail
}


func (cache *PlanetCache) GetEventAttackShotDetail() (*types.EventAttackShotDetail) {
    return cache.EventAttackShotDetail
}

func (cache *PlanetCache) FlushEventAttackShotDetail() ( *types.EventAttackShotDetail) {
    cache.EventAttackShotDetailLoaded = false
    return cache.EventAttackShotDetail
}

/* Setters - SET DOES NOT COMMIT()
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

// Get the Owner ID data
func (cache *PlanetCache) SetStatus(status types.PlanetStatus) () {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }

    cache.Planet.Status = types.PlanetStatus
    cache.PlanetChanged = true
}


// Set the Owner data manually
// Useful for loading multiple defenders
func (cache *PlanetCache) ManualLoadOwner(owner *PlayerCache) {
    cache.Owner = owner
    cache.OwnerLoaded = true
}

// Set the Event data manually
// Used to manage the same event across objects
func (cache *PlanetCache) ManualLoadEventAttackDetail(eventAttackDetail *types.EventAttackDetail) {
    cache.EventAttackDetail = eventAttackDetail
    cache.EventAttackDetailLoaded = true
}
func (cache *PlanetCache) ManualLoadEventAttackShotDetail(eventAttackShotDetail *types.EventAttackShotDetail) {
    cache.EventAttackShotDetail = eventAttackShotDetail
    cache.EventAttackShotDetailLoaded = true
}


/* Flag Commands for the Status field */

func (cache *PlanetCache) IsComplete() bool {
   return (cache.GetPlanet().Status == types.PlanetStatus_complete)
}

func (cache *PlanetCache) IsActive() bool {
   return (cache.GetPlanet().Status == types.PlanetStatus_active)
}


/* Rough but Consistent Randomness Check */
func (cache *PlanetCache) IsSuccessful(successRate fraction.Fraction) bool {
    uctx := sdk.UnwrapSDKContext(cache.Ctx)

	var seed int64

	buf := bytes.NewBuffer(uctx.BlockHeader().LastCommitHash)
	binary.Read(buf, binary.BigEndian, &seed)

    seed = seed + cache.GetOwner().GetNextNonce()

	randomnessOrb := rand.New(rand.NewSource(seed))
	min := 1
	max := int(successRate.Denominator())

	return (int(successRate.Numerator()) <= (randomnessOrb.Intn(max-min+1) + min))
}


/* Game Logic */

func (cache *PlanetCache) AttemptComplete() (bool) {
    if (cache.GetBuriedOre() > 0) {
        return false
    }

    cache.SetStatus(types.PlanetStatus_complete)
    return true

}
