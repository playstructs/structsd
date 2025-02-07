package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
    sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	"github.com/nethruster/go-fraction"


    // Used in Randomness Orb
	"math/rand"
    "bytes"
    "encoding/binary"
    "fmt"

)


type PlanetCache struct {
    PlanetId string
    K *Keeper
    Ctx context.Context

    AnyChange bool

    Ready bool

    PlanetLoaded  bool
    PlanetChanged bool
    Planet  types.Planet

    OwnerLoaded bool
    Owner *PlayerCache

    BlockStartRaidAttributeId string
    BlockStartRaidLoaded bool
    BlockStartRaidChanged bool
    BlockStartRaid  uint64

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

    LowOrbitBallisticsInterceptorNetworkSuccessRateNumeratorAttributeId string
    LowOrbitBallisticsInterceptorNetworkSuccessRateDenominatorAttributeId string
    LowOrbitBallisticsInterceptorNetworkSuccessRateLoaded bool
    LowOrbitBallisticsInterceptorNetworkSuccessRateChanged bool
    LowOrbitBallisticsInterceptorNetworkSuccessRateNumerator uint64
    LowOrbitBallisticsInterceptorNetworkSuccessRateDenominator uint64

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

        AnyChange: false,

        PlanetLoaded: false,
        PlanetChanged: false,
        OwnerLoaded: false,
        BlockStartRaidLoaded: false,
        BlockStartRaidChanged: false,
        BuriedOreLoaded: false,
        BuriedOreChanged: false,
        PlanetaryShieldLoaded: false,
        PlanetaryShieldChanged: false,
        RepairNetworkQuantityLoaded: false,
        RepairNetworkQuantityChanged: false,
        DefensiveCannonQuantityLoaded: false,
        DefensiveCannonQuantityChanged: false,
        CoordinatedGlobalShieldNetworkQuantityLoaded: false,
        CoordinatedGlobalShieldNetworkQuantityChanged: false,
        LowOrbitBallisticsInterceptorNetworkQuantityLoaded: false,
        LowOrbitBallisticsInterceptorNetworkQuantityChanged: false,
        AdvancedLowOrbitBallisticsInterceptorNetworkQuantityLoaded: false,
        AdvancedLowOrbitBallisticsInterceptorNetworkQuantityChanged: false,
        LowOrbitBallisticsInterceptorNetworkSuccessRateLoaded: false,
        LowOrbitBallisticsInterceptorNetworkSuccessRateChanged: false,
        OrbitalJammingStationQuantityLoaded: false,
        OrbitalJammingStationQuantityChanged: false,
        AdvancedOrbitalJammingStationQuantityLoaded: false,
        AdvancedOrbitalJammingStationQuantityChanged: false,
        EventAttackDetailLoaded: false,
        EventAttackShotDetailLoaded: false,

        BlockStartRaidAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, planetId),
        BuriedOreAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_ore, planetId),

        PlanetaryShieldAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetaryShield, planetId),
        RepairNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_repairNetworkQuantity, planetId),
        DefensiveCannonQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_defensiveCannonQuantity, planetId),

        CoordinatedGlobalShieldNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_coordinatedGlobalShieldNetworkQuantity, planetId),

        LowOrbitBallisticsInterceptorNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkQuantity, planetId),
        AdvancedLowOrbitBallisticsInterceptorNetworkQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_advancedLowOrbitBallisticsInterceptorNetworkQuantity, planetId),

        LowOrbitBallisticsInterceptorNetworkSuccessRateNumeratorAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateNumerator, planetId),
        LowOrbitBallisticsInterceptorNetworkSuccessRateDenominatorAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateDenominator, planetId),


        OrbitalJammingStationQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_orbitalJammingStationQuantity, planetId),
        AdvancedOrbitalJammingStationQuantityAttributeId: GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_advancedOrbitalJammingStationQuantity, planetId),


    }
}

func (cache *PlanetCache) Commit() () {
    cache.AnyChange = false

    fmt.Printf("\n Updating Planet From Cache (%s) \n", cache.PlanetId)

    if (cache.PlanetChanged) {
        cache.K.SetPlanet(cache.Ctx, cache.Planet)
        cache.PlanetChanged = false
    }

    if (cache.Owner != nil && cache.GetOwner().IsChanged()) {
        cache.GetOwner().Commit()
    }

    if (cache.BlockStartRaidChanged) {
        cache.K.SetPlanetAttribute(cache.Ctx, cache.BlockStartRaidAttributeId, cache.BlockStartRaid)
        cache.BlockStartRaidChanged = false
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

    if (cache.LowOrbitBallisticsInterceptorNetworkSuccessRateChanged) {
        cache.K.SetPlanetAttribute(cache.Ctx, cache.LowOrbitBallisticsInterceptorNetworkSuccessRateNumeratorAttributeId, cache.LowOrbitBallisticsInterceptorNetworkSuccessRateNumerator)
        cache.K.SetPlanetAttribute(cache.Ctx, cache.LowOrbitBallisticsInterceptorNetworkSuccessRateDenominatorAttributeId, cache.LowOrbitBallisticsInterceptorNetworkSuccessRateDenominator)
        cache.LowOrbitBallisticsInterceptorNetworkSuccessRateChanged = false
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

func (cache *PlanetCache) IsChanged() bool {
    return cache.AnyChange
}

func (cache *PlanetCache) Changed() {
    cache.AnyChange = true
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
func (cache *PlanetCache) LoadBlockStartRaid() {
    cache.BlockStartRaid = cache.K.GetPlanetAttribute(cache.Ctx, cache.BlockStartRaidAttributeId)
    cache.BlockStartRaidLoaded = true
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

// Load the Planet LowOrbitBallisticsInterceptorNetworkSuccessRate records
func (cache *PlanetCache) LoadLowOrbitBallisticsInterceptorNetworkSuccessRate() {
    cache.LowOrbitBallisticsInterceptorNetworkSuccessRateNumerator = cache.K.GetPlanetAttribute(cache.Ctx, cache.LowOrbitBallisticsInterceptorNetworkSuccessRateNumeratorAttributeId)
    cache.LowOrbitBallisticsInterceptorNetworkSuccessRateDenominator = cache.K.GetPlanetAttribute(cache.Ctx, cache.LowOrbitBallisticsInterceptorNetworkSuccessRateDenominatorAttributeId)
    cache.LowOrbitBallisticsInterceptorNetworkSuccessRateLoaded = true
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

func (cache *PlanetCache) GetBlockStartRaid() (uint64) {
    if (!cache.BlockStartRaidLoaded) { cache.LoadBlockStartRaid() }
    return cache.BlockStartRaid
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

func (cache *PlanetCache) GetLowOrbitBallisticsInterceptorNetworkSuccessRate() (successRate fraction.Fraction, err error) {
    if (!cache.LowOrbitBallisticsInterceptorNetworkSuccessRateLoaded) { cache.LoadLowOrbitBallisticsInterceptorNetworkSuccessRate() }

    successRate, err = fraction.New(cache.LowOrbitBallisticsInterceptorNetworkSuccessRateNumerator, cache.LowOrbitBallisticsInterceptorNetworkSuccessRateDenominator)
    return
}

func (cache *PlanetCache) GetOrbitalJammingStationQuantity() (uint64) {
    if (!cache.OrbitalJammingStationQuantityLoaded) { cache.LoadOrbitalJammingStationQuantity() }
    return cache.OrbitalJammingStationQuantity
}

func (cache *PlanetCache) GetAdvancedOrbitalJammingStationQuantity() (uint64) {
    if (!cache.AdvancedOrbitalJammingStationQuantityLoaded) { cache.LoadAdvancedOrbitalJammingStationQuantity() }
    return cache.AdvancedOrbitalJammingStationQuantity
}

func (cache *PlanetCache) GetPlanetId() string {
    return cache.PlanetId
}

func (cache *PlanetCache) GetLocationListStart() string {
    return cache.GetPlanet().LocationListStart
}

func (cache *PlanetCache) GetLocationListLast() string {
    return cache.GetPlanet().LocationListStart
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

func (cache *PlanetCache) SetStatus(status types.PlanetStatus) () {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }

    cache.Planet.Status = status
    cache.PlanetChanged = true
    cache.Changed()
}

func (cache *PlanetCache) SetLocationListStart(fleetId string) {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }

    cache.Planet.LocationListStart = fleetId
    cache.PlanetChanged = true
    cache.Changed()

    if (fleetId != "") {
        uctx := sdk.UnwrapSDKContext(cache.Ctx)
        _ = uctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: fleetId, PlanetId: cache.GetPlanetId(), Status: types.RaidStatus_initiated}})
        cache.ResetBlockStartRaid()
    }

}

func (cache *PlanetCache) SetLocationListLast(fleetId string) {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }

    cache.Planet.LocationListStart = fleetId
    cache.PlanetChanged = true
    cache.Changed()
}

func (cache *PlanetCache) ResetBlockStartRaid() {
    uctx := sdk.UnwrapSDKContext(cache.Ctx)
    cache.BlockStartRaid = uint64(uctx.BlockHeight())
    cache.BlockStartRaidLoaded = true
    cache.BlockStartRaidChanged = true
    cache.Changed()
}

func (cache *PlanetCache) BuriedOreDecrement(amount uint64) {

    if (cache.GetBuriedOre() > amount) {
        cache.BuriedOre = cache.BuriedOre - amount
    } else {
        cache.BuriedOre = 0
    }

    cache.BuriedOreChanged = true
    cache.Changed()
}

func (cache *PlanetCache) PlanetaryShieldIncrement(amount uint64) {
    cache.PlanetaryShield = cache.GetPlanetaryShield() + amount
    cache.PlanetaryShieldChanged = true
    cache.Changed()
}

func (cache *PlanetCache) PlanetaryShieldDecrement(amount uint64) {

    if (cache.GetPlanetaryShield() > amount) {
        cache.PlanetaryShield = cache.PlanetaryShield - amount
    } else {
        cache.PlanetaryShield = 0
    }

    cache.PlanetaryShieldChanged = true
    cache.Changed()
}

func (cache *PlanetCache) DefensiveCannonQuantityIncrement(amount uint64) {
    cache.DefensiveCannonQuantity = cache.GetDefensiveCannonQuantity() + amount
    cache.DefensiveCannonQuantityChanged = true
    cache.Changed()
}

func (cache *PlanetCache) DefensiveCannonQuantityDecrement(amount uint64) {

    if (cache.GetDefensiveCannonQuantity() > amount) {
        cache.DefensiveCannonQuantity = cache.DefensiveCannonQuantity - amount
    } else {
        cache.DefensiveCannonQuantity = 0
    }

    cache.DefensiveCannonQuantityChanged = true
    cache.Changed()
}


func (cache *PlanetCache) LowOrbitBallisticsInterceptorNetworkQuantityIncrement(amount uint64) {
    cache.LowOrbitBallisticsInterceptorNetworkQuantity = cache.GetLowOrbitBallisticsInterceptorNetworkQuantity() + amount
    cache.LowOrbitBallisticsInterceptorNetworkQuantityChanged = true
    cache.Changed()

    cache.LowOrbitBallisticsInterceptorNetworkRecalculate()
}

func (cache *PlanetCache) LowOrbitBallisticsInterceptorNetworkQuantityDecrement(amount uint64) {

    if (cache.GetLowOrbitBallisticsInterceptorNetworkQuantity() > amount) {
        cache.LowOrbitBallisticsInterceptorNetworkQuantity = cache.LowOrbitBallisticsInterceptorNetworkQuantity - amount
    } else {
        cache.LowOrbitBallisticsInterceptorNetworkQuantity = 0
    }

    cache.LowOrbitBallisticsInterceptorNetworkQuantityChanged = true
    cache.Changed()
    cache.LowOrbitBallisticsInterceptorNetworkRecalculate()
}

func (cache *PlanetCache) LowOrbitBallisticsInterceptorNetworkRecalculate() {
    if ((cache.GetLowOrbitBallisticsInterceptorNetworkQuantity() + cache.GetAdvancedLowOrbitBallisticsInterceptorNetworkQuantity()) != 0) {
        oneRate, _ := fraction.New(1,1)
        individualFailureRate, _ := fraction.New(2,3)

        overallFailureRate := individualFailureRate

        // Intentionally starts at 1, since we start by adding one above.
        for system := uint64(1); system < cache.GetLowOrbitBallisticsInterceptorNetworkQuantity(); system++ {
            overallFailureRate = overallFailureRate.Multiply(individualFailureRate)
        }

        overallSuccessRate := oneRate.Subtract(overallFailureRate)

        cache.LowOrbitBallisticsInterceptorNetworkSuccessRateNumerator = uint64(overallSuccessRate.Numerator())
        cache.LowOrbitBallisticsInterceptorNetworkSuccessRateDenominator = uint64(overallSuccessRate.Denominator())
        cache.LowOrbitBallisticsInterceptorNetworkSuccessRateChanged = true
        cache.LowOrbitBallisticsInterceptorNetworkSuccessRateLoaded  = true
        cache.Changed()

    }
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

func (cache *PlanetCache) IsEmptyOfOre() bool {
    return (cache.GetBuriedOre() == 0)
}

/* Rough but Consistent Randomness Check */
func (cache *PlanetCache) IsSuccessful(successRate fraction.Fraction) bool {
    uctx := sdk.UnwrapSDKContext(cache.Ctx)

	var seed int64

	buf := bytes.NewBuffer(uctx.BlockHeader().AppHash)
	binary.Read(buf, binary.BigEndian, &seed)
    fmt.Printf("Checking randomness using seed %d \n", seed)
    seed = seed + cache.GetOwner().GetNextNonce()
    fmt.Printf("Offsetting seed with nonce to %d \n", seed)
    fmt.Printf("Odds of %d in %d \n", successRate.Numerator(), successRate.Denominator())

	randomnessOrb := rand.New(rand.NewSource(seed))
	min := 1
	max := int(successRate.Denominator())

    fmt.Printf("Result: %t \n", (int(successRate.Numerator()) <= (randomnessOrb.Intn(max-min+1) + min)))
	return (int(successRate.Numerator()) <= (randomnessOrb.Intn(max-min+1) + min))
}

func (cache *PlanetCache) BuildInitiateReadiness(structure *types.Struct, structType *types.StructType, ambit types.Ambit, ambitSlot uint64) (error) {
    if structure.GetOwner() != cache.GetOwnerId() {
         sdkerrors.Wrapf(types.ErrStructAction, "Struct owner must match planet ")
    }

    if structType.Type == types.CommandStruct {
        sdkerrors.Wrapf(types.ErrStructAction, "Command Structs can only be built directly in the fleet")
    }

    if cache.GetOwner().GetFleet().IsAway() {
        sdkerrors.Wrapf(types.ErrStructAction, "Structs cannot be built unless Fleet is On Station")
    }

    if !cache.GetOwner().GetFleet().HasCommandStruct() {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Fleet (%s) needs a Command Struct before deploy", cache.GetOwner().GetFleetId())
    }

    if cache.GetOwner().GetFleet().GetCommandStruct().IsOffline() {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Fleet (%s) needs an Online Command Struct before deploy", cache.GetOwner().GetFleetId())
    }

    if (structType.Category != types.ObjectType_planet) {
        sdkerrors.Wrapf(types.ErrStructAction, "Struct Type cannot exist in this location (%s) ")
    }

    // Check that the Struct can exist in the specified ambit
    if types.Ambit_flag[ambit]&structType.PossibleAmbit == 0 {
        return sdkerrors.Wrapf(types.ErrStructAction, "Struct cannot be exist in the defined ambit (%s) based on structType (%d) ", ambit, structType.Id)
    }

    var slots uint64
    var slot string
    // Check Ambit / Slot
    switch ambit {
        case types.Ambit_land:
            slots = cache.GetPlanet().LandSlots
            slot  = cache.GetPlanet().Land[ambitSlot]
        case types.Ambit_water:
            slots = cache.GetPlanet().WaterSlots
            slot  = cache.GetPlanet().Water[ambitSlot]
        case types.Ambit_air:
            slots = cache.GetPlanet().AirSlots
            slot  = cache.GetPlanet().Air[ambitSlot]
        case types.Ambit_space:
            slots = cache.GetPlanet().SpaceSlots
            slot  = cache.GetPlanet().Space[ambitSlot]
        default:
            return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The Struct Build was initiated on a non-existent ambit")
    }

    if (ambitSlot >= slots) {
        return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The planet (%s) specified doesn't have that slot available to build on", cache.GetPlanetId())
    }
    if (slot != "") {
        return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The planet (%s) specified already has a struct on that slot", cache.GetPlanetId())
    }

    return nil
}



func (cache *PlanetCache) MoveReadiness(structure *StructCache, ambit types.Ambit, ambitSlot uint64) (error) {
    if structure.GetOwnerId() != cache.GetOwnerId() {
         sdkerrors.Wrapf(types.ErrStructAction, "Struct owner must match planet ")
    }

    if structure.GetStructType().Type == types.CommandStruct {
        sdkerrors.Wrapf(types.ErrStructAction, "Command Structs can only be built directly in the fleet")
    }

    if (structure.GetStructType().Category != types.ObjectType_planet) {
        sdkerrors.Wrapf(types.ErrStructAction, "Struct Type cannot exist in this location (%s) ")
    }

    // Check that the Struct can exist in the specified ambit
    if types.Ambit_flag[ambit]&structure.GetStructType().PossibleAmbit == 0 {
        return sdkerrors.Wrapf(types.ErrStructAction, "Struct cannot be exist in the defined ambit (%s) based on structType (%d) ", ambit, structure.GetStructType().Id)
    }

    var slots uint64
    var slot string
    // Check Ambit / Slot
    switch ambit {
        case types.Ambit_land:
            slots = cache.GetPlanet().LandSlots
            slot  = cache.GetPlanet().Land[ambitSlot]
        case types.Ambit_water:
            slots = cache.GetPlanet().WaterSlots
            slot  = cache.GetPlanet().Water[ambitSlot]
        case types.Ambit_air:
            slots = cache.GetPlanet().AirSlots
            slot  = cache.GetPlanet().Air[ambitSlot]
        case types.Ambit_space:
            slots = cache.GetPlanet().SpaceSlots
            slot  = cache.GetPlanet().Space[ambitSlot]
        default:
            return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The Struct Build was initiated on a non-existent ambit")
    }

    if (ambitSlot >= slots) {
        return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The planet (%s) specified doesn't have that slot available to build on", cache.GetPlanetId())
    }
    if (slot != "") {
        return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The planet (%s) specified already has a struct on that slot", cache.GetPlanetId())
    }

    return nil
}


func (cache *PlanetCache) SetSlot(structure types.Struct) (err error) {

    fmt.Printf(" Planet %s", cache.GetPlanetId())
    fmt.Printf(" Setting Slot: %d", structure.Slot)

    if (!cache.PlanetLoaded) { cache.LoadPlanet() }

    switch structure.OperatingAmbit {
        case types.Ambit_water:
            cache.Planet.Water[structure.Slot] = structure.Id
        case types.Ambit_land:
            cache.Planet.Land[structure.Slot]  = structure.Id
        case types.Ambit_air:
            cache.Planet.Air[structure.Slot]   = structure.Id
        case types.Ambit_space:
            cache.Planet.Space[structure.Slot] = structure.Id
        default:
            err = sdkerrors.Wrapf(types.ErrStructAction, "Struct cannot exist in the defined ambit (%s) ", structure.OperatingAmbit)
    }

    cache.PlanetChanged = true
    cache.Changed()
	return
}


func (cache *PlanetCache) ClearSlot(ambit types.Ambit, slot uint64) {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }

    switch ambit {
        case types.Ambit_water:
            cache.Planet.Water[slot] = ""
        case types.Ambit_land:
            cache.Planet.Land[slot]  = ""
        case types.Ambit_air:
            cache.Planet.Air[slot]   = ""
        case types.Ambit_space:
            cache.Planet.Space[slot] = ""
    }
    cache.PlanetChanged = true
    cache.Changed()
}

/* Game Logic */

// AttemptComplete
func (cache *PlanetCache) AttemptComplete() (error) {
    if (cache.IsEmptyOfOre()) {
        cache.SetStatus(types.PlanetStatus_complete)


        // Destroy Structs
        structsToDestroy := append(cache.GetPlanet().Space, cache.GetPlanet().Air...)
        structsToDestroy  = append(structsToDestroy, cache.GetPlanet().Land...)
        structsToDestroy  = append(structsToDestroy, cache.GetPlanet().Water...)

        // For Space
        for _, structId := range structsToDestroy {
            if structId != "" {
                planetStruct := cache.K.GetStructCacheFromId(cache.Ctx, structId)
                planetStruct.ManualLoadOwner(cache.GetOwner())
                planetStruct.ManualLoadPlanet(cache)
                planetStruct.DestroyAndCommit()
            }
        }

        // Send Fleets away
        for cache.GetLocationListStart() != "" {
               currentFleet, _ := cache.K.GetFleetCacheFromId(cache.Ctx, cache.GetLocationListStart())
               currentFleet.ManualLoadPlanet(cache)
               currentFleet.PeaceDeal()
        }

        cache.Commit()
        return nil
    }
    return sdkerrors.Wrapf(types.ErrPlanetExploration, "New Planet cannot be explored while current planet (%s) has Ore available for mining", cache.GetPlanetId())
}



func (cache *PlanetCache) AttemptDefenseCannon(attacker *StructCache) (cannoned bool) {
    if (cache.GetDefensiveCannonQuantity() > 0) {
        attacker.TakePlanetaryDefenseCanonDamage(cache.GetDefensiveCannonQuantity())
    }
    return
}