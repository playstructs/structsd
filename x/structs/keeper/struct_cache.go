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

	BlockStartBuildAttributeId string
	BlockStartOreMineAttributeId string
	BlockStartOreRefineAttributeId string
	ProtectedStructIndexAttributeId string
	ReadyAttributeId string

	structType *types.StructType

	CounterSpent bool
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

func (cache *StructCache) IsCounterSpent() bool {
	return cache.CounterSpent
}

func (cache *StructCache) SetCounterSpent() {
	cache.CounterSpent = true
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
    if cache.structType == nil {
        st, _ := cache.CC.GetStructType(cache.GetTypeId())
        result := st.GetStructType()
        cache.structType = &result
    }
    return *cache.structType
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
    if cache.IsOnline() {
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

	randomnessCheck := ((randomnessOrb.Intn(max-min+1) + min) <= int(successRate.Numerator()))
	cache.CC.k.logger.Info("Struct Success-Check Randomness", "structId", cache.GetStructId(), "seed", seed, "offset", cache.GetOwner().GetNextNonce(), "seedOffset", seedOffset, "numerator", successRate.Numerator(), "denominator", successRate.Denominator(), "success", randomnessCheck)

	return randomnessCheck
}

/* Permissions */
func (cache *StructCache) CanBePlayedBy(callingPlayer *PlayerCache) error {
    return cache.CC.PermissionCheck(cache.GetOwner(), callingPlayer, types.PermPlay)
}

func (cache *StructCache) CanAllocateAsSourceBy(_ *PlayerCache) error {
    return types.NewAllocationError(cache.ID(), "unacceptable_source")
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

func (cache *StructCache) OreRefine() error {

	cache.GetOwner().StoredOreDecrement(1)
	if err := cache.GetOwner().DepositRefinedAlpha(); err != nil {
		return err
	}

	cache.ResetBlockStartOreRefine()
	return nil
}

func (cache *StructCache) isReachable(targetLocationId string, targetPlanetId string) bool {
	switch cache.GetLocationType() {
	case types.ObjectType_planet:
		return cache.GetPlanet().GetLocationListStart() == targetLocationId
	case types.ObjectType_fleet:
		if cache.GetFleet().IsOnStation() {
			return cache.GetPlanet().GetLocationListStart() == targetLocationId
		}
		if cache.GetFleet().GetLocationListForward() == "" && cache.GetPlanetId() == targetPlanetId {
			return true
		}
		return cache.GetFleet().GetLocationListForward() == targetLocationId ||
			cache.GetFleet().GetLocationListBackward() == targetLocationId
	default:
		return false
	}
}

func (cache *StructCache) CanAttack(targetStruct *StructCache, weaponSystem types.TechWeaponSystem) (err error) {
	if targetStruct.IsDestroyed() {
		return types.NewCombatTargetingError(cache.StructId, targetStruct.StructId, weaponSystem.String(), "destroyed")
	}
	if !cache.GetStructType().CanTargetAmbit(weaponSystem, cache.GetOperatingAmbit(), targetStruct.GetOperatingAmbit()) {
		return types.NewCombatTargetingError(cache.StructId, targetStruct.StructId, weaponSystem.String(), "out_of_range").WithAmbits(cache.GetOperatingAmbit().String(), targetStruct.GetOperatingAmbit().String())
	}
	if (!cache.GetStructType().GetWeaponBlockable(weaponSystem)) && (targetStruct.GetStructType().CanBlockTargeting()) {
		return types.NewCombatTargetingError(cache.StructId, targetStruct.StructId, weaponSystem.String(), "blocked")
	}
	if targetStruct.IsHidden() && (targetStruct.GetOperatingAmbit() != cache.GetOperatingAmbit()) {
		return types.NewCombatTargetingError(cache.StructId, targetStruct.StructId, weaponSystem.String(), "hidden")
	}
	if !cache.isReachable(targetStruct.GetLocationId(), targetStruct.GetPlanetId()) {
		return types.NewCombatTargetingError(cache.StructId, targetStruct.StructId, weaponSystem.String(), "unreachable")
	}
	return nil
}

func (cache *StructCache) CanCounterAttack(attackerStruct *StructCache) (err error) {
	readinessError := cache.ReadinessCheck()
	if readinessError != nil {
		return readinessError
	}

	if cache.IsCounterSpent() {
		return types.NewCombatTargetingError(cache.StructId, attackerStruct.StructId, "counter", "spent").AsCounter()
	}

	if attackerStruct.IsDestroyed() || cache.IsDestroyed() {
		return types.NewCombatTargetingError(cache.StructId, attackerStruct.StructId, "counter", "destroyed").AsCounter()
	}
	if !cache.GetStructType().CanCounterTargetAmbit(cache.GetOperatingAmbit(), attackerStruct.GetOperatingAmbit()) {
		return types.NewCombatTargetingError(cache.StructId, attackerStruct.StructId, "counter", "out_of_range").AsCounter().WithAmbits(cache.GetOperatingAmbit().String(), attackerStruct.GetOperatingAmbit().String())
	}
	if !cache.isReachable(attackerStruct.GetLocationId(), attackerStruct.GetPlanetId()) {
		return types.NewCombatTargetingError(cache.StructId, attackerStruct.StructId, "counter", "unreachable").AsCounter()
	}
	return nil
}

// applyPostDestructionDamageCore applies post-destruction damage to the attacker without mutating attack event details.
func (attackingStruct *StructCache) applyPostDestructionDamageCore(destroyedStruct *StructCache) (damage uint64, attackerDestroyed bool, passive types.TechPassiveWeaponry) {
	if attackingStruct.IsDestroyed() {
		return 0, false, types.TechPassiveWeaponry_noPassiveWeaponry
	}

	damage = destroyedStruct.GetStructType().PostDestructionDamage
	passive = destroyedStruct.GetStructType().PassiveWeaponry

	if damage != 0 {
		attackingStruct.CC.SetStructAttributeDecrement(attackingStruct.HealthAttributeId, damage)

		if attackingStruct.GetHealth() == 0 {
			attackingStruct.DestroyAndCommit()
		}
	}

	attackerDestroyed = attackingStruct.IsDestroyed()
	return damage, attackerDestroyed, passive
}


func (cache *StructCache) DestroyAndCommit() {

	if !cache.IsBuilt() {
		// Struct was still building — release the BuildDraw energy that was
		// reserved during build initiation. GoOffline() won't handle this
		// because the struct was never online.
		cache.GetOwner().StructsLoadDecrement(cache.GetStructType().BuildDraw)
	}

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
