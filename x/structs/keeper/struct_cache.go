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
    OwnerChanged bool
    Owner *PlayerCache

    FleetLoaded bool
    FleetChanged bool
    Fleet *FleetCache

    PlanetLoaded bool
    PlanetChanged bool
    Planet *PlanetCache

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

    Blocker bool
    Defender bool

    // Event Tracking
    EventAttackDetailLoaded bool
    EventAttackDetail *types.EventAttackDetail

    EventAttackShotDetailLoaded bool
    EventAttackShotDetail *types.EventAttackShotDetail

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

// Build this initial Struct Cache object
// This does no validation on the provided structId
func (k *Keeper) InitiateStruct(ctx context.Context, creatorAddress string, owner *PlayerCache, structType *types.StructType, destinationId string, destinationType types.ObjectType, ambit types.Ambit, slot uint64) (StructCache, error) {

    var ownerChanged bool

    var planet *PlanetCache
    var planetChanged bool

    var fleet *FleetCache
    var fleetChanged bool

    structure := types.CreateBaseStruct(structType, creatorAddress, owner.GetPlayerId(), destinationId, destinationType, ambit)

    switch destinationType {
        case types.ObjectType_planet:
            // Load Planet from the PlanetId
            planet := k.GetPlanetCacheFromId(ctx, destinationId)
            planetFound := planet.LoadPlanet()
            if !planetFound {
                return StructCache{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Planet (%s) was not found. Building a Struct in a void might be tough", destinationId)
            }

            err := planet.BuildInitiateReadiness(&structure, structType, ambit, slot)
            if (err != nil) {
                return StructCache{}, err
            }

            structure.Slot = slot

        case types.ObjectType_fleet:
                // Load the Fleet
                fleet, _ := k.GetFleetCacheFromId(ctx, destinationId)
                fleetFound := fleet.LoadFleet()
                if !fleetFound {
                    return StructCache{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Fleet (%s) was not found. Building a Struct in a void might be tough", destinationId)
                }

                err := fleet.BuildInitiateReadiness(&structure, structType, ambit, slot)
                if (err != nil) {
                    return StructCache{}, err
                }

                if (structType.Type != types.CommandStruct) {
                    structure.Slot = slot
                }

        default:
            return StructCache{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "We're not building these yet")
    }


    // Append Struct
    structure = k.AppendStruct(ctx, structure)


    owner.StructsLoadIncrement(structType.GetBuildDraw())
    owner.BuildQuantityIncrement(structType.GetId())
    ownerChanged = true


   switch destinationType {
        case types.ObjectType_planet:

            // Update the cross reference on the planet
            err := planet.SetSlot(structure)
            if (err != nil) {
                return StructCache{}, err
            }
            planetChanged = true

        case types.ObjectType_fleet:
            // Update the cross reference on the planet
            if (structType.Type == types.CommandStruct) {
                fleet.SetCommandStruct(structure)
            } else {
                err := fleet.SetSlot(structure)
                if (err != nil) {
                    return StructCache{}, err
                }
            }

            fleetChanged = true

    }


    // Start to put the pieces together
    structCache := StructCache{
                  StructId: structure.Id,
                  K: k,
                  Ctx: ctx,

                  Owner: owner,
                  OwnerLoaded: true,
                  OwnerChanged: ownerChanged,

                  Planet: planet,
                  PlanetLoaded: true,
                  PlanetChanged: planetChanged,

                  Fleet: fleet,
                  FleetLoaded: true,
                  FleetChanged: fleetChanged,

                  // Include the health value
                  HealthAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_health, structure.Id),
                  HealthChanged: true,
                  HealthLoaded: true,
                  Health: structType.GetMaxHealth(),

                  // Include the initial status value
                  StatusAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_status, structure.Id),
                  StatusChanged: true,
                  StatusLoaded: true,
                  Status: types.StructState(types.StructStateMaterialized),


                  // include the initial build block value
                  BlockStartBuildAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structure.Id),
                  BlockStartOreMineAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, structure.Id),
                  BlockStartOreRefineAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, structure.Id),

                  ProtectedStructIndexAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, structure.Id),
    }

    return structCache, nil
}


func (cache *StructCache) Commit() () {

    if (cache.StructureChanged) {
        cache.K.SetStruct(cache.Ctx, cache.Structure)
        cache.StructureChanged = false
    }

    if (cache.OwnerChanged) {
        cache.GetOwner().Commit()
        cache.OwnerChanged = false
    }

    if (cache.PlanetChanged) {
        cache.GetPlanet().Commit()
        cache.PlanetChanged = false
    }

    if (cache.FleetChanged) {
        cache.GetFleet().Commit()
        cache.FleetChanged = false
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
    newFleet, _ := cache.K.GetFleetCacheFromId(cache.Ctx, cache.GetOwner().GetFleetId())
    cache.Fleet = &newFleet
    cache.FleetLoaded = true
    return cache.FleetLoaded
}

// Load the Planet data
func (cache *StructCache) LoadPlanet() (bool) {
    switch (cache.GetLocationType()) {
        case types.ObjectType_planet:
            newPlanet := cache.K.GetPlanetCacheFromId(cache.Ctx, cache.GetLocationId())
            cache.Planet = &newPlanet
            cache.PlanetLoaded = true
        case types.ObjectType_fleet:
            if (cache.GetFleet().GetLocationType() == types.ObjectType_planet) {
                newPlanet := cache.K.GetPlanetCacheFromId(cache.Ctx, cache.GetFleet().GetLocationId())
                cache.Planet = &newPlanet
                cache.PlanetLoaded = true
            }
    }
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

// Set the Owner data manually
// Useful for loading multiple defenders
func (cache *StructCache) ManualLoadOwner(owner *PlayerCache) {
    cache.Owner = owner
    cache.OwnerLoaded = true
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

func (cache *StructCache) GetStruct()   (types.Struct)  { if (!cache.StructureLoaded) { cache.LoadStruct() }; return cache.Structure }
func (cache *StructCache) GetStructId() (string)        {  return cache.StructId }

func (cache *StructCache) GetHealth()               (uint64)            { if (!cache.HealthLoaded) { cache.LoadHealth() }; return cache.Health }
func (cache *StructCache) GetStatus()               (types.StructState) { if (!cache.StatusLoaded) { cache.LoadStatus() }; return cache.Status }
func (cache *StructCache) GetBlockStartBuild()      (uint64)            { if (!cache.BlockStartBuildLoaded) { cache.LoadBlockStartBuild() }; return cache.BlockStartBuild }
func (cache *StructCache) GetBlockStartOreMine()    (uint64)            { if (!cache.BlockStartOreMineLoaded) { cache.LoadBlockStartOreMine() }; return cache.BlockStartOreMine }
func (cache *StructCache) GetBlockStartOreRefine()  (uint64)            { if (!cache.BlockStartOreRefineLoaded) { cache.LoadBlockStartOreRefine() }; return cache.BlockStartOreRefine }

func (cache *StructCache) GetStructType()   (*types.StructType) { if (!cache.StructTypeLoaded) { cache.LoadStructType() }; return cache.StructType }
func (cache *StructCache) GetTypeId()       (uint64)            { if (!cache.StructureLoaded) { cache.LoadStruct() }; return cache.Structure.Type }

func (cache *StructCache) GetOwner()    (*PlayerCache)  { if (!cache.OwnerLoaded) { cache.LoadOwner() }; return cache.Owner }
func (cache *StructCache) GetOwnerId()  (string)        { if (!cache.StructureLoaded) { cache.LoadStruct() }; return cache.Structure.Owner }

func (cache *StructCache) GetLocationId()       (string)            { if (!cache.StructureLoaded) { cache.LoadStruct() }; return cache.Structure.LocationId }
func (cache *StructCache) GetLocationType()     (types.ObjectType)  { if (!cache.StructureLoaded) { cache.LoadStruct() }; return cache.Structure.LocationType }
func (cache *StructCache) GetOperatingAmbit()   (types.Ambit)       { if (!cache.StructureLoaded) { cache.LoadStruct() }; return cache.Structure.OperatingAmbit }

func (cache *StructCache) GetPlanet()   (*PlanetCache)  { if (!cache.PlanetLoaded) { cache.LoadPlanet() }; return cache.Planet }
func (cache *StructCache) GetPlanetId() (string)        { return cache.GetPlanet().GetPlanetId() }
func (cache *StructCache) GetFleet()    (*FleetCache)   { if (!cache.FleetLoaded) { cache.LoadFleet() }; return cache.Fleet }

func (cache *StructCache) GetDefenders() ([]types.StructDefender) { if (!cache.DefendersLoaded) { cache.LoadDefenders() }; return cache.Defenders }

func (cache *StructCache) GetEventAttackDetail() (*types.EventAttackDetail) { if (!cache.EventAttackDetailLoaded) { cache.EventAttackDetail = types.CreateEventAttackDetail() }; return cache.EventAttackDetail }
func (cache *StructCache) GetEventAttackShotDetail() (*types.EventAttackShotDetail) { if (!cache.EventAttackShotDetailLoaded) { cache.EventAttackShotDetail = types.CreateEventAttackShotDetail(cache.StructId) }; return cache.EventAttackShotDetail }

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

func (cache *StructCache) ResetBlockStartOreMine() {
    uctx := sdk.UnwrapSDKContext(cache.Ctx)
    cache.BlockStartOreMine = uint64(uctx.BlockHeight())
    cache.BlockStartOreMineLoaded = true
    cache.BlockStartOreMineChanged = true
}

func (cache *StructCache) ResetBlockStartOreRefine() {
    uctx := sdk.UnwrapSDKContext(cache.Ctx)
    cache.BlockStartOreRefine = uint64(uctx.BlockHeight())
    cache.BlockStartOreRefineLoaded = true
    cache.BlockStartOreRefineChanged = true
}

func (cache *StructCache) ClearBlockStartOreMine() {
    cache.BlockStartOreMine = 0
    cache.BlockStartOreMineLoaded = true
    cache.BlockStartOreMineChanged = true
}

func (cache *StructCache) ClearBlockStartOreRefine() {
    cache.BlockStartOreRefine = 0
    cache.BlockStartOreRefineLoaded = true
    cache.BlockStartOreRefineChanged = true
}

func (cache *StructCache) FlushEventAttackShotDetail() ( *types.EventAttackShotDetail) {
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

func (cache *StructCache) IsOffline() bool {
    return !cache.IsOnline()
}

func (cache *StructCache) IsHidden() bool {
   return cache.GetStatus()&types.StructStateHidden != 0
}

func (cache *StructCache) StatusAddOnline() {
    cache.Status = cache.GetStatus() | types.StructStateOnline
    cache.StatusChanged = true
}

func (cache *StructCache) StatusAddHidden() {
    cache.Status = cache.GetStatus() | types.StructStateHidden
    cache.StatusChanged = true
}

func (cache *StructCache) StatusAddDestroyed() {
    cache.Status = cache.GetStatus() | types.StructStateDestroyed
    cache.StatusChanged = true
}

func (cache *StructCache) StatusRemoveHidden() {
    if (cache.IsHidden()) {
        cache.Status = cache.Status &^ types.StructStateHidden
        cache.StatusChanged = true
    }
}

func (cache *StructCache) StatusRemoveOnline() {
    if (cache.IsOnline()) {
        cache.Status = cache.Status &^ types.StructStateOnline
        cache.StatusChanged = true
    }
}



func (cache *StructCache) IsDestroyed() bool {
   return cache.GetStatus()&types.StructStateDestroyed != 0
}

func (cache *StructCache) GridStatusAddReady() {
    cache.K.SetGridAttribute(cache.Ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_ready, cache.StructId), 1)
}

func (cache *StructCache) GridStatusRemoveReady() {
    cache.K.ClearGridAttribute(cache.Ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_ready, cache.StructId ))
}


func (cache *StructCache) ActivationReadinessCheck() (err error) {
    // Check Struct is Built
    if (!cache.IsBuilt()){
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) isn't finished being built yet", cache.StructId)
    }

    // Check Struct is Offline
    if (cache.IsOffline()){
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) is already online", cache.StructId)
    }

    // Check Player is Online
    if (cache.GetOwner().IsOffline()) {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Player (%s) is offline due to power", cache.GetOwnerId())
    }

    // Check Player Capacity
    if (!cache.GetOwner().CanSupportLoadAddition(cache.GetStructType().GetPassiveDraw())) {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Player (%s) cannot handle the new load requirements", cache.GetOwnerId())
    }

    return
}

func (cache *StructCache) GoOnline() {
    // Add to the players struct load
    cache.GetOwner().StructsLoadIncrement(cache.GetStructType().GetPassiveDraw())
    //k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, sudoPlayer.Id), structType.PassiveDraw)

    // Turn on the mining systems
    if (cache.GetStructType().HasOreMiningSystem()) {
        cache.ResetBlockStartOreMine()
    }

    // Turn on the refinery
    if (cache.GetStructType().HasOreRefiningSystem()) {
        cache.ResetBlockStartOreRefine()
    }

    // Raise the planetary shields
    if (cache.GetStructType().HasOreReserveDefensesSystem()) {
        cache.GetPlanet().PlanetaryShieldIncrement(cache.GetStructType().GetPlanetaryShieldContribution())
    }

    // TODO
    // This is the least generic/abstracted part of the code for now.
    // Prob need to clean this up down the road
    if (cache.GetStructType().HasPlanetaryDefensesSystem()) {
        switch (cache.GetStructType().GetPlanetaryDefenses()) {
            case types.TechPlanetaryDefenses_defensiveCannon:
                cache.GetPlanet().DefensiveCannonQuantityIncrement(1)
            case types.TechPlanetaryDefenses_lowOrbitBallisticInterceptorNetwork:
                cache.GetPlanet().LowOrbitBallisticsInterceptorNetworkQuantityIncrement(1)
        }
    }


    if (cache.GetStructType().HasPowerGenerationSystem()) {
        cache.GridStatusAddReady()
    }

    // Set the struct status flag to include built
    cache.StatusAddOnline()
}


func (cache *StructCache) GoOffline() {
    // Add to the players struct load
    cache.GetOwner().StructsLoadDecrement(cache.GetStructType().GetPassiveDraw())
    //k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, sudoPlayer.Id), structType.PassiveDraw)

    // Turn off the mining systems
    if (cache.GetStructType().HasOreMiningSystem()) {
        cache.ClearBlockStartOreMine()
    }

    // Turn off the refinery
    if (cache.GetStructType().HasOreRefiningSystem()) {
        cache.ClearBlockStartOreRefine()
    }

    // Lower the planetary shields
    if (cache.GetStructType().HasOreReserveDefensesSystem()) {
        cache.GetPlanet().PlanetaryShieldDecrement(cache.GetStructType().GetPlanetaryShieldContribution())
    }

    // TODO
    // This is the least generic/abstracted part of the code for now.
    // Prob need to clean this up down the road
    if (cache.GetStructType().HasPlanetaryDefensesSystem()) {
        switch (cache.GetStructType().GetPlanetaryDefenses()) {
            case types.TechPlanetaryDefenses_defensiveCannon:
                cache.GetPlanet().DefensiveCannonQuantityDecrement(1)
            case types.TechPlanetaryDefenses_lowOrbitBallisticInterceptorNetwork:
                cache.GetPlanet().LowOrbitBallisticsInterceptorNetworkQuantityDecrement(1)
        }
    }

    if (cache.GetStructType().HasPowerGenerationSystem()) {
        cache.GridStatusRemoveReady()

        // Remove all allocations
        allocations := cache.K.GetAllocationsFromSource(cache.Ctx, cache.StructId, false)
        cache.K.DestroyAllAllocations(cache.Ctx, allocations)
    }

    // Set the struct status flag to include built
    cache.StatusRemoveOnline()
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

func (cache *StructCache) CanOreMinePlanet() (error) {

    if (!cache.GetStructType().HasOreMiningSystem()) {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) has no mining system", cache.StructId)
    }

    if (cache.GetBlockStartOreMine() == 0) {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) not mining", cache.StructId)
    }

    if (cache.GetPlanet().IsComplete()) {
        return sdkerrors.Wrapf(types.ErrStructMine, "Planet (%s) is already complete. Move on bud, no work to be done here", cache.GetPlanet().GetPlanetId())
    }

    if (cache.GetPlanet().IsEmptyOfOre()) {
        return sdkerrors.Wrapf(types.ErrStructMine, "Planet (%s) is empty, nothing to mine", cache.GetPlanet().GetPlanetId())
    }

    return nil

}

func (cache *StructCache) OreMinePlanet() {
    cache.GetOwner().StoredOreIncrement(1)
    cache.GetPlanet().BuriedOreDecrement(1)

    cache.ResetBlockStartOreMine()
}


func (cache *StructCache) CanOreRefine() (error) {

    if (!cache.GetStructType().HasOreRefiningSystem()) {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) has no refining system", cache.StructId)
    }

    if (cache.GetBlockStartOreRefine() == 0) {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) not mining", cache.StructId)
    }

    if (!cache.GetOwner().HasStoredOre()) {
        return sdkerrors.Wrapf(types.ErrStructMine, "Player (%s) has no Ore to refine. Move on bud, no work to be done here", cache.GetOwner().PlayerId)
    }

    return nil

}

func (cache *StructCache) OreRefine() {

    cache.GetOwner().StoredOreDecrement(1)
    cache.GetOwner().DepositRefinedAlpha()

    cache.ResetBlockStartOreRefine()
}

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
            } else {
                if (targetStruct.IsHidden() && (targetStruct.GetOperatingAmbit() != cache.GetOperatingAmbit())) {
                    err = sdkerrors.Wrapf(types.ErrStructAction, "Target Struct (%s) is current hidden from Attacker Struct (%s)", targetStruct.StructId, cache.StructId)
                }
            }
        }
     }

     // Now that the inexpensive checks are done, lets go deeper
     if (err == nil) {
        switch (cache.GetLocationType()) {
            case types.ObjectType_planet:
                if (cache.GetPlanet().GetLocationListStart() == targetStruct.GetLocationId()) {
                    // The enemy fleet is here
                } else {
                    err = sdkerrors.Wrapf(types.ErrStructAction, "Target Struct (%s) is unreachable by Planetary Attacker Struct (%s)", targetStruct.StructId, cache.StructId)
                }

            case types.ObjectType_fleet:
                // Is the Fleet at home?
                if cache.GetFleet().IsOnStation() {
                    // If the Fleet is On Station, ensure the enemy is reachable
                    if cache.GetPlanet().GetLocationListStart() == targetStruct.GetLocationId() {
                        // The Fleet is on station, and the enemy is reachable
                        // Proceed with the intended action for the Fleet attacking the target
                    } else {
                        err = sdkerrors.Wrapf(types.ErrStructAction, "Target Struct (%s) is unreachable by from the Struct (%s) on Planet", targetStruct.StructId, cache.StructId)
                    }
                // Or is the Fleet out raiding another planet?
                } else {
                    // If the Fleet is away, first check if the target is on the same planet
                    if cache.GetFleet().GetLocationListForward() == "" && cache.GetPlanetId() == targetStruct.GetPlanetId() {
                        // Target has reached the planetary raid
                        // Proceed with the intended action for the Fleet attacking the target
                    // Otherwise check if the target is adjacent (either forward or backward)
                    } else if cache.GetFleet().GetLocationListForward() == targetStruct.GetLocationId() && cache.GetFleet().GetLocationListBackward() == targetStruct.GetLocationId() {
                        // The target is to either side of the Fleet
                        // Proceed with the intended action for the Fleet attacking the target
                    } else {
                        err = sdkerrors.Wrapf(types.ErrStructAction, "Target Struct (%s) is unreachable by Fleet Attacker Struct (%s)", targetStruct.StructId, cache.StructId)
                    }
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

    cache.GetEventAttackShotDetail().SetEvade(canEvade, cache.GetStructType().GetUnitDefenses())

    // If there has already been an successful evade then don't both evading harder
    if (!canEvade) {
        // Check for Planetary Defenses - Low Orbit Ballistic Interceptor Network
        if (attackerStruct.GetLocationType() == types.ObjectType_fleet) {

            // Is the Struct at home? Either via their fleet or on the planet directly
            if (cache.GetPlanet().GetOwnerId() == cache.GetOwnerId()) {

                // Grab the success rate for the interceptor network. If it returns an error, then the planet doesn't have it
                successRate, successRateError := cache.GetPlanet().GetLowOrbitBallisticsInterceptorNetworkSuccessRate()
                if (successRateError == nil) {

                    // Only effective is the Struct is in the Air or Space
                    if ((attackerStruct.GetOperatingAmbit() == types.Ambit_air) || (attackerStruct.GetOperatingAmbit() == types.Ambit_space)) {

                        // Only effective if the target is in the Water or on Land
                        if ((cache.GetOperatingAmbit() == types.Ambit_water) || (cache.GetOperatingAmbit() == types.Ambit_land)) {
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
    if (cache.IsDestroyed()) { return 0 }

    for shot := uint64(0); shot < attackingStruct.GetStructType().GetWeaponShots(weaponSystem); shot++ {
        if (attackingStruct.IsSuccessful(attackingStruct.GetStructType().GetWeaponShotSuccessRate(weaponSystem))) {
            damage = damage + attackingStruct.GetStructType().GetWeaponDamage(weaponSystem)
        }
    }

    cache.GetEventAttackShotDetail().SetDamageDealt(damage)

    if (damage != 0) {
        damageReduction := cache.GetStructType().GetAttackReduction()

        if (damageReduction > 0) {
            cache.GetEventAttackShotDetail().SetDamageReduction(damageReduction, cache.GetStructType().GetUnitDefenses())
        }


        if (damageReduction > damage) {
            damage = 0
        } else {
            damage = damage - damageReduction
        }
    }

    cache.GetEventAttackShotDetail().SetDamage(damage)

    if (damage != 0) {

        if (damage > cache.GetHealth()) {
            cache.Health = 0
            cache.HealthChanged = true

        } else {
            cache.Health = cache.GetHealth() - damage
            cache.HealthChanged = true
        }

        if (cache.Health == 0) {
            if (cache.Blocker) {
                cache.GetEventAttackShotDetail().SetBlockerDestroyed()
            } else {
                cache.GetEventAttackShotDetail().SetTargetDestroyed()
            }

            // destruction damage from the grave
            if (cache.GetStructType().GetPostDestructionDamage() > 0) {
                attackingStruct.TakePostDestructionDamage(cache)
            }

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

    cache.GetEventAttackDetail().SetRecoilDamage(damage, cache.IsDestroyed())
    return
}


func (cache *StructCache) TakePostDestructionDamage(attackingStruct *StructCache) (damage uint64) {
    if (cache.IsDestroyed()) { return 0 }

    damage = cache.GetStructType().GetPostDestructionDamage()

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

    cache.GetEventAttackShotDetail().SetPostDestructionDamage(damage, cache.IsDestroyed(), attackingStruct.GetStructType().GetPassiveWeaponry())

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
            // destruction damage from the grave
            if (cache.GetStructType().GetPostDestructionDamage() > 0) {
                counterStruct.TakePostDestructionDamage(cache)
            }
            cache.DestroyAndCommit()
        }

    }

    if (cache.Defender) {
        cache.GetEventAttackShotDetail().AppendDefenderCounter(counterStruct.StructId, damage, cache.IsDestroyed())
    } else {
        cache.GetEventAttackShotDetail().AppendTargetCounter(damage, cache.IsDestroyed(), counterStruct.GetStructType().GetPassiveWeaponry())
    }

    return
}


func (cache *StructCache) TakePlanetaryDefenseCanonDamage(damage uint64) (uint64) {
    if (cache.IsDestroyed()) { return 0 }

    if (damage != 0) {

        if (damage > cache.GetHealth()) {
            damage = cache.GetHealth()
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

    cache.GetEventAttackDetail().SetPlanetaryDefenseCannonDamage(damage, cache.IsDestroyed())

    return damage
}


func (cache *StructCache) AttemptBlock(attacker *StructCache, weaponSystem types.TechWeaponSystem, target *StructCache) (blocked bool) {
    if (cache.Ready && attacker.Ready) {
        if (cache.GetOperatingAmbit() == target.GetOperatingAmbit()) {
            blocked = true
            cache.Blocker = true
            cache.GetEventAttackShotDetail().SetBlocker(cache.StructId)
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
    cache.K.SetStructAttributeDecrement(cache.Ctx, GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, cache.GetOwnerId(), cache.GetStructType().GetId()),1)

    // Don't clear these now, clear them on sweeps?
    // "health":               StructAttributeType_health,
    // "status":               StructAttributeType_status,

    // It's possible the build was never complete, so clear out this attribute to be safe
    cache.K.ClearStructAttribute(cache.Ctx, cache.BlockStartBuildAttributeId)

    // Destroy mining systems
    if (cache.GetStructType().HasOreMiningSystem()) {
        cache.K.ClearStructAttribute(cache.Ctx, cache.BlockStartOreMineAttributeId)
    }

    // Turn off the refinery
    if (cache.GetStructType().HasOreRefiningSystem()) {
        cache.K.ClearStructAttribute(cache.Ctx, cache.BlockStartOreRefineAttributeId)
    }

    // Clear Defensive Relationships
    cache.K.ClearStructAttribute(cache.Ctx, cache.ProtectedStructIndexAttributeId)

    // TODO clean this up to be more function based.. but it's fine
    if (cache.GetStructType().HasPowerGenerationSystem()) {
        // Clear Load
        cache.K.ClearGridAttribute(cache.Ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, cache.StructId ))

        // Clear Capacity
        cache.K.ClearGridAttribute(cache.Ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, cache.StructId ))

        // Clear Fuel
        cache.K.ClearGridAttribute(cache.Ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, cache.StructId ))

        // Clear Power
        cache.K.ClearGridAttribute(cache.Ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, cache.StructId ))

        // Clear Allocation Pointer Start + End
        cache.K.ClearGridAttribute(cache.Ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_allocationPointerStart, cache.StructId ))
        cache.K.ClearGridAttribute(cache.Ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_allocationPointerEnd, cache.StructId ))
    }

    // Clear Permissions
    // This only clears permissions for the current owner, which is likely a problem in the future.
    permissionId := GetObjectPermissionIDBytes(cache.StructId, cache.GetOwnerId())
    cache.K.PermissionClearAll(cache.Ctx, permissionId)

    // We're not going to remove it from the location yet, that happens during sweeps

    // Set to Destroyed
    cache.StatusAddDestroyed()

    // Might need to do this manually so it doesn't undo some of the above...
    //cache.Commit()
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


    // Can the Struct be a catalyst for raid-end
        // cache.GetFleet().Defeat()
        // Check for raid win conditions
    if (cache.CanTriggerRaidDefeatByDestruction()) {
        cache.GetFleet().Defeat()
    }
}


func (cache *StructCache) CanTriggerRaidDefeatByDestruction() (bool) {
    if (!cache.GetStructType().GetTriggerRaidDefeatByDestruction()) { return false }

    // Make sure the ship isn't at home
    // This win condition only works to defeat the attacking fleet
    if (cache.GetPlanet().GetOwnerId() != cache.GetOwnerId()) {
        return true
    }
    return false
}

func (cache *StructCache) AttemptMove(destinationId string, destinationType types.ObjectType, ambit types.Ambit, slot uint64) (error) {
    if (!cache.StructureLoaded) { cache.LoadStruct() }

    if cache.IsOffline() {
        return sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct cannot move when offline")
    }

    var planet PlanetCache
    var fleet FleetCache

    switch destinationType {
        case types.ObjectType_planet:
            // Load Planet from the PlanetId
            planet = cache.K.GetPlanetCacheFromId(cache.Ctx, destinationId)
            planetFound := planet.LoadPlanet()
            if !planetFound {
                return  sdkerrors.Wrapf(types.ErrObjectNotFound, "Planet (%s) was not found. Building a Struct in a void might be tough", destinationId)
            }

            err := planet.MoveReadiness(cache, ambit, slot)
            if (err != nil) {
                return err
            }

        case types.ObjectType_fleet:
            // Load the Fleet
            fleet, _ = cache.K.GetFleetCacheFromId(cache.Ctx, destinationId)
            fleetFound := fleet.LoadFleet()
            if !fleetFound {
                return sdkerrors.Wrapf(types.ErrObjectNotFound, "Fleet (%s) was not found. Building a Struct in a void might be tough", destinationId)
            }

            err := fleet.MoveReadiness(cache, ambit, slot)
            if (err != nil) {
                return  err
            }

        default:
            return sdkerrors.Wrapf(types.ErrObjectNotFound, "We're not building these yet")
    }

    switch (cache.Structure.LocationType) {
        case types.ObjectType_planet:
            previousPlanet := cache.K.GetPlanetCacheFromId(cache.Ctx, cache.Structure.LocationId)
            previousPlanet.ClearSlot(cache.Structure.OperatingAmbit, cache.Structure.Slot)
            previousPlanet.Commit()
        case types.ObjectType_fleet:
            if (cache.GetStructType().Type != types.CommandStruct) {
                previousFleet, _ := cache.K.GetFleetCacheFromId(cache.Ctx, cache.Structure.LocationId)
                previousFleet.ClearSlot(cache.Structure.OperatingAmbit, cache.Structure.Slot)
                previousFleet.Commit()
            }
    }

   switch destinationType {
        case types.ObjectType_planet:

            cache.Structure.LocationId = destinationId
            cache.Structure.LocationType = destinationType
            cache.Structure.OperatingAmbit = ambit

            // Update the cross reference on the planet
            err := planet.SetSlot(cache.Structure)
            if (err != nil) {
                return err
            }


            // TODO, do a check to see if the old planet had uncommitted changes
            breaking change for comment above

            cache.Planet = &planet
            cache.PlanetLoaded = true
            cache.PlanetChanged = true
        case types.ObjectType_fleet:

            // Update the cross reference on the planet
            if (cache.GetStructType().Type == types.CommandStruct) {
                cache.Structure.OperatingAmbit = ambit
            } else {

                cache.Structure.LocationId = destinationId
                cache.Structure.LocationType = destinationType
                cache.Structure.OperatingAmbit = ambit
                cache.Structure.Slot = slot


                err := fleet.SetSlot(cache.Structure)
                if (err != nil) {
                    return err
                }

                // TODO, do a check to see if the old fleet had uncommitted changes
                breaking change for comment above

                cache.Fleet = &fleet
                cache.FleetLoaded = true
                cache.FleetChanged = true
            }

            cache.StructureChanged = true
    }


    return  nil
}
