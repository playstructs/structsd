package keeper

import (
	"context"

	//sdk "github.com/cosmos/cosmos-sdk/types"
    sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"

)


type StructCache struct {
    StructId string
    K *Keeper
    Ctx context.Context

    StructureLoaded  bool
    StructureChanged bool
    Structure  types.Struct

    StructTypeLoaded  bool
    StructType *types.StructType

    OwnerLoaded bool
    Owner *PlayerCache

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

    if (cache.StructureChanged) { cache.K.SetStruct(cache.Ctx, cache.Structure) }

    if (cache.HealthChanged) { cache.K.SetStructAttribute(cache.Ctx, cache.HealthAttributeId, cache.Health) }
    if (cache.StatusChanged) { cache.K.SetStructAttribute(cache.Ctx, cache.StatusAttributeId, uint64(cache.Status)) }

    if (cache.BlockStartBuildChanged) { cache.K.SetStructAttribute(cache.Ctx, cache.BlockStartBuildAttributeId, cache.BlockStartBuild) }
    if (cache.BlockStartOreMineChanged) { cache.K.SetStructAttribute(cache.Ctx, cache.BlockStartOreMineAttributeId, cache.BlockStartOreMine) }
    if (cache.BlockStartOreRefineChanged) { cache.K.SetStructAttribute(cache.Ctx, cache.BlockStartOreRefineAttributeId, cache.BlockStartOreRefine) }

    if (cache.ProtectedStructIndexChanged) { cache.K.SetStructAttribute(cache.Ctx, cache.ProtectedStructIndexAttributeId, cache.ProtectedStructIndex) }
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

func (cache *StructCache) ReadinessCheck() (err error) {
    if (cache.IsOffline()) {
        err = sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) is offline. Activate it", cache.StructId)
    } else {
        if (cache.GetOwner().IsOffline()) {
            err = sdkerrors.Wrapf(types.ErrGridMalfunction, "Player (%s) is offline due to power", cache.GetOwnerId())
        }
    }
    return
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
            err = sdkerrors.Wrapf(types.ErrStructAction, "Target Struct (%s) cannot be hit from Attacker Struct (%s)", targetStruct.StructId, cache.StructId)
        } else {
            // Not MVP CanBlockTargeting always returns false
            if ((!cache.GetStructType().GetWeaponBlockable(weaponSystem)) && (targetStruct.GetStructType().CanBlockTargeting())) {
                err = sdkerrors.Wrapf(types.ErrStructAction, "Target Struct (%s) currently blocking Attacker Struct (%s)", targetStruct.StructId, cache.StructId)
            }
        }
     }

     // Now that the inexpensive checks are done, lets go deeper
     if (err == nil) {
        // TODO right here
        // Add in Location check
     }
    return
}

func (cache *StructCache) TakeDamage(damage uint64, attackingStruct *StructCache, weaponSystem types.TechWeaponSystem) {

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
}

func (cache *StructCache) DestroyAndCommit() {

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