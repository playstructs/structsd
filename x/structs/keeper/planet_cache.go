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

    BlockRaiderArrivedAttributeId string
    BlockStartOreMineAttributeId string
    BlockStartOreRefineAttributeId string
    OreMiningActiveQuantityAttributeId string
    OreRefiningActiveQuantityAttributeId string
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

func (cache *PlanetCache) GetBlockRaiderArrived() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.BlockRaiderArrivedAttributeId)
}

func (cache *PlanetCache) GetBlockStartOreMine() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.BlockStartOreMineAttributeId)
}

func (cache *PlanetCache) GetBlockStartOreRefine() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.BlockStartOreRefineAttributeId)
}

func (cache *PlanetCache) GetOreMiningActiveQuantity() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.OreMiningActiveQuantityAttributeId)
}

func (cache *PlanetCache) GetOreRefiningActiveQuantity() (uint64) {
    return cache.CC.GetPlanetAttribute(cache.OreRefiningActiveQuantityAttributeId)
}

func (cache *PlanetCache) GetPlanetId() string {
    return cache.PlanetId
}

func (cache *PlanetCache) GetLocationListStart() string {
    return cache.GetPlanet().LocationListStart
}

// IsDefenderCommandStructVulnerable reports whether the planet owner's
// Command Ship is absent, offline, destroyed, or non-existent. The planet
// raid hashing puzzle can only be completed while this is true.
//
// The Command Ship is fleet-bound and travels with the fleet, so the
// defending fleet being on station is a complete proxy for "the Command
// Ship is home." A defender who moves their fleet away (to raid elsewhere)
// leaves their home planet undefended even with an online Command Ship.
//
// Ordering matters here: PlayerCache.GetFleet() falls back to creating a
// fleet (and a pre-built, online Command Ship) when the player has no
// FleetId yet, so the empty-FleetId check must happen before any fleet
// load to keep this a pure read.
func (cache *PlanetCache) IsDefenderCommandStructVulnerable() bool {
    if (cache.GetOwnerId() == "") {
        return true
    }

    defender := cache.GetOwner()
    if (defender.GetFleetId() == "") {
        return true
    }

    defenderFleet, defenderFleetError := cache.CC.GetFleetById(defender.GetFleetId())
    if (defenderFleetError != nil) {
        return true
    }

    // Command Ship only defends the home planet while the fleet is on station
    if (!defenderFleet.IsOnStation()) {
        return true
    }

    if (!defenderFleet.HasCommandStruct()) {
        return true
    }

    commandStruct := defenderFleet.GetCommandStruct()
    if (commandStruct.IsDestroyed()) {
        return true
    }

    return !commandStruct.IsOnline()
}

// RefreshRaidVulnerability recomputes the shieldsVulnerable state of an
// in-progress raid on this planet and updates the vulnerability clock,
// emitting a raid status event only on a transition. A running clock is
// preserved while the planet stays vulnerable, so repeated calls (e.g. a
// defender hopping between enemy planets) neither restart the puzzle nor
// spam events.
func (cache *PlanetCache) RefreshRaidVulnerability() {
    raidFleetId := cache.GetLocationListStart()
    if (raidFleetId == "") {
        // No raid in progress, nothing to update
        return
    }

    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    if (cache.IsDefenderCommandStructVulnerable()) {
        // Transition into vulnerable: start the clock and announce it
        if (cache.GetBlockStartRaid() == 0) {
            cache.ResetBlockStartRaid()
            _ = uctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: raidFleetId, PlanetId: cache.GetPlanetId(), Status: types.RaidStatus_shieldsVulnerable}})
        }
    } else {
        // Transition out of vulnerable: shields restored, stop the clock
        if (cache.GetBlockStartRaid() != 0) {
            cache.ClearBlockStartRaid()
            _ = uctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: raidFleetId, PlanetId: cache.GetPlanetId(), Status: types.RaidStatus_ongoing}})
        }
    }
}

func (cache *PlanetCache) GetLocationListLast() string {
    return cache.GetPlanet().LocationListLast
}

func (cache *PlanetCache) GetLocationListExtra() uint64 {
    return cache.GetPlanet().LocationListExtra
}

func (cache *PlanetCache) GetLocationListCount() uint64 {
    return cache.GetPlanet().LocationListCount
}

// GetLocationListCapacity is the max visiting fleets allowed in the raid
// queue. The base spot is always implied; LocationListExtra adds more.
func (cache *PlanetCache) GetLocationListCapacity() uint64 {
    return uint64(1) + cache.GetLocationListExtra()
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

    previousStart := cache.Planet.LocationListStart

    cache.Planet.LocationListStart = fleetId
    cache.Changed = true

    if (fleetId == "") {
        // Raid is over: shift ore clocks by the paused duration, then clear
        // both the raider-arrived marker and the vulnerability clock.
        cache.PauseOreClocksForRaid()
        cache.ClearBlockRaiderArrived()
        cache.ClearBlockStartRaid()
        return
    }

    // First raider to arrive anchors the mining/refining pause window.
    // A promotion (front fleet replaced mid-raid) leaves the marker alone.
    if (previousStart == "") {
        cache.ResetBlockRaiderArrived()
    }

    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    _ = uctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: fleetId, PlanetId: cache.GetPlanetId(), Status: types.RaidStatus_initiated}})

    if (cache.IsDefenderCommandStructVulnerable()) {
        // The raid clock starts at the later of raider arrival and the
        // defending Command Ship going down. A promotion (front fleet
        // replaced mid-raid) inherits the already-running clock.
        if (previousStart == "" || cache.GetBlockStartRaid() == 0) {
            cache.ResetBlockStartRaid()
        }
        _ = uctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: fleetId, PlanetId: cache.GetPlanetId(), Status: types.RaidStatus_shieldsVulnerable}})
    } else {
        // Shields are up; the hashing puzzle cannot be won until the
        // defending Command Ship goes offline (which restarts the clock).
        cache.ClearBlockStartRaid()
    }
}

func (cache *PlanetCache) SetLocationListLast(fleetId string) {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }

    cache.Planet.LocationListLast = fleetId
    cache.Changed = true
}

func (cache *PlanetCache) SetLocationListExtra(extra uint64) {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }

    cache.Planet.LocationListExtra = extra
    cache.Changed = true
}

func (cache *PlanetCache) SetLocationListCount(count uint64) {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }

    cache.Planet.LocationListCount = count
    cache.Changed = true
}

func (cache *PlanetCache) IncrementLocationListCount() {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }

    cache.Planet.LocationListCount++
    cache.Changed = true
}

func (cache *PlanetCache) DecrementLocationListCount() {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }

    if cache.Planet.LocationListCount > 0 {
        cache.Planet.LocationListCount--
        cache.Changed = true
    }
}

func (cache *PlanetCache) ResetBlockStartRaid() {
    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    cache.CC.SetPlanetAttribute(cache.BlockStartRaidAttributeId, uint64(uctx.BlockHeight()))
}

func (cache *PlanetCache) ClearBlockStartRaid() {
    cache.CC.ClearPlanetAttribute(cache.BlockStartRaidAttributeId)
}

func (cache *PlanetCache) ResetBlockRaiderArrived() {
    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    cache.CC.SetPlanetAttribute(cache.BlockRaiderArrivedAttributeId, uint64(uctx.BlockHeight()))
}

func (cache *PlanetCache) ClearBlockRaiderArrived() {
    cache.CC.ClearPlanetAttribute(cache.BlockRaiderArrivedAttributeId)
}

func (cache *PlanetCache) ResetBlockStartOreMine() {
    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    cache.CC.SetPlanetAttribute(cache.BlockStartOreMineAttributeId, uint64(uctx.BlockHeight()))
}

func (cache *PlanetCache) ClearBlockStartOreMine() {
    cache.CC.ClearPlanetAttribute(cache.BlockStartOreMineAttributeId)
}

func (cache *PlanetCache) ResetBlockStartOreRefine() {
    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    cache.CC.SetPlanetAttribute(cache.BlockStartOreRefineAttributeId, uint64(uctx.BlockHeight()))
}

func (cache *PlanetCache) ClearBlockStartOreRefine() {
    cache.CC.ClearPlanetAttribute(cache.BlockStartOreRefineAttributeId)
}

// OreMiningActivate records that a mining rig came online. The shared
// planet clock is only re-anchored when the first rig activates (0 -> 1);
// additional online rigs leave the accrued age alone.
func (cache *PlanetCache) OreMiningActivate() {
    if cache.GetOreMiningActiveQuantity() == 0 {
        cache.ResetBlockStartOreMine()
    }
    cache.CC.SetPlanetAttributeIncrement(cache.OreMiningActiveQuantityAttributeId, 1)
}

// OreMiningDeactivate records that a mining rig went offline. The clock
// is left alone: oreMiningActiveQuantity is the authoritative on/off
// signal, so a stale clock with a zero counter is harmless.
func (cache *PlanetCache) OreMiningDeactivate() {
    cache.CC.SetPlanetAttributeDecrement(cache.OreMiningActiveQuantityAttributeId, 1)
}

// OreRefiningActivate mirrors OreMiningActivate for refineries.
func (cache *PlanetCache) OreRefiningActivate() {
    if cache.GetOreRefiningActiveQuantity() == 0 {
        cache.ResetBlockStartOreRefine()
    }
    cache.CC.SetPlanetAttributeIncrement(cache.OreRefiningActiveQuantityAttributeId, 1)
}

// OreRefiningDeactivate mirrors OreMiningDeactivate for refineries.
func (cache *PlanetCache) OreRefiningDeactivate() {
    cache.CC.SetPlanetAttributeDecrement(cache.OreRefiningActiveQuantityAttributeId, 1)
}

// shiftOreClockForRaid advances an ore clock past the window a raid held it
// paused. The pause starts at whichever came later, the raid or the clock
// itself, so a clock stamped before the raid keeps its pre-raid age while a
// clock anchored mid-raid lands at age zero. A clock already at or beyond the
// raid end is returned untouched, which keeps the subtraction underflow-free.
func shiftOreClockForRaid(clock uint64, raiderArrived uint64, raidEnd uint64) uint64 {
    pauseStart := clock
    if raiderArrived > pauseStart {
        pauseStart = raiderArrived
    }
    if pauseStart >= raidEnd {
        return clock
    }
    return clock + (raidEnd - pauseStart)
}

// PauseOreClocksForRaid shifts the planet's ore mine/refine clocks forward
// by the duration the raid paused them, so difficulty age is preserved
// across the raid window. No-op when no raider-arrived marker is set
// (e.g. migration miss or empty call).
func (cache *PlanetCache) PauseOreClocksForRaid() {
    raiderArrived := cache.GetBlockRaiderArrived()
    if raiderArrived == 0 {
        return
    }

    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    raidEnd := uint64(uctx.BlockHeight())

    if cache.GetOreMiningActiveQuantity() != 0 {
        shifted := shiftOreClockForRaid(cache.GetBlockStartOreMine(), raiderArrived, raidEnd)
        cache.CC.SetPlanetAttribute(cache.BlockStartOreMineAttributeId, shifted)
    }

    if cache.GetOreRefiningActiveQuantity() != 0 {
        shifted := shiftOreClockForRaid(cache.GetBlockStartOreRefine(), raiderArrived, raidEnd)
        cache.CC.SetPlanetAttribute(cache.BlockStartOreRefineAttributeId, shifted)
    }
}

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
    } else {
        // Clear the success rate when no interceptor networks remain
        // Without this, stale success rate values linger and the planet
        // continues to benefit from interception after all sources are destroyed.
        cache.CC.SetPlanetAttribute(cache.LowOrbitBallisticsInterceptorNetworkSuccessRateNumeratorAttributeId, 0)
        cache.CC.SetPlanetAttribute(cache.LowOrbitBallisticsInterceptorNetworkSuccessRateDenominatorAttributeId, 0)
    }
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

    randomnessCheck := ((randomnessOrb.Intn(max-min+1) + min) <= int(successRate.Numerator()))

    cache.CC.k.logger.Info("Planetary Success-Check Randomness", "planetId", cache.GetPlanetId(), "seed", seed, "offset", cache.GetOwner().GetNextNonce(), "seedOffset", seedOffset, "numerator", successRate.Numerator(), "denominator", successRate.Denominator(), "success", randomnessCheck)
	return randomnessCheck
}

func (cache *PlanetCache) BuildInitiateReadiness(structure *types.Struct, structType *StructTypeCache, ambit types.Ambit, ambitSlot uint64) (error) {
    if structure.GetOwner() != cache.GetOwnerId() {
         return types.NewStructOwnershipError(structure.Id, cache.GetOwnerId(), structure.GetOwner()).WithLocation("planet", cache.GetPlanetId())
    }

    if structType.GetStructType().Type == types.CommandStruct {
        return types.NewStructLocationError(structType.GetStructType().Id, ambit.String(), "command_struct_fleet_only")
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

    if (structType.GetStructType().Category != types.ObjectType_planet) {
        return types.NewStructLocationError(structType.ID(), ambit.String(), "outside_planet")
    }

    // Check that the Struct can exist in the specified ambit
    if types.Ambit_flag[ambit]&structType.GetStructType().PossibleAmbit == 0 {
        return types.NewStructLocationError(structType.ID(), ambit.String(), "invalid_ambit")
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
            return types.NewStructBuildError(structType.ID(), "planet", cache.GetPlanetId(), "invalid_ambit").WithAmbit(ambit.String())
    }

    if (ambitSlot >= slots) {
        return types.NewStructBuildError(structType.ID(), "planet", cache.GetPlanetId(), "slot_unavailable").WithSlot(ambitSlot)
    }
    if (slot != "") {
        return types.NewStructBuildError(structType.ID(), "planet", cache.GetPlanetId(), "slot_occupied").WithSlot(ambitSlot).WithExistingStruct(slot)
    }

    return nil
}



func (cache *PlanetCache) MoveReadiness(structure *StructCache, ambit types.Ambit, ambitSlot uint64) (error) {
    if structure.GetOwnerId() != cache.GetOwnerId() {
         return types.NewStructOwnershipError(structure.GetStructId(), cache.GetOwnerId(), structure.GetOwnerId()).WithLocation("planet", cache.GetPlanetId())
    }

    if structure.GetStructType().Type == types.CommandStruct {
        return types.NewStructLocationError(structure.GetTypeId(), ambit.String(), "command_struct_fleet_only")
    }

    if (structure.GetStructType().Category != types.ObjectType_planet) {
        return types.NewStructLocationError(structure.GetTypeId(), ambit.String(), "outside_planet")
    }

    // Check that the Struct can exist in the specified ambit
    if types.Ambit_flag[ambit]&structure.GetStructType().PossibleAmbit == 0 {
        return types.NewStructLocationError(structure.GetTypeId(), ambit.String(), "invalid_ambit")
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
            return types.NewStructBuildError(structure.GetTypeId(), "planet", cache.GetPlanetId(), "invalid_ambit").WithAmbit(ambit.String())
    }

    if (ambitSlot >= slots) {
        return types.NewStructBuildError(structure.GetTypeId(), "planet", cache.GetPlanetId(), "slot_unavailable").WithSlot(ambitSlot)
    }
    if (slot != "") {
        return types.NewStructBuildError(structure.GetTypeId(), "planet", cache.GetPlanetId(), "slot_occupied").WithSlot(ambitSlot).WithExistingStruct(slot)
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
                planetStruct.DestroyAndCommit()
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

func (cache *PlanetCache) CanAllocateAsSourceBy(activePlayer *PlayerCache) error {
    return types.NewAllocationError(cache.ID(), "unacceptable_source")
}

func (cache *PlanetCache) GetName() string {
    if !cache.PlanetLoaded { cache.LoadPlanet() }
    return cache.Planet.Name
}

func (cache *PlanetCache) SetName(name string) {
    if !cache.PlanetLoaded { cache.LoadPlanet() }
    cache.Planet.Name = name
    cache.Changed = true
}

func (cache *PlanetCache) CanUpdateUGCBy(activePlayer *PlayerCache) error {
    return cache.CC.UGCPermissionCheck(cache, activePlayer)
}