package keeper

import (
	//"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	"github.com/nethruster/go-fraction"

    // Used in Randomness Orb
	"math/rand"
    "bytes"
    "encoding/binary"


)


type PlanetCache struct {
    PlanetId string
    CC  *CurrentContext

    Changed bool

    Ready bool

    PlanetLoaded  bool
    Planet  types.Planet


    BlockStartRaidAttributeId string
    BuriedOreAttributeId string
    PlanetaryShieldAttributeId string
    RepairNetworkQuantityAttributeId string
    DefensiveCannonQuantityAttributeId string
    CoordinatedGlobalShieldNetworkQuantityAttributeId string
    LowOrbitBallisticsInterceptorNetworkQuantityAttributeId string
    AdvancedLowOrbitBallisticsInterceptorNetworkQuantityAttributeId string
    LowOrbitBallisticsInterceptorNetworkSuccessRateNumeratorAttributeId string
    LowOrbitBallisticsInterceptorNetworkSuccessRateDenominatorAttributeId string
    OrbitalJammingStationQuantityAttributeId string
    AdvancedOrbitalJammingStationQuantityAttributeId string

    // Event Tracking
    EventAttackDetailLoaded bool
    EventAttackDetail *types.EventAttackDetail

    EventAttackShotDetailLoaded bool
    EventAttackShotDetail *types.EventAttackShotDetail

}


func (cache *PlanetCache) Commit() () {
    if (cache.Changed) {
        cache.CC.k.logger.Info("Updating Planet From Cache","planetId",cache.PlanetId)
        cache.CC.k.SetPlanet(cache.CC.ctx, cache.Planet)
    }
    cache.Changed = false
}

func (cache *PlanetCache) IsChanged() bool {
    return cache.Changed
}

func (cache *PlanetCache) ID() string {
    return cache.PlanetId
}


/* Separate Loading functions for each of the underlying containers */

// Load the core Planet data
func (cache *PlanetCache) LoadPlanet() (bool) {
    cache.Planet, cache.PlanetLoaded = cache.CC.k.GetPlanet(cache.CC.ctx, cache.PlanetId)
    return cache.PlanetLoaded
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
func (cache *PlanetCache) GetOwner() (player *PlayerCache) {
    player, _ = cache.CC.GetPlayer(cache.GetOwnerId())
    return
}

func (cache *PlanetCache) GetPlanet() (types.Planet) {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }
    return cache.Planet
}

func (cache *PlanetCache) GetBlockStartRaid() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.BlockStartRaidAttributeId)
}

func (cache *PlanetCache) GetBuriedOre() (uint64) {
    return cache.CC.GetGridAttribute(cache.BuriedOreAttributeId)
}

func (cache *PlanetCache) GetPlanetaryShield() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.PlanetaryShieldAttributeId)
}

func (cache *PlanetCache) GetRepairNetworkQuantity() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.RepairNetworkQuantityAttributeId)
}

func (cache *PlanetCache) GetDefensiveCannonQuantity() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.DefensiveCannonQuantityAttributeId)
}

func (cache *PlanetCache) GetCoordinatedGlobalShieldNetworkQuantity() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.CoordinatedGlobalShieldNetworkQuantityAttributeId)
}

func (cache *PlanetCache) GetLowOrbitBallisticsInterceptorNetworkQuantity() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.LowOrbitBallisticsInterceptorNetworkQuantityAttributeId)
}

func (cache *PlanetCache) GetAdvancedLowOrbitBallisticsInterceptorNetworkQuantity() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.AdvancedLowOrbitBallisticsInterceptorNetworkQuantityAttributeId)
}

func (cache *PlanetCache) GetLowOrbitBallisticsInterceptorNetworkSuccessRate() (successRate fraction.Fraction, err error) {
    successRate, err = fraction.New(cache.CC.GetPlanetAttribute(cache.LowOrbitBallisticsInterceptorNetworkSuccessRateNumeratorAttributeId), cache.CC.GetPlanetAttribute(cache.LowOrbitBallisticsInterceptorNetworkSuccessRateDenominatorAttributeId))
    return
}

func (cache *PlanetCache) GetOrbitalJammingStationQuantity() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.OrbitalJammingStationQuantityAttributeId)
}

func (cache *PlanetCache) GetAdvancedOrbitalJammingStationQuantity() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.AdvancedOrbitalJammingStationQuantityAttributeId)
}

func (cache *PlanetCache) GetPlanetId() string {
    return cache.PlanetId
}

func (cache *PlanetCache) GetLocationListStart() string {
    return cache.GetPlanet().LocationListStart
}

func (cache *PlanetCache) GetLocationListLast() string {
    return cache.GetPlanet().LocationListLast
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
    cache.Changed = true
}

func (cache *PlanetCache) SetLocationListStart(fleetId string) {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }

    cache.Planet.LocationListStart = fleetId
    cache.Changed = true

    if (fleetId != "") {
        uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
        _ = uctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: fleetId, PlanetId: cache.GetPlanetId(), Status: types.RaidStatus_initiated}})
        cache.ResetBlockStartRaid()
    }

}

func (cache *PlanetCache) SetLocationListLast(fleetId string) {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }

    cache.Planet.LocationListLast = fleetId
    cache.Changed = true
}

func (cache *PlanetCache) ResetBlockStartRaid() {
    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    cache.CC.SetPlanetAttribute(cache.BlockStartRaidAttributeId, uint64(uctx.BlockHeight()))
}

// TODO START UPDATING THESE TO POINT TO CC GRID
func (cache *PlanetCache) BuriedOreDecrement(amount uint64) {
    cache.CC.SetGridAttributeDecrement(cache.BuriedOreAttributeId, amount)
}

func (cache *PlanetCache) PlanetaryShieldIncrement(amount uint64) {
    cache.CC.SetPlanetAttributeIncrement(cache.PlanetaryShieldAttributeId, amount)
}

func (cache *PlanetCache) PlanetaryShieldDecrement(amount uint64) {
    cache.CC.SetPlanetAttributeDecrement(cache.PlanetaryShieldAttributeId, amount)
}

func (cache *PlanetCache) DefensiveCannonQuantityIncrement(amount uint64) {
    cache.CC.SetPlanetAttributeIncrement(cache.DefensiveCannonQuantityAttributeId, amount)
}

func (cache *PlanetCache) DefensiveCannonQuantityDecrement(amount uint64) {
    cache.CC.SetPlanetAttributeDecrement(cache.DefensiveCannonQuantityAttributeId, amount)
}


func (cache *PlanetCache) LowOrbitBallisticsInterceptorNetworkQuantityIncrement(amount uint64) {
    cache.CC.SetPlanetAttributeIncrement(cache.LowOrbitBallisticsInterceptorNetworkQuantityAttributeId, amount)
    cache.LowOrbitBallisticsInterceptorNetworkRecalculate()
}

func (cache *PlanetCache) LowOrbitBallisticsInterceptorNetworkQuantityDecrement(amount uint64) {
    cache.CC.SetPlanetAttributeDecrement(cache.LowOrbitBallisticsInterceptorNetworkQuantityAttributeId, amount)
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

        cache.CC.SetPlanetAttribute(cache.LowOrbitBallisticsInterceptorNetworkSuccessRateNumeratorAttributeId, uint64(overallSuccessRate.Numerator()))
        cache.CC.SetPlanetAttribute(cache.LowOrbitBallisticsInterceptorNetworkSuccessRateDenominatorAttributeId, uint64(overallSuccessRate.Denominator()))
    }
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
    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)

	var seed int64

	buf := bytes.NewBuffer(uctx.BlockHeader().AppHash)
	binary.Read(buf, binary.BigEndian, &seed)

    seedOffset := seed + cache.GetOwner().GetNextNonce()
	randomnessOrb := rand.New(rand.NewSource(seedOffset))
	min := 1
	max := int(successRate.Denominator())

    randomnessCheck := (int(successRate.Numerator()) <= (randomnessOrb.Intn(max-min+1) + min))

    cache.CC.k.logger.Info("Planetary Success-Check Randomness", "planetId", cache.GetPlanetId(), "seed", seed, "offset", cache.GetOwner().GetNextNonce(), "seedOffset", seedOffset, "numerator", successRate.Numerator(), "denominator", successRate.Denominator(), "success", randomnessCheck)
	return randomnessCheck
}

func (cache *PlanetCache) BuildInitiateReadiness(structure *types.Struct, structType *types.StructType, ambit types.Ambit, ambitSlot uint64) (error) {
    if structure.GetOwner() != cache.GetOwnerId() {
         return types.NewStructOwnershipError(structure.Id, cache.GetOwnerId(), structure.GetOwner()).WithLocation("planet", cache.GetPlanetId())
    }

    if structType.Type == types.CommandStruct {
        return types.NewStructLocationError(structType.GetId(), ambit.String(), "command_struct_fleet_only")
    }

    if cache.GetOwner().GetFleet().IsAway() {
        return types.NewFleetStateError(cache.GetOwner().GetFleetId(), "away", "build")
    }

    if !cache.GetOwner().GetFleet().HasCommandStruct() {
        return types.NewFleetCommandError(cache.GetOwner().GetFleetId(), "no_command_struct")
    }

    if cache.GetOwner().GetFleet().GetCommandStruct().IsOffline() {
        return types.NewFleetCommandError(cache.GetOwner().GetFleetId(), "command_offline")
    }

    if (structType.Category != types.ObjectType_planet) {
        return types.NewStructLocationError(structType.GetId(), ambit.String(), "outside_planet")
    }

    // Check that the Struct can exist in the specified ambit
    if types.Ambit_flag[ambit]&structType.PossibleAmbit == 0 {
        return types.NewStructLocationError(structType.GetId(), ambit.String(), "invalid_ambit")
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
            return types.NewStructBuildError(structType.GetId(), "planet", cache.GetPlanetId(), "invalid_ambit").WithAmbit(ambit.String())
    }

    if (ambitSlot >= slots) {
        return types.NewStructBuildError(structType.GetId(), "planet", cache.GetPlanetId(), "slot_unavailable").WithSlot(ambitSlot)
    }
    if (slot != "") {
        return types.NewStructBuildError(structType.GetId(), "planet", cache.GetPlanetId(), "slot_occupied").WithSlot(ambitSlot).WithExistingStruct(slot)
    }

    return nil
}



func (cache *PlanetCache) MoveReadiness(structure *StructCache, ambit types.Ambit, ambitSlot uint64) (error) {
    if structure.GetOwnerId() != cache.GetOwnerId() {
         return types.NewStructOwnershipError(structure.GetStructId(), cache.GetOwnerId(), structure.GetOwnerId()).WithLocation("planet", cache.GetPlanetId())
    }

    if structure.GetStructType().Type == types.CommandStruct {
        return types.NewStructLocationError(structure.GetStructType().GetId(), ambit.String(), "command_struct_fleet_only")
    }

    if (structure.GetStructType().Category != types.ObjectType_planet) {
        return types.NewStructLocationError(structure.GetStructType().GetId(), ambit.String(), "outside_planet")
    }

    // Check that the Struct can exist in the specified ambit
    if types.Ambit_flag[ambit]&structure.GetStructType().PossibleAmbit == 0 {
        return types.NewStructLocationError(structure.GetStructType().GetId(), ambit.String(), "invalid_ambit")
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
            return types.NewStructBuildError(structure.GetStructType().GetId(), "planet", cache.GetPlanetId(), "invalid_ambit").WithAmbit(ambit.String())
    }

    if (ambitSlot >= slots) {
        return types.NewStructBuildError(structure.GetStructType().GetId(), "planet", cache.GetPlanetId(), "slot_unavailable").WithSlot(ambitSlot)
    }
    if (slot != "") {
        return types.NewStructBuildError(structure.GetStructType().GetId(), "planet", cache.GetPlanetId(), "slot_occupied").WithSlot(ambitSlot).WithExistingStruct(slot)
    }

    return nil
}


func (cache *PlanetCache) SetSlot(structure types.Struct) (err error) {

    cache.CC.k.logger.Info("Planet Slot Update","planetId", cache.GetPlanetId(), "slot", structure.Slot, "ambit", structure.OperatingAmbit)

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
            err = types.NewStructLocationError(0, structure.OperatingAmbit.String(), "invalid_ambit").WithStruct(structure.Id)
    }

    cache.Changed = true
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

    cache.Changed = true
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
                planetStruct := cache.CC.GetStruct(structId)
                planetStruct.Destroy()
            }
        }

        // Send Fleets away
        for cache.GetLocationListStart() != "" {
               currentFleet, _ := cache.CC.GetFleetById(cache.GetLocationListStart())
               currentFleet.PeaceDeal()
        }

        return nil
    }
    return types.NewPlanetStateError(cache.GetPlanetId(), "has_ore", "explore")
}

func (cache *PlanetCache) AttemptDefenseCannon(attacker *StructCache) (cannoned bool) {
    if (cache.GetDefensiveCannonQuantity() > 0) {
        attacker.TakePlanetaryDefenseCanonDamage(cache.GetDefensiveCannonQuantity())
    }
    return
}