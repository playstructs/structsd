package keeper

import (

	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/nethruster/go-fraction"

	// Used in Randomness Orb
	"bytes"
	"encoding/binary"
	"math/rand"
)

type StructCache struct {
	StructId string
	CC       *CurrentContext

	Ready bool

	Changed bool
	StructureLoaded  bool
	Structure        types.Struct

	HealthAttributeId string
	StatusAttributeId string
//	Status            types.StructState
	BlockStartBuildAttributeId string
	BlockStartOreMineAttributeId string
	BlockStartOreRefineAttributeId string
	ProtectedStructIndexAttributeId string
	ReadyAttributeId string

	Blocker  bool
	Defender bool

	// Event Tracking
	EventAttackDetailLoaded bool
	EventAttackDetail       *types.EventAttackDetail

	EventAttackShotDetailLoaded bool
	EventAttackShotDetail       *types.EventAttackShotDetail
}


func (cache *StructCache) Commit() {
	if cache.Changed {
    	cache.CC.k.logger.Info("Updating Struct From Cache", "structId", cache.StructId)
		cache.CC.k.SetStruct(cache.CC.ctx, cache.Structure)
		cache.Changed = false
	}
}

func (cache *StructCache) IsChanged() bool {
	return cache.Changed
}

func (cache *StructCache) ID() string {
	return cache.StructId
}

/* Separate Loading functions for each of the underlying containers */

// Load the core Struct data
func (cache *StructCache) LoadStruct() bool {
	cache.Structure, cache.StructureLoaded = cache.CC.k.GetStruct(cache.CC.ctx, cache.StructId)
	return cache.StructureLoaded
}




// Set the Event data manually
// Used to manage the same event across objects
func (cache *StructCache) ManualLoadEventAttackDetail(eventAttackDetail *types.EventAttackDetail) {
	cache.EventAttackDetail = eventAttackDetail
	cache.EventAttackDetailLoaded = true
}
func (cache *StructCache) ManualLoadEventAttackShotDetail(eventAttackShotDetail *types.EventAttackShotDetail) {
	cache.EventAttackShotDetail = eventAttackShotDetail
	cache.EventAttackShotDetailLoaded = true
}

/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

func (cache *StructCache) CheckStruct() error {
	if !cache.StructureLoaded {
		if !cache.LoadStruct() {
		    return types.NewObjectNotFoundError("struct", cache.GetStructId())
		}
	}
	return nil
}

func (cache *StructCache) GetStruct() types.Struct {
	if !cache.StructureLoaded {
		cache.LoadStruct()
	}
	return cache.Structure
}


func (cache *StructCache) GetStructId() string { return cache.StructId }

func (cache *StructCache) GetHealth() uint64 {
	return cache.CC.GetStructAttribute(cache.HealthAttributeId)
}

func (cache *StructCache) GetStatus() types.StructState {
    return types.StructState(cache.CC.GetStructAttribute(cache.StatusAttributeId))
}


func (cache *StructCache) GetBlockStartBuild() uint64 {
	return cache.CC.GetStructAttribute(cache.BlockStartBuildAttributeId)
}

func (cache *StructCache) GetBlockStartOreMine() uint64 {
    return cache.CC.GetStructAttribute(cache.BlockStartOreMineAttributeId)
}

func (cache *StructCache) GetBlockStartOreRefine() uint64 {
    return cache.CC.GetStructAttribute(cache.BlockStartOreRefineAttributeId)
}

func (cache *StructCache) GetStructType() types.StructType {
    structType, _ := cache.CC.GetStructType(cache.GetTypeId())
    return structType.GetStructType()

}
func (cache *StructCache) GetTypeId() uint64 {
	if !cache.StructureLoaded {
		cache.LoadStruct()
	}
	return cache.Structure.Type
}

func (cache *StructCache) GetOwner() *PlayerCache {
	player, _ := cache.CC.GetPlayer(cache.GetOwnerId())
	return player
}

func (cache *StructCache) GetOwnerId() string {
	if !cache.StructureLoaded {
		cache.LoadStruct()
	}
	return cache.Structure.Owner
}

func (cache *StructCache) GetLocationId() string {
	if !cache.StructureLoaded {
		cache.LoadStruct()
	}
	return cache.Structure.LocationId
}
func (cache *StructCache) GetLocationType() types.ObjectType {
	if !cache.StructureLoaded {
		cache.LoadStruct()
	}
	return cache.Structure.LocationType
}
func (cache *StructCache) GetOperatingAmbit() types.Ambit {
	if !cache.StructureLoaded {
		cache.LoadStruct()
	}
	return cache.Structure.OperatingAmbit
}
func (cache *StructCache) GetSlot() uint64 {
	if !cache.StructureLoaded {
		cache.LoadStruct()
	}
	return cache.Structure.Slot
}

func (cache *StructCache) GetPlanet() (planet *PlanetCache) {
	switch cache.GetLocationType() {
	    case types.ObjectType_planet:
		    planet = cache.CC.GetPlanet(cache.GetLocationId())
    	case types.ObjectType_fleet:
	    	planet = cache.CC.GetPlanet(cache.GetFleet().GetLocationId())
	}
	return
}

func (cache *StructCache) GetPlanetId() string { return cache.GetPlanet().GetPlanetId() }

func (cache *StructCache) GetFleet() *FleetCache {
	fleet, _ := cache.CC.GetFleetById(cache.GetOwner().GetFleetId())
	return fleet
}

func (cache *StructCache) GetDefenders() []*StructCache {
	return cache.CC.GetAllStructDefender(cache.GetStructId())
}

func (cache *StructCache) GetEventAttackDetail() *types.EventAttackDetail {
	if !cache.EventAttackDetailLoaded {
		cache.EventAttackDetail = types.CreateEventAttackDetail()
		cache.EventAttackDetailLoaded = true
	}
	return cache.EventAttackDetail
}
func (cache *StructCache) GetEventAttackShotDetail() *types.EventAttackShotDetail {
	if !cache.EventAttackShotDetailLoaded {
		cache.EventAttackShotDetail = types.CreateEventAttackShotDetail(cache.StructId)
		cache.EventAttackShotDetailLoaded = true
	}
	return cache.EventAttackShotDetail
}

/* Setters - SET DOES NOT COMMIT()
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

// Set the Owner Id data
func (cache *StructCache) SetOwnerId(owner string) {
	if !cache.StructureLoaded {
		cache.LoadStruct()
	}

	cache.Structure.Owner = owner
	cache.Changed = true
}

func (cache *StructCache) ResetBlockStartOreMine() {
	uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
	cache.CC.SetStructAttribute(cache.BlockStartOreMineAttributeId, uint64(uctx.BlockHeight()))
}

func (cache *StructCache) ResetBlockStartOreRefine() {
	uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
	cache.CC.SetStructAttribute(cache.BlockStartOreRefineAttributeId, uint64(uctx.BlockHeight()))
}

func (cache *StructCache) ClearBlockStartOreMine() {
	cache.CC.SetStructAttribute(cache.BlockStartOreMineAttributeId, 0)
}

func (cache *StructCache) ClearBlockStartOreRefine() {
	cache.CC.SetStructAttribute(cache.BlockStartOreRefineAttributeId, 0)
}

func (cache *StructCache) FlushEventAttackShotDetail() *types.EventAttackShotDetail {
	cache.EventAttackShotDetailLoaded = false
	return cache.EventAttackShotDetail
}

/* Flag Commands for the Status field */

// Does the Struct exist in any State?
// This is the most efficient check that a Struct exists
func (cache *StructCache) IsMaterialized() bool {
	return cache.GetStatus()&types.StructStateMaterialized != 0
}

func (cache *StructCache) IsBuilt() bool {
	return cache.GetStatus()&types.StructStateBuilt != 0
}

func (cache *StructCache) IsOnline() bool {
	return cache.GetStatus()&types.StructStateOnline != 0
}

func (cache *StructCache) IsCommandable() bool {
	if cache.GetStructType().Category == types.ObjectType_fleet {
		if !cache.GetFleet().HasCommandStruct() {
			return false
		}

		if cache.GetFleet().GetCommandStruct().IsOffline() {
			return false
		}
	}
	return true
}

func (cache *StructCache) IsOffline() bool {
	return !cache.IsOnline()
}

func (cache *StructCache) IsHidden() bool {
	return cache.GetStatus()&types.StructStateHidden != 0
}

func (cache *StructCache) StatusAddBuilt() {
	cache.CC.SetStructAttributeFlagAdd(cache.StatusAttributeId, uint64(types.StructStateBuilt))
}

func (cache *StructCache) StatusAddOnline() {
	cache.CC.SetStructAttributeFlagAdd(cache.StatusAttributeId, uint64(types.StructStateOnline))
}

func (cache *StructCache) StatusAddHidden() {
	cache.CC.SetStructAttributeFlagAdd(cache.StatusAttributeId, uint64(types.StructStateHidden))
}

func (cache *StructCache) StatusAddDestroyed() {
    cache.CC.SetStructAttributeFlagAdd(cache.StatusAttributeId, uint64(types.StructStateDestroyed))
}

func (cache *StructCache) StatusRemoveHidden() {
	if cache.IsHidden() {
    	cache.CC.SetStructAttributeFlagRemove(cache.StatusAttributeId, uint64(types.StructStateHidden))
	}
}

func (cache *StructCache) StatusRemoveOnline() {
	if cache.IsOnline() {
	    cache.CC.SetStructAttributeFlagRemove(cache.StatusAttributeId, uint64(types.StructStateOnline))
	}
}

func (cache *StructCache) IsDestroyed() bool {
	return cache.GetStatus()&types.StructStateDestroyed != 0
}

func (cache *StructCache) GridStatusAddReady() {
	cache.CC.SetGridAttribute(cache.ReadyAttributeId, 1)
}

func (cache *StructCache) GridStatusRemoveReady() {
    cache.CC.ClearGridAttribute(cache.ReadyAttributeId)
}

func (cache *StructCache) ActivationReadinessCheck() (err error) {
	// Check Struct is Built
	if !cache.IsBuilt() {
		return types.NewStructStateError(cache.StructId, "building", "built", "activation")
	}

	// Check Struct is Online
	if cache.IsOnline() {
		return types.NewStructStateError(cache.StructId, "online", "offline", "activation")
	}

	// Check Player is Online
	if cache.GetOwner().IsOffline() {
		return types.NewPlayerPowerError(cache.GetOwnerId(), "offline")
	}

	// Check Player Capacity
	if !cache.GetOwner().CanSupportLoadAddition(cache.GetStructType().PassiveDraw) {
		return types.NewPlayerPowerError(cache.GetOwnerId(), "capacity_exceeded").WithCapacity(cache.GetStructType().PassiveDraw, cache.GetOwner().GetAvailableCapacity())
	}

	return
}

func (cache *StructCache) GoOnline() {
	// Add to the players struct load
	cache.GetOwner().StructsLoadIncrement(cache.GetStructType().PassiveDraw)
	//k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, sudoPlayer.Id), structType.PassiveDraw)

	// Turn on the mining systems
	if cache.GetStructType().HasOreMiningSystem() {
		cache.ResetBlockStartOreMine()
	}

	// Turn on the refinery
	if cache.GetStructType().HasOreRefiningSystem() {
		cache.ResetBlockStartOreRefine()
	}

	// Raise the planetary shields
	if cache.GetStructType().HasOreReserveDefensesSystem() {
		cache.GetPlanet().PlanetaryShieldIncrement(cache.GetStructType().PlanetaryShieldContribution)
	}

	// TODO
	// This is the least generic/abstracted part of the code for now.
	// Prob need to clean this up down the road
	if cache.GetStructType().HasPlanetaryDefensesSystem() {
		switch cache.GetStructType().PlanetaryDefenses {
		case types.TechPlanetaryDefenses_defensiveCannon:
			cache.GetPlanet().DefensiveCannonQuantityIncrement(1)
		case types.TechPlanetaryDefenses_lowOrbitBallisticInterceptorNetwork:
			cache.GetPlanet().LowOrbitBallisticsInterceptorNetworkQuantityIncrement(1)
		}
	}

	if cache.GetStructType().HasPowerGenerationSystem() {
		cache.GridStatusAddReady()
	}

	// Set the struct status flag to include built
	cache.StatusAddOnline()
}

func (cache *StructCache) GoOffline() {
	// Add to the players struct load
	cache.GetOwner().StructsLoadDecrement(cache.GetStructType().PassiveDraw)
	//k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, sudoPlayer.Id), structType.PassiveDraw)

	// Turn off the mining systems
	if cache.GetStructType().HasOreMiningSystem() {
		cache.ClearBlockStartOreMine()
	}

	// Turn off the refinery
	if cache.GetStructType().HasOreRefiningSystem() {
		cache.ClearBlockStartOreRefine()
	}

	// Lower the planetary shields
	if cache.GetStructType().HasOreReserveDefensesSystem() {
		cache.GetPlanet().PlanetaryShieldDecrement(cache.GetStructType().PlanetaryShieldContribution)
	}

	// TODO
	// This is the least generic/abstracted part of the code for now.
	// Prob need to clean this up down the road
	if cache.GetStructType().HasPlanetaryDefensesSystem() {
		switch cache.GetStructType().PlanetaryDefenses {
		case types.TechPlanetaryDefenses_defensiveCannon:
			cache.GetPlanet().DefensiveCannonQuantityDecrement(1)
		case types.TechPlanetaryDefenses_lowOrbitBallisticInterceptorNetwork:
			cache.GetPlanet().LowOrbitBallisticsInterceptorNetworkQuantityDecrement(1)
		}
	}

	if cache.GetStructType().HasPowerGenerationSystem() {
		cache.GridStatusRemoveReady()

		// Remove all allocations
		allocations := cache.CC.GetAllAllocationBySource(cache.StructId)
		for _, allocation := range allocations {
		    allocation.Destroy()
		}
	}

	// Set the struct status flag to include built
	cache.StatusRemoveOnline()
}

func (cache *StructCache) ReadinessCheck() error {
	if cache.IsOffline() {
		return types.NewStructStateError(cache.StructId, "offline", "online", "readiness_check")
	} else {
		if cache.GetOwner().IsOffline() {
			return types.NewPlayerPowerError(cache.GetOwnerId(), "offline")
		}
	}

	cache.Ready = true
	return nil
}

/* Rough but Consistent Randomness Check */
func (cache *StructCache) IsSuccessful(successRate fraction.Fraction) bool {
	uctx := sdk.UnwrapSDKContext(cache.CC.ctx)

	var seed int64

	buf := bytes.NewBuffer(uctx.BlockHeader().AppHash)
	binary.Read(buf, binary.BigEndian, &seed)

	seedOffset := seed + cache.GetOwner().GetNextNonce()

	randomnessOrb := rand.New(rand.NewSource(seedOffset))
	min := 1
	max := int(successRate.Denominator())

	randomnessCheck := (int(successRate.Numerator()) <= (randomnessOrb.Intn(max-min+1) + min))
	cache.CC.k.logger.Info("Struct Success-Check Randomness", "structId", cache.GetStructId(), "seed", seed, "offset", cache.GetOwner().GetNextNonce(), "seedOffset", seedOffset, "numerator", successRate.Numerator(), "denominator", successRate.Denominator(), "success", randomnessCheck)

	return randomnessCheck
}

/* Permissions */
func (cache *StructCache) CanBePlayedBy(address string) error {

	// Make sure the address calling this has Play permissions
	if !cache.CC.PermissionHasOneOf(GetAddressPermissionIDBytes(address), types.PermissionPlay) {
		return types.NewPermissionError("address", address, "", "", uint64(types.PermissionPlay), "play")
	}

	callingPlayer, err := cache.CC.GetPlayerByAddress(address)
	if err != nil {
		return err
	}
	if callingPlayer.GetPlayerId() != cache.GetOwnerId() {
		if !cache.CC.PermissionHasOneOf(GetObjectPermissionIDBytes(cache.GetOwnerId(), callingPlayer.GetPlayerId()), types.PermissionPlay) {
			return types.NewPermissionError("player", callingPlayer.GetPlayerId(), "player", cache.GetOwnerId(), uint64(types.PermissionPlay), "play")
		}
	}
	return nil
}

func (cache *StructCache) CanBeHashedBy(address string) (string, bool, error) {
	owner := true
	// Make sure the address calling this has Hash permissions
	if !cache.CC.PermissionHasOneOf(GetAddressPermissionIDBytes(address), types.PermissionHash) {
		return "", owner, types.NewPermissionError("address", address, "", "", uint64(types.PermissionHash), "hash")
	}

	callingPlayer, err := cache.CC.GetPlayerByAddress(address)
	if err != nil {
		return "", owner, err
	}
	if callingPlayer.GetPlayerId() != cache.GetOwnerId() {
		owner = false
		if !cache.CC.PermissionHasOneOf(GetObjectPermissionIDBytes(cache.GetOwnerId(), callingPlayer.GetPlayerId()), types.PermissionHash) {
			return callingPlayer.PlayerId, owner, types.NewPermissionError("player", callingPlayer.PlayerId, "player", cache.GetOwnerId(), uint64(types.PermissionHash), "hash")
		}
	}

	return cache.GetOwnerId(), owner, nil
}

/* Game Functions */

func (cache *StructCache) CanOreMinePlanet() error {

	if !cache.GetStructType().HasOreMiningSystem() {
		return types.NewStructCapabilityError(cache.StructId, "mining")
	}

    /*
	if cache.GetBlockStartOreMine() == 0 {
		return types.NewStructStateError(cache.StructId, "not_mining", "mining", "ore_mine")
	}
	*/

	if cache.GetPlanet().IsComplete() {
		return types.NewPlanetStateError(cache.GetPlanet().GetPlanetId(), "complete", "mine")
	}

	if cache.GetPlanet().IsEmptyOfOre() {
		return types.NewPlanetStateError(cache.GetPlanet().GetPlanetId(), "empty", "mine")
	}

	return nil

}

func (cache *StructCache) OreMinePlanet() {
	cache.GetOwner().StoredOreIncrement(1)
	cache.GetPlanet().BuriedOreDecrement(1)

	cache.ResetBlockStartOreMine()
}

func (cache *StructCache) CanOreRefine() error {

	if !cache.GetStructType().HasOreRefiningSystem() {
		return types.NewStructCapabilityError(cache.StructId, "refining")
	}

    /*
	if cache.GetBlockStartOreRefine() == 0 {
		return types.NewStructStateError(cache.StructId, "not_refining", "refining", "ore_refine")
	}
	*/

	if !cache.GetOwner().HasStoredOre() {
		return types.NewPlayerAffordabilityError(cache.GetOwner().PlayerId, "refine", "ore")
	}

	return nil

}

func (cache *StructCache) OreRefine() {

	cache.GetOwner().StoredOreDecrement(1)
	cache.GetOwner().DepositRefinedAlpha()

	cache.ResetBlockStartOreRefine()
}

func (cache *StructCache) CanAttack(targetStruct *StructCache, weaponSystem types.TechWeaponSystem) (err error) {

	if targetStruct.IsDestroyed() {
		err = types.NewCombatTargetingError(cache.StructId, targetStruct.StructId, weaponSystem.String(), "destroyed")
	} else {
		if !cache.GetStructType().CanTargetAmbit(weaponSystem, cache.GetOperatingAmbit(), targetStruct.GetOperatingAmbit()) {
			err = types.NewCombatTargetingError(cache.StructId, targetStruct.StructId, weaponSystem.String(), "out_of_range").WithAmbits(cache.GetOperatingAmbit().String(), targetStruct.GetOperatingAmbit().String())
		} else {
			// Not MVP CanBlockTargeting always returns false
			if (!cache.GetStructType().GetWeaponBlockable(weaponSystem)) && (targetStruct.GetStructType().CanBlockTargeting()) {
				err = types.NewCombatTargetingError(cache.StructId, targetStruct.StructId, weaponSystem.String(), "blocked")
			} else {
				if targetStruct.IsHidden() && (targetStruct.GetOperatingAmbit() != cache.GetOperatingAmbit()) {
					err = types.NewCombatTargetingError(cache.StructId, targetStruct.StructId, weaponSystem.String(), "hidden")
				}
			}
		}
	}

	// Now that the inexpensive checks are done, lets go deeper
	if err == nil {
		switch cache.GetLocationType() {
		case types.ObjectType_planet:
			if cache.GetPlanet().GetLocationListStart() == targetStruct.GetLocationId() {
				// The enemy fleet is here
			} else {
				err = types.NewCombatTargetingError(cache.StructId, targetStruct.StructId, weaponSystem.String(), "unreachable")
			}

		case types.ObjectType_fleet:
			// Is the Fleet at home?
			if cache.GetFleet().IsOnStation() {
				// If the Fleet is On Station, ensure the enemy is reachable
				if cache.GetPlanet().GetLocationListStart() == targetStruct.GetLocationId() {
					// The Fleet is on station, and the enemy is reachable
					// Proceed with the intended action for the Fleet attacking the target
				} else {
					err = types.NewCombatTargetingError(cache.StructId, targetStruct.StructId, weaponSystem.String(), "unreachable")
				}
				// Or is the Fleet out raiding another planet?
			} else {
				// If the Fleet is away, first check if the target is on the same planet
				if cache.GetFleet().GetLocationListForward() == "" && cache.GetPlanetId() == targetStruct.GetPlanetId() {
					// Target has reached the planetary raid
					// Proceed with the intended action for the Fleet attacking the target
					// Otherwise check if the target is adjacent (either forward or backward)
				} else if cache.GetFleet().GetLocationListForward() == targetStruct.GetLocationId() || cache.GetFleet().GetLocationListBackward() == targetStruct.GetLocationId() {
					// The target is to either side of the Fleet
					// Proceed with the intended action for the Fleet attacking the target
				} else {
					err = types.NewCombatTargetingError(cache.StructId, targetStruct.StructId, weaponSystem.String(), "unreachable")
				}
			}
		default:
			err = types.NewCombatTargetingError(cache.StructId, targetStruct.StructId, weaponSystem.String(), "unreachable")
		}
	}
	return
}

func (cache *StructCache) CanCounterAttack(attackerStruct *StructCache) (err error) {

	if attackerStruct.IsDestroyed() || cache.IsDestroyed() {
		cache.CC.k.logger.Info("Counter Struct or Attacker Struct is already destroyed", "counterStruct", cache.StructId, "target", attackerStruct.StructId)
		err = types.NewCombatTargetingError(cache.StructId, attackerStruct.StructId, "counter", "destroyed").AsCounter()
	} else {
		if !cache.GetStructType().CanCounterTargetAmbit(cache.GetOperatingAmbit(), attackerStruct.GetOperatingAmbit()) {
			cache.CC.k.logger.Info("Attacker Struct cannot be hit from Counter Struct using this weapon system", "target", attackerStruct.StructId, "counterStruct", cache.StructId)
			err = types.NewCombatTargetingError(cache.StructId, attackerStruct.StructId, "counter", "out_of_range").AsCounter().WithAmbits(cache.GetOperatingAmbit().String(), attackerStruct.GetOperatingAmbit().String())
		}
	}

	// Now that the inexpensive checks are done, lets go deeper
	if err == nil {
		switch cache.GetLocationType() {
		case types.ObjectType_planet:
			if cache.GetPlanet().GetLocationListStart() == attackerStruct.GetLocationId() {
				// The enemy fleet is here
			} else {
				err = types.NewCombatTargetingError(cache.StructId, attackerStruct.StructId, "counter", "unreachable").AsCounter()
			}

		case types.ObjectType_fleet:
			// Is the Fleet at home?
			if cache.GetFleet().IsOnStation() {
				// If the Fleet is On Station, ensure the enemy is reachable
				if cache.GetPlanet().GetLocationListStart() == attackerStruct.GetLocationId() {
					// The Fleet is on station, and the enemy is reachable
					// Proceed with the intended action for the Fleet attacking the target
				} else {
					err = types.NewCombatTargetingError(cache.StructId, attackerStruct.StructId, "counter", "unreachable").AsCounter()
				}
				// Or is the Fleet out raiding another planet?
			} else {
				// If the Fleet is away, first check if the target is on the same planet
				if cache.GetFleet().GetLocationListForward() == "" && cache.GetPlanetId() == attackerStruct.GetPlanetId() {
					// Target has reached the planetary raid
					// Proceed with the intended action for the Fleet attacking the target
					// Otherwise check if the target is adjacent (either forward or backward)
				} else if cache.GetFleet().GetLocationListForward() == attackerStruct.GetLocationId() || cache.GetFleet().GetLocationListBackward() == attackerStruct.GetLocationId() {
					// The target is to either side of the Fleet
					// Proceed with the intended action for the Fleet attacking the target
				} else {
					err = types.NewCombatTargetingError(cache.StructId, attackerStruct.StructId, "counter", "unreachable").AsCounter()
				}
			}
		default:
			err = types.NewCombatTargetingError(cache.StructId, attackerStruct.StructId, "counter", "unreachable").AsCounter()
		}
	}
	return
}

func (cache *StructCache) CanEvade(attackerStruct *StructCache, weaponSystem types.TechWeaponSystem) (canEvade bool) {

	var successRate fraction.Fraction
	switch attackerStruct.GetStructType().GetWeaponControl(weaponSystem) {
	case types.TechWeaponControl_guided:
		successRate = cache.GetStructType().GetGuidedDefensiveSuccessRate()
	case types.TechWeaponControl_unguided:
		successRate = cache.GetStructType().GetUnguidedDefensiveSuccessRate()
	}

	if successRate.Numerator() != int64(0) {
		canEvade = cache.IsSuccessful(successRate)
	}

	cache.GetEventAttackShotDetail().SetEvade(canEvade, cache.GetStructType().UnitDefenses)

	// If there has already been an successful evade then don't both evading harder
	if !canEvade {
		// Check for Planetary Defenses - Low Orbit Ballistic Interceptor Network
		if attackerStruct.GetLocationType() == types.ObjectType_fleet {

			// Is the Struct at home? Either via their fleet or on the planet directly
			if cache.GetPlanet().GetOwnerId() == cache.GetOwnerId() {

				// Grab the success rate for the interceptor network. If it returns an error, then the planet doesn't have it
				successRate, successRateError := cache.GetPlanet().GetLowOrbitBallisticsInterceptorNetworkSuccessRate()
				if successRateError == nil {

					// Only effective is the Struct is in the Air or Space
					if (attackerStruct.GetOperatingAmbit() == types.Ambit_air) || (attackerStruct.GetOperatingAmbit() == types.Ambit_space) {

						// Only effective if the target is in the Water or on Land
						if (cache.GetOperatingAmbit() == types.Ambit_water) || (cache.GetOperatingAmbit() == types.Ambit_land) {
							canEvade = cache.IsSuccessful(successRate)
							cache.GetEventAttackShotDetail().SetEvadeByPlanetaryDefenses(canEvade, types.TechPlanetaryDefenses_lowOrbitBallisticInterceptorNetwork)
						}
					}
				}
			}
		}
	}

	return
}

func (cache *StructCache) TakeAttackDamage(attackingStruct *StructCache, weaponSystem types.TechWeaponSystem) (damage uint64) {
	if cache.IsDestroyed() {
		return 0
	}


	for shot := uint64(0); shot < attackingStruct.GetStructType().GetWeaponShots(weaponSystem); shot++ {
		if attackingStruct.IsSuccessful(attackingStruct.GetStructType().GetWeaponShotSuccessRate(weaponSystem)) {
			damage = damage + attackingStruct.GetStructType().GetWeaponDamage(weaponSystem)
		}
	}

	cache.GetEventAttackShotDetail().SetDamageDealt(damage)

	if damage != 0 {
		damageReduction := cache.GetStructType().AttackReduction

		if damageReduction > 0 {
			cache.GetEventAttackShotDetail().SetDamageReduction(damageReduction, cache.GetStructType().UnitDefenses)
		}

		if damageReduction > damage {
			damage = 0
		} else {
			damage = damage - damageReduction
		}
	}

	cache.GetEventAttackShotDetail().SetDamage(damage)

	if damage != 0 {

        cache.CC.SetStructAttributeDecrement(cache.HealthAttributeId, damage)

		if cache.GetHealth() == 0 {
			if cache.Blocker {
				cache.GetEventAttackShotDetail().SetBlockerDestroyed()
			} else {
				cache.GetEventAttackShotDetail().SetTargetDestroyed()
			}

			// destruction damage from the grave
			if cache.GetStructType().PostDestructionDamage > 0 {
				attackingStruct.TakePostDestructionDamage(cache)
			}

			cache.DestroyAndCommit()
		}

	}

	// Always set final health (uses same Blocker pattern as SetBlockerDestroyed/SetTargetDestroyed)
	if cache.Blocker {

		cache.GetEventAttackShotDetail().SetBlockerHealthAfter(cache.GetHealth())
	} else {
		cache.GetEventAttackShotDetail().SetTargetHealthAfter(cache.GetHealth())
	}

	return
}

func (cache *StructCache) TakeRecoilDamage(weaponSystem types.TechWeaponSystem) (damage uint64) {
	if cache.IsDestroyed() {
		return 0
	}

	damage = cache.GetStructType().GetWeaponRecoilDamage(weaponSystem)

	if damage != 0 {
        cache.CC.SetStructAttributeDecrement(cache.HealthAttributeId, damage)

		if cache.GetHealth() == 0 {
			cache.DestroyAndCommit()
		}
	}

	cache.GetEventAttackDetail().SetRecoilDamage(damage, cache.IsDestroyed())
	return
}

func (cache *StructCache) TakePostDestructionDamage(attackingStruct *StructCache) (damage uint64) {
	if cache.IsDestroyed() {
		return 0
	}

	damage = cache.GetStructType().PostDestructionDamage

	if damage != 0 {
        cache.CC.SetStructAttributeDecrement(cache.HealthAttributeId, damage)

		if cache.GetHealth() == 0 {
			cache.DestroyAndCommit()
		}

	}

	cache.GetEventAttackShotDetail().SetPostDestructionDamage(damage, cache.IsDestroyed(), attackingStruct.GetStructType().PassiveWeaponry)

	return
}

func (cache *StructCache) TakeCounterAttackDamage(counterStruct *StructCache) (damage uint64) {
	if cache.IsDestroyed() {
		return 0
	}

	damage = counterStruct.GetStructType().GetCounterAttackDamage(cache.GetOperatingAmbit() == counterStruct.GetOperatingAmbit())
	cache.CC.k.logger.Info("Struct Counter-Attack", "damage", damage, "counterAttacker", counterStruct.GetStructId(), "target", cache.GetStructId())

	if damage != 0 {
        cache.CC.SetStructAttributeDecrement(cache.HealthAttributeId, damage)

		if cache.GetHealth() == 0 {
			// destruction damage from the grave
			cache.CC.k.logger.Info("Struct Destroyed During Counter-Attack", "counterAttacker", counterStruct.GetStructId(), "target", cache.GetStructId())
			if cache.GetStructType().PostDestructionDamage > 0 {
				counterStruct.TakePostDestructionDamage(cache)
			}
			cache.DestroyAndCommit()
		}

	}

	if counterStruct.Defender {
		cache.CC.k.logger.Info("Generating a Defender Counter-Attack Record for the event")
		cache.GetEventAttackShotDetail().AppendDefenderCounter(counterStruct.StructId, damage, cache.IsDestroyed(), counterStruct.GetTypeId(), counterStruct.GetLocationType(), counterStruct.GetLocationId(), counterStruct.GetOperatingAmbit(), counterStruct.GetSlot())
	} else {
		cache.CC.k.logger.Info("Generating a Target Counter-Attack Record for the event")
		cache.GetEventAttackShotDetail().AppendTargetCounter(damage, cache.IsDestroyed(), counterStruct.GetStructType().PassiveWeaponry)
	}

	return
}

func (cache *StructCache) TakePlanetaryDefenseCanonDamage(damage uint64) uint64 {
	if cache.IsDestroyed() {
		return 0
	}

	if damage != 0 {
        cache.CC.SetStructAttributeDecrement(cache.HealthAttributeId, damage)

		if cache.GetHealth() == 0 {
			cache.DestroyAndCommit()
		}
	}

	cache.GetEventAttackDetail().SetPlanetaryDefenseCannonDamage(damage, cache.IsDestroyed())

	return damage
}

func (cache *StructCache) AttemptBlock(attacker *StructCache, weaponSystem types.TechWeaponSystem, target *StructCache) (blocked bool) {
	if cache.Ready && attacker.Ready {
		if cache.GetOperatingAmbit() == target.GetOperatingAmbit() {
			blocked = true
			cache.Blocker = true
			cache.GetEventAttackShotDetail().SetBlocker(cache.StructId, cache.GetTypeId(), cache.GetLocationType(), cache.GetLocationId(), cache.GetOperatingAmbit(), cache.GetSlot())
			cache.TakeAttackDamage(attacker, weaponSystem)
		}
	}
	return
}

func (cache *StructCache) DestroyAndCommit() {

	// Go Offline
	// Most of the destruction process is handled during this sub-process
	cache.GoOffline()

	// Drop the Struct Type count for the owner
	cache.CC.SetStructAttributeDecrement(GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, cache.GetOwnerId(), cache.GetTypeId()), 1)

	// Don't clear these now, clear them on sweeps?
	// "health":               StructAttributeType_health,
	// "status":               StructAttributeType_status,

	// It's possible the build was never complete, so clear out this attribute to be safe
	cache.CC.ClearStructAttribute(cache.BlockStartBuildAttributeId)

	// Destroy mining systems
	if cache.GetStructType().HasOreMiningSystem() {
		cache.CC.ClearStructAttribute(cache.BlockStartOreMineAttributeId)
	}

	// Turn off the refinery
	if cache.GetStructType().HasOreRefiningSystem() {
		cache.CC.ClearStructAttribute(cache.BlockStartOreRefineAttributeId)
	}

	// Clear Defensive Relationships
	cache.CC.k.DestroyStructDefender(cache.CC.ctx, cache.GetStructId())

	// TODO clean this up to be more function based.. but it's fine
	if cache.GetStructType().HasPowerGenerationSystem() {
		// Clear out infusions
		infusions := cache.CC.GetAllInfusionByDestination(cache.StructId)
		for _, infusion := range infusions {
		    infusion.Destroy()
		}

		// Clear out all remaining allocations
		// clearing out all infusions should automatically clear allocations too,
		// but some allocations, such as automated ones may still exist
		allocations := cache.CC.GetAllAllocationBySource(cache.StructId)
        for _, allocation := range allocations {
            allocation.Destroy()
        }

		// Clear Load
		cache.CC.ClearGridAttribute(GetGridAttributeIDByObjectId(types.GridAttributeType_load, cache.StructId))

		// Clear Capacity
		cache.CC.ClearGridAttribute(GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, cache.StructId))

		// Clear Fuel
		cache.CC.ClearGridAttribute(GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, cache.StructId))

		// Clear Power
		cache.CC.ClearGridAttribute(GetGridAttributeIDByObjectId(types.GridAttributeType_power, cache.StructId))

	}

	// Clear Permissions
	// This only clears permissions for the current owner, which is likely a problem in the future.
	permissionId := GetObjectPermissionIDBytes(cache.StructId, cache.GetOwnerId())
	cache.CC.ClearPermissions(permissionId)

	// We're not going to remove it from the location yet, that happens during sweeps

	// Set to Destroyed
	cache.StatusAddDestroyed()


	// Can the Struct be a catalyst for raid-end
	// cache.GetFleet().Defeat()
	// Check for raid win conditions
	if cache.CanTriggerRaidDefeatByDestruction() {
		cache.GetFleet().Defeat()
	}

	cache.CC.k.AppendStructDestructionQueue(cache.CC.ctx, cache.StructId)
}

func (cache *StructCache) CanTriggerRaidDefeatByDestruction() bool {
	if !cache.GetStructType().TriggerRaidDefeatByDestruction {
		return false
	}

	// Make sure the ship isn't at home
	// This win condition only works to defeat the attacking fleet
	if cache.GetPlanet().GetOwnerId() != cache.GetOwnerId() {
		return true
	}
	return false
}

func (cache *StructCache) AttemptMove(destinationType types.ObjectType, ambit types.Ambit, slot uint64) error {
	if !cache.StructureLoaded {
		cache.LoadStruct()
	}

	if cache.IsOffline() {
		return types.NewStructStateError(cache.StructId, "offline", "online", "move")
	}

	switch destinationType {
        case types.ObjectType_planet:
            err := cache.GetOwner().GetPlanet().MoveReadiness(cache, ambit, slot)
            if err != nil {
                return err
            }
        case types.ObjectType_fleet:
            err := cache.GetOwner().GetFleet().MoveReadiness(cache, ambit, slot)
            if err != nil {
                return err
            }
        default:
            return types.NewStructBuildError(cache.GetTypeId(), destinationType.String(), "", "type_unsupported")
	}

	switch cache.Structure.LocationType {
        case types.ObjectType_planet:
            cache.GetOwner().GetPlanet().ClearSlot(cache.Structure.OperatingAmbit, cache.Structure.Slot)
        case types.ObjectType_fleet:
            if cache.GetStructType().Type != types.CommandStruct {
                cache.GetOwner().GetFleet().ClearSlot(cache.Structure.OperatingAmbit, cache.Structure.Slot)
            }
	}

	switch destinationType {
        case types.ObjectType_planet:

            cache.Structure.LocationId = cache.GetOwner().GetPlanetId()
            cache.Structure.LocationType = destinationType
            cache.Structure.OperatingAmbit = ambit

            // Update the cross reference on the planet
            err := cache.GetOwner().GetPlanet().SetSlot(cache.Structure)
            if err != nil {
                return err
            }

        case types.ObjectType_fleet:

            // Update the cross reference on the planet
            if cache.GetStructType().Type == types.CommandStruct {
                cache.Structure.OperatingAmbit = ambit
            } else {

                cache.Structure.LocationId = cache.GetOwner().GetFleetId()
                cache.Structure.LocationType = destinationType
                cache.Structure.OperatingAmbit = ambit
                cache.Structure.Slot = slot

                err := cache.GetOwner().GetFleet().SetSlot(cache.Structure)
                if err != nil {
                    return err
                }
            }


	}
    cache.Changed = true

	return nil
}
