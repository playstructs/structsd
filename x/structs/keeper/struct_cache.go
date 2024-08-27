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

)


type StructCache struct {
    StructId string
    K *Keeper
    Ctx context.Context

    Ready bool

    StructureLoaded  bool
    StructureChanged bool
    Structure  types.Struct

    StructTypeLoaded  bool
    StructType *types.StructType

    OwnerLoaded bool
    Owner *PlayerCache

    FleetLoaded bool
    Fleet *types.Fleet

    PlanetLoaded bool
    Planet *types.Planet

    DefendersLoaded bool
    Defenders []types.StructDefender

    HealthAttributeId string
    HealthLoaded  bool
    HealthChanged bool
    Health  uint64

    StatusAttributeId string
    StatusLoaded  bool
    StatusChanged bool
    Status types.StructState

    BlockStartBuildAttributeId string
    BlockStartBuildLoaded bool
    BlockStartBuildChanged bool
    BlockStartBuild  uint64

    BlockStartOreMineAttributeId string
    BlockStartOreMineLoaded bool
    BlockStartOreMineChanged bool
    BlockStartOreMine uint64

    BlockStartOreRefineAttributeId string
    BlockStartOreRefineLoaded bool
    BlockStartOreRefineChanged bool
    BlockStartOreRefine   uint64

    ProtectedStructIndexAttributeId string
    ProtectedStructIndexLoaded bool
    ProtectedStructIndexChanged bool
    ProtectedStructIndex   uint64
}

// Build this initial Struct Cache object
// This does no validation on the provided structId
func (k *Keeper) GetStructCacheFromId(ctx context.Context, structId string) (StructCache) {
    return StructCache{
        StructId: structId,
        K: k,
        Ctx: ctx,

        HealthAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_health, structId),
        StatusAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_status, structId),

        BlockStartBuildAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structId),
        BlockStartOreMineAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, structId),
        BlockStartOreRefineAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, structId),

        ProtectedStructIndexAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, structId),
    }
}

func (cache *StructCache) Commit() () {

    if (cache.StructureChanged) {
        cache.K.SetStruct(cache.Ctx, cache.Structure)
        cache.StructureChanged = false
    }

    if (cache.HealthChanged) {
        cache.K.SetStructAttribute(cache.Ctx, cache.HealthAttributeId, cache.Health)
        cache.HealthChanged = false
    }

    if (cache.StatusChanged) {
        cache.K.SetStructAttribute(cache.Ctx, cache.StatusAttributeId, uint64(cache.Status))
        cache.StatusChanged = false
    }

    if (cache.BlockStartBuildChanged) {
        cache.K.SetStructAttribute(cache.Ctx, cache.BlockStartBuildAttributeId, cache.BlockStartBuild)
        cache.BlockStartBuildChanged = false
    }
    if (cache.BlockStartOreMineChanged) {
        cache.K.SetStructAttribute(cache.Ctx, cache.BlockStartOreMineAttributeId, cache.BlockStartOreMine)
        cache.BlockStartOreMineChanged = false
    }
    if (cache.BlockStartOreRefineChanged) {
        cache.K.SetStructAttribute(cache.Ctx, cache.BlockStartOreRefineAttributeId, cache.BlockStartOreRefine)
        cache.BlockStartOreRefineChanged = false
    }

    if (cache.ProtectedStructIndexChanged) {
        cache.K.SetStructAttribute(cache.Ctx, cache.ProtectedStructIndexAttributeId, cache.ProtectedStructIndex)
        cache.ProtectedStructIndexChanged = false
    }
}

/* Separate Loading functions for each of the underlying containers */

// Load the core Struct data
func (cache *StructCache) LoadStruct() (bool) {
    cache.Structure, cache.StructureLoaded = cache.K.GetStruct(cache.Ctx, cache.StructId)
    return cache.StructureLoaded
}

// Load the Struct Type data
func (cache *StructCache) LoadStructType() (bool) {
    newStructType, newStructTypeFound := cache.K.GetStructType(cache.Ctx, cache.GetTypeId())
    cache.StructType = &newStructType
    cache.StructTypeLoaded = newStructTypeFound
    return cache.StructTypeLoaded
}

// Load the Player data
func (cache *StructCache) LoadOwner() (bool) {
    newOwner, _ := cache.K.GetPlayerCacheFromId(cache.Ctx, cache.GetOwnerId())
    cache.Owner = &newOwner
    cache.OwnerLoaded = true
    return cache.OwnerLoaded
}

// Load the Fleet data
func (cache *StructCache) LoadFleet() (bool) {
    newFleet, newFleetFound := cache.K.GetFleet(cache.Ctx, cache.GetLocationId())
    cache.Fleet = &newFleet
    cache.FleetLoaded = newFleetFound
    return cache.FleetLoaded
}

// Load the Planet data
func (cache *StructCache) LoadPlanet() (bool) {
    newPlanet, newPlanetFound := cache.K.GetPlanet(cache.Ctx, cache.GetLocationId())
    cache.Planet = &newPlanet
    cache.PlanetLoaded = newPlanetFound
    return cache.PlanetLoaded
}

// Load the Defenders data
func (cache *StructCache) LoadDefenders() (bool) {
    cache.Defenders = cache.K.GetAllStructDefender(cache.Ctx, cache.GetOwnerId())
    cache.DefendersLoaded = true
    return cache.DefendersLoaded
}

// Load the Health record
func (cache *StructCache) LoadHealth() {
    cache.Health = cache.K.GetStructAttribute(cache.Ctx, cache.HealthAttributeId)
    cache.HealthLoaded = true
}

// Load the Struct Status record
func (cache *StructCache) LoadStatus() {
    cache.Status = types.StructState(cache.K.GetStructAttribute(cache.Ctx, cache.StatusAttributeId))
    cache.StatusLoaded = true
}

// Load the Struct BlockStartBuild record
func (cache *StructCache) LoadBlockStartBuild() {
    cache.BlockStartBuild = cache.K.GetStructAttribute(cache.Ctx, cache.BlockStartBuildAttributeId)
    cache.BlockStartBuildLoaded = true
}

// Load the Struct BlockStarOreMine record
func (cache *StructCache) LoadBlockStartOreMine() {
    cache.BlockStartOreMine = cache.K.GetStructAttribute(cache.Ctx, cache.BlockStartOreMineAttributeId)
    cache.BlockStartOreMineLoaded = true
}

// Load the Struct BlockStartOreRefine record
func (cache *StructCache) LoadBlockStartOreRefine() {
    cache.BlockStartOreRefine = cache.K.GetStructAttribute(cache.Ctx, cache.BlockStartOreRefineAttributeId)
    cache.BlockStartOreRefineLoaded = true
}

// Load the Struct BlockStartOreRefine record
func (cache *StructCache) LoadProtectedStructIndex() {
    cache.ProtectedStructIndex = cache.K.GetStructAttribute(cache.Ctx, cache.ProtectedStructIndexAttributeId)
    cache.ProtectedStructIndexLoaded = true
}


/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

func (cache *StructCache) GetHealth() (uint64) {
    if (!cache.HealthLoaded) { cache.LoadHealth() }
    return cache.Health
}

// Get the Owner ID data
func (cache *StructCache) GetOwnerId() (string) {
    if (!cache.StructureLoaded) { cache.LoadStruct() }
    return cache.Structure.Owner
}

// Get the Owner data
func (cache *StructCache) GetOwner() (*PlayerCache) {
    if (!cache.OwnerLoaded) { cache.LoadOwner() }
    return cache.Owner
}

// Get the Defenders data
func (cache *StructCache) GetDefenders() ([]types.StructDefender) {
    if (!cache.DefendersLoaded) { cache.LoadDefenders() }
    return cache.Defenders
}

func (cache *StructCache) GetFleet() (*types.Fleet) {
    if (!cache.FleetLoaded) { cache.LoadFleet() }
    return cache.Fleet
}

func (cache *StructCache) GetPlanet() (*types.Planet) {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }
    return cache.Planet
}

// Get the Location ID data
func (cache *StructCache) GetLocationId() (string) {
    if (!cache.StructureLoaded) { cache.LoadStruct() }
    return cache.Structure.LocationId
}

// Get the Location Type data
func (cache *StructCache) GetLocationType() (types.ObjectType) {
    if (!cache.StructureLoaded) { cache.LoadStruct() }
    return cache.Structure.LocationType
}

func (cache *StructCache) GetOperatingAmbit() (types.Ambit) {
    if (!cache.StructureLoaded) { cache.LoadStruct() }
    return cache.Structure.OperatingAmbit
}

func (cache *StructCache) GetStatus() (types.StructState) {
    if (!cache.StatusLoaded) { cache.LoadStatus() }
    return cache.Status
}

func (cache *StructCache) GetStruct() (types.Struct) {
    if (!cache.StructureLoaded) { cache.LoadStruct() }
    return cache.Structure
}

func (cache *StructCache) GetTypeId() (uint64) {
    if (!cache.StructureLoaded) { cache.LoadStruct() }
    return cache.Structure.Type
}

func (cache *StructCache) GetStructType() (*types.StructType) {
    if (!cache.StructTypeLoaded) { cache.LoadStructType() }
    return cache.StructType
}

/* Setters - SET DOES NOT COMMIT()
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */


// Set the Owner Id data
func (cache *StructCache) SetOwnerId(owner string) {
    if (!cache.StructureLoaded) { cache.LoadStruct() }

    cache.Structure.Owner = owner
    cache.StructureChanged = true

    // Player object might be stale now
    cache.OwnerLoaded = false
}

// Set the Owner data manually
// Useful for loading multiple defenders
func (cache *StructCache) ManualLoadOwner(owner *PlayerCache) {
    cache.Owner = owner
    cache.OwnerLoaded = true
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

func (cache *StructCache) IsOffline() bool {
    return !cache.IsOnline()
}

func (cache *StructCache) IsDestroyed() bool {
   return cache.GetStatus()&types.StructStateDestroyed != 0
}



func (cache *StructCache) ReadinessCheck() (err error) {
    if (cache.IsOffline()) {
        err = sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) is offline. Activate it", cache.StructId)
    } else {
        if (cache.GetOwner().IsOffline()) {
            err = sdkerrors.Wrapf(types.ErrGridMalfunction, "Player (%s) is offline due to power", cache.GetOwnerId())
        }
    }

    cache.Ready = true
    return
}

/* Rough but Consistent Randomness Check */
func (cache *StructCache) IsSuccessful(successRate fraction.Fraction) bool {
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

/* Permissions */
func (cache *StructCache) CanBePlayedBy(address string) (err error) {

    // Make sure the address calling this has Play permissions
    if (!cache.K.PermissionHasOneOf(cache.Ctx, GetAddressPermissionIDBytes(address), types.PermissionPlay)) {
        err = sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", address)

    } else {
        callingPlayer, err := cache.K.GetPlayerCacheFromAddress(cache.Ctx, address)
        if (err != nil) {
            if (callingPlayer.PlayerId != cache.GetOwnerId()) {
                if (!cache.K.PermissionHasOneOf(cache.Ctx, GetObjectPermissionIDBytes(cache.GetOwnerId(), callingPlayer.PlayerId), types.PermissionPlay)) {
                   err = sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling account (%s) has no play permissions on target player (%s)", callingPlayer.PlayerId, cache.GetOwnerId())
                }
            }
        }
    }

    return
}

/* Game Functions */

func (cache *StructCache) CanAttack(targetStruct *StructCache, weaponSystem types.TechWeaponSystem) (err error) {

     if (targetStruct.IsDestroyed()) {
        err = sdkerrors.Wrapf(types.ErrStructAction, "Target Struct (%s) is already destroyed", targetStruct.StructId)
     } else {
        if (!cache.GetStructType().CanTargetAmbit(weaponSystem, targetStruct.GetOperatingAmbit())) {
            err = sdkerrors.Wrapf(types.ErrStructAction, "Target Struct (%s) cannot be hit from Attacker Struct (%s) using this weapon system", targetStruct.StructId, cache.StructId)
        } else {
            // Not MVP CanBlockTargeting always returns false
            if ((!cache.GetStructType().GetWeaponBlockable(weaponSystem)) && (targetStruct.GetStructType().CanBlockTargeting())) {
                err = sdkerrors.Wrapf(types.ErrStructAction, "Target Struct (%s) currently blocking Attacker Struct (%s)", targetStruct.StructId, cache.StructId)
            }
        }
     }

     // Now that the inexpensive checks are done, lets go deeper
     if (err == nil) {
        switch (cache.GetLocationType()) {
            case types.ObjectType_planet:
                if (cache.GetPlanet().GetLocationListStart() != targetStruct.GetLocationId()) {
                    err = sdkerrors.Wrapf(types.ErrStructAction, "Target Struct (%s) is unreachable by Planetary Attacker Struct (%s)", targetStruct.StructId, cache.StructId)
                }
            case types.ObjectType_fleet:
                if ((cache.GetFleet().GetLocationListForward() != targetStruct.GetLocationId()) && (cache.GetFleet().GetLocationListBackward() != targetStruct.GetLocationId())) {
                    err = sdkerrors.Wrapf(types.ErrStructAction, "Target Struct (%s) is unreachable by Fleet Attacker Struct (%s)", targetStruct.StructId, cache.StructId)
                }
            default:
                err = sdkerrors.Wrapf(types.ErrStructAction, "Target Struct (%s) is unreachable by Attacker Struct (%s). Should tell an adult about this one", targetStruct.StructId, cache.StructId)
        }
     }
    return
}


func (cache *StructCache) CanCounterAttack(attackerStruct *StructCache) (err error) {

     if (attackerStruct.IsDestroyed() || cache.IsDestroyed()) {
        err = sdkerrors.Wrapf(types.ErrStructAction, "Counter Struct (%s) or Attacker Struct (%s) is already destroyed", cache.StructId, attackerStruct.StructId)
     } else {
        if (!cache.GetStructType().CanCounterTargetAmbit(attackerStruct.GetOperatingAmbit())) {
            err = sdkerrors.Wrapf(types.ErrStructAction, "Attacker Struct (%s) cannot be hit from Counter Struct (%s) using this weapon system", attackerStruct.StructId, cache.StructId)
        }
     }

     // Now that the inexpensive checks are done, lets go deeper
     if (err == nil) {
        switch (cache.GetLocationType()) {
            case types.ObjectType_planet:
                if (cache.GetPlanet().GetLocationListStart() != attackerStruct.GetLocationId()) {
                    err = sdkerrors.Wrapf(types.ErrStructAction, "Target Struct (%s) is unreachable by Planetary Attacker Struct (%s)", attackerStruct.StructId, cache.StructId)
                }
            case types.ObjectType_fleet:
                if ((cache.GetFleet().GetLocationListForward() != attackerStruct.GetLocationId()) && (cache.GetFleet().GetLocationListBackward() != attackerStruct.GetLocationId())) {
                    err = sdkerrors.Wrapf(types.ErrStructAction, "Attacker Struct (%s) is unreachable by Counter Attacker Struct (%s)", attackerStruct.StructId, cache.StructId)
                }
            default:
                err = sdkerrors.Wrapf(types.ErrStructAction, "Attacker Struct (%s) is unreachable by Counter Struct (%s). Should tell an adult about this one", attackerStruct.StructId, cache.StructId)
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

    if (successRate.Numerator() != int64(0)) {
        canEvade = cache.IsSuccessful(successRate)
    }
    return
}

func (cache *StructCache) TakeAttackDamage(attackingStruct *StructCache, weaponSystem types.TechWeaponSystem) (damage uint64) {
    if (cache.IsDestroyed()) { return 0 }

    for shot := uint64(0); shot < attackingStruct.GetStructType().GetWeaponShots(weaponSystem); shot++ {
        if (attackingStruct.IsSuccessful(attackingStruct.GetStructType().GetWeaponShotSuccessRate(weaponSystem))) {
            damage = damage + attackingStruct.GetStructType().GetWeaponDamage(weaponSystem)
        }
    }

    if (damage != 0) {
        damageReduction := cache.GetStructType().GetAttackReduction()

        if (damageReduction > damage) {
            damage = 0
        } else {
            damage = damage - damageReduction
        }
    }

    if (damage != 0) {

        if (damage > cache.GetHealth()) {
            cache.Health = 0
            cache.HealthChanged = true

        } else {
            cache.Health = cache.GetHealth() - damage
            cache.HealthChanged = true
        }

        if (cache.Health == 0) {
            // TODO destruction damage from the grave
            cache.DestroyAndCommit()
        }

    }

    return
}


func (cache *StructCache) TakeRecoilDamage(weaponSystem types.TechWeaponSystem) (damage uint64) {
    if (cache.IsDestroyed()) { return 0 }

    damage = cache.GetStructType().GetWeaponRecoilDamage(weaponSystem)

    if (damage != 0) {

        if (damage > cache.GetHealth()) {
            cache.Health = 0
            cache.HealthChanged = true

        } else {
            cache.Health = cache.GetHealth() - damage
            cache.HealthChanged = true
        }

        if (cache.Health == 0) {
            cache.DestroyAndCommit()
        }

    }

    return
}


func (cache *StructCache) TakeCounterAttackDamage(counterStruct *StructCache) (damage uint64) {
    if (cache.IsDestroyed()) { return 0 }

    damage = counterStruct.GetStructType().GetCounterAttackDamage(cache.GetOperatingAmbit() == counterStruct.GetOperatingAmbit())

    if (damage != 0) {

        if (damage > cache.GetHealth()) {
            cache.Health = 0
            cache.HealthChanged = true

        } else {
            cache.Health = cache.GetHealth() - damage
            cache.HealthChanged = true
        }

        if (cache.Health == 0) {
            cache.DestroyAndCommit()
        }

    }

    return
}

func (cache *StructCache) AttemptBlock(attacker *StructCache, weaponSystem types.TechWeaponSystem, target *StructCache) (blocked bool) {
    if (cache.Ready && attacker.Ready) {
        if (cache.GetOperatingAmbit() == target.GetOperatingAmbit()) {
            blocked = true
            cache.TakeAttackDamage(attacker, weaponSystem)
        }
    }
    return
}

func (cache *StructCache) DestroyAndCommit() {
    if (!cache.IsDestroyed()) {

    }

    // Go Offline
    cache.Status = cache.Status &^ types.StructStateOnline

    // Remove from Planet
    // Remove from Planet Attributes

    // If a power planet
        // Remove infusions
        // Remove allocations
            // cascade


    // Set to Destroyed
    cache.Status = cache.Status | types.StructStateDestroyed

}