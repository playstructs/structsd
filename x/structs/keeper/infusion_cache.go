package keeper

import (
	"context"
    "structs/x/structs/types"

    "cosmossdk.io/math"

)


type InfusionCache struct {
    DestinationType types.ObjectType
    DestinationId string
    Address string

    K *Keeper
    Ctx context.Context

    AnyChange bool
    Ready bool

    InfusionLoaded  bool
    InfusionChanged bool
    Infusion  types.Infusion

    InfusionSnapshot types.Infusion

    OwnerLoaded bool
    Owner *PlayerCache

    DestinationFuelAttributeId string
    DestinationFuelLoaded bool
    DestinationFuelChanged bool
    DestinationFuel uint64

    DestinationCapacityAttributeId string
    DestinationCapacityLoaded bool
    DestinationCapacityChanged bool
    DestinationCapacity uint64

    PlayerCapacityAttributeId string
    PlayerCapacityLoaded bool
    PlayerCapacityChanged bool
    PlayerCapacity uint64

}

func (k *Keeper) GetInfusionCache(ctx context.Context, destinationType types.ObjectType, destinationId string, address string) (InfusionCache) {
    return InfusionCache{
        DestinationType: destinationType,
        DestinationId: destinationId,
        Address: address,

        K: k,
        Ctx: ctx,

        AnyChange: false,

        InfusionLoaded: false,
        InfusionChanged: false,
        OwnerLoaded: false,

        DestinationFuelAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, destinationId),
        DestinationFuelLoaded: false,
        DestinationFuelChanged: false,

        DestinationCapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, destinationId),
        DestinationCapacityLoaded: false,
        DestinationCapacityChanged: false,

        //PlayerCapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, destinationId),
        PlayerCapacityLoaded: false,
        PlayerCapacityChanged: false,

    }
}

func (cache *InfusionCache) Commit() () {
    cache.AnyChange = false

    cache.K.logger.Info("Updating Infusion From Cache","destinationId",cache.DestinationId,"address",cache.Address)

    if (cache.InfusionChanged) {
        cache.K.SetInfusion(cache.Ctx, cache.Infusion)
        cache.InfusionChanged = false
    }

    if (cache.Owner != nil && cache.GetOwner().IsChanged()) {
        cache.GetOwner().Commit()
    }

    if (cache.DestinationFuelChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.DestinationFuelAttributeId, cache.DestinationFuel)
        cache.DestinationFuelChanged = false
    }

    if (cache.DestinationCapacityChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.DestinationCapacityAttributeId, cache.DestinationCapacity)

        // Further propagate these changes into the Allocation system
        destinationAllocationId, destinationAutoResizeAllocationFound := cache.K.GetAutoResizeAllocationBySource(cache.Ctx, cache.DestinationId)
        if (destinationAutoResizeAllocationFound) {
            cache.K.AutoResizeAllocation(cache.Ctx, destinationAllocationId, cache.DestinationId, cache.GetSnapshotDestinationCapacity(), cache.GetDestinationCapacity())
        } else {
            if cache.GetSnapshotDestinationCapacity() > cache.GetDestinationCapacity() {
                cache.K.AppendGridCascadeQueue(cache.Ctx, cache.DestinationId)
            }
        }

        cache.DestinationCapacityChanged = false
    }

    if (cache.PlayerCapacityChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.PlayerCapacityAttributeId, cache.PlayerCapacity)

        // Further propagate these changes into the Allocation system
        playerAllocationId, playerAutoResizeAllocationFound := cache.K.GetAutoResizeAllocationBySource(cache.Ctx, cache.GetOwnerId())
        if (playerAutoResizeAllocationFound) {
            cache.K.AutoResizeAllocation(cache.Ctx, playerAllocationId, cache.GetOwnerId(), cache.GetSnapshotPlayerCapacity(), cache.GetPlayerCapacity())
        } else {
            if cache.GetSnapshotPlayerCapacity() > cache.GetPlayerCapacity() {
                cache.K.AppendGridCascadeQueue(cache.Ctx, cache.GetOwnerId())
            }
        }

        cache.PlayerCapacityChanged = false
    }

    cache.Snapshot()
}

func (cache *InfusionCache) IsChanged() bool {
    return cache.AnyChange
}

func (cache *InfusionCache) Changed() {
    cache.AnyChange = true
}

// Load the core Infusion data
func (cache *InfusionCache) LoadInfusion() (bool) {
    cache.Infusion, cache.InfusionLoaded = cache.K.GetInfusion(cache.Ctx, cache.DestinationId, cache.Address)

    // Replacing the old Upsert methodology with this Load-or-Create
    if !cache.InfusionLoaded {

        cache.Infusion = types.Infusion{
            DestinationType: cache.DestinationType,
            DestinationId: cache.DestinationId,
            Address: cache.Address,
            PlayerId: cache.GetOwnerId(),
            Commission: math.LegacyZeroDec(),
            Fuel: 0,
            Power: 0,
            Ratio: 0,
            Defusing: 0,
        }

        cache.InfusionLoaded = true
    }

    cache.Snapshot()

    return cache.InfusionLoaded
}


// Load the Player data
func (cache *InfusionCache) LoadOwner() (bool) {
    newOwner, _ := cache.K.GetPlayerCacheFromId(cache.Ctx, cache.GetOwnerId())
    cache.Owner = &newOwner
    cache.OwnerLoaded = true
    return cache.OwnerLoaded
}


func (cache *InfusionCache) LoadDestinationFuel() {
    cache.DestinationFuel = cache.K.GetGridAttribute(cache.Ctx, cache.DestinationFuelAttributeId)
    cache.DestinationFuelLoaded = true
}

func (cache *InfusionCache) LoadDestinationCapacity() {
    cache.DestinationCapacity = cache.K.GetGridAttribute(cache.Ctx, cache.DestinationCapacityAttributeId)
    cache.DestinationCapacityLoaded = true
}

func (cache *InfusionCache) LoadPlayerCapacity() {
    // We don't have the PlayerId during object creation
    // So we need to set the AttributeId here
    if cache.PlayerCapacityAttributeId == "" {
        cache.PlayerCapacityAttributeId = GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, cache.GetOwnerId())
    }

    cache.PlayerCapacity = cache.K.GetGridAttribute(cache.Ctx, cache.PlayerCapacityAttributeId)
    cache.PlayerCapacityLoaded = true
}

/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

func (cache *InfusionCache) GetOwnerId() (string) {
    if (!cache.InfusionLoaded) {
        // If the Infusion isn't loaded yet, we can get the playerId the old fashion way.
        return GetObjectID(types.ObjectType_player, cache.K.GetPlayerIndexFromAddress(cache.Ctx, cache.Address))
    }
    return cache.Infusion.PlayerId
}

func (cache *InfusionCache) GetOwner()          (*PlayerCache) { if (!cache.OwnerLoaded) { cache.LoadOwner() }; return cache.Owner }
func (cache *InfusionCache) GetInfusion()       (types.Infusion) { if (!cache.InfusionLoaded) { cache.LoadInfusion() }; return cache.Infusion }

func (cache *InfusionCache) GetDestinationFuel()        (uint64) { if (!cache.DestinationFuelLoaded) { cache.LoadDestinationFuel() }; return cache.DestinationFuel }
func (cache *InfusionCache) GetDestinationCapacity()    (uint64) { if (!cache.DestinationCapacityLoaded) { cache.LoadDestinationCapacity() }; return cache.DestinationCapacity }
func (cache *InfusionCache) GetPlayerCapacity()         (uint64) { if (!cache.PlayerCapacityLoaded) { cache.LoadPlayerCapacity() }; return cache.PlayerCapacity }

func (cache *InfusionCache) GetFuel()                   (uint64) { if (!cache.InfusionLoaded) { cache.LoadInfusion() }; return cache.Infusion.Fuel }
func (cache *InfusionCache) GetPower()                  (uint64) { if (!cache.InfusionLoaded) { cache.LoadInfusion() }; return cache.Infusion.Power }
func (cache *InfusionCache) GetSnapshotFuel()           (uint64) { if (!cache.InfusionLoaded) { cache.LoadInfusion() }; return cache.InfusionSnapshot.Fuel }
func (cache *InfusionCache) GetSnapshotPower()          (uint64) { if (!cache.InfusionLoaded) { cache.LoadInfusion() }; return cache.InfusionSnapshot.Power }


func (cache *InfusionCache) GetPendingPlayerCapacity() uint64 {
    if (!cache.InfusionLoaded) {
        cache.LoadInfusion()
    }
    return cache.GetPower() - cache.GetPendingDestinationCapacity()
}

func (cache *InfusionCache) GetPendingDestinationCapacity() uint64 {
    if (!cache.InfusionLoaded) { cache.LoadInfusion() }
    return cache.Infusion.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(cache.GetPower()))).RoundInt().Uint64()
}

func (cache *InfusionCache) GetSnapshotPlayerCapacity() (uint64) {
    if (!cache.InfusionLoaded) {
        cache.LoadInfusion()
    }
    return cache.GetSnapshotPower() - cache.GetSnapshotDestinationCapacity()
}

func (cache *InfusionCache) GetSnapshotDestinationCapacity() uint64 {
    if (!cache.InfusionLoaded) { cache.LoadInfusion() }
    return cache.InfusionSnapshot.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(cache.GetSnapshotPower()))).RoundInt().Uint64()
}



/* Setters - SET DOES NOT COMMIT()
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */


func (cache *InfusionCache) Snapshot() {
    cache.InfusionSnapshot = cache.Infusion
}

func (cache *InfusionCache) SetCalculatedPower() {
    cache.Infusion.Power = cache.Infusion.Ratio * cache.Infusion.Fuel
}

func (cache *InfusionCache) SetCalculatedDestinationFuel() {
    // Update the Fuel record on the Destination
    if cache.GetSnapshotFuel() != cache.GetFuel() {

        var resetAmount uint64
        if cache.GetSnapshotFuel() < cache.GetDestinationFuel() {
            resetAmount = cache.GetDestinationFuel() - cache.GetSnapshotFuel()
        }

        cache.DestinationFuel = resetAmount + cache.GetFuel()
        cache.DestinationFuelChanged = true
    }
}

func (cache *InfusionCache) SetCalculatedDestinationCapacity() {
    // Update the Capacity record on the Destination
    if cache.GetSnapshotDestinationCapacity() != cache.GetPendingDestinationCapacity() {

        var resetAmount uint64
        if cache.GetSnapshotDestinationCapacity() < cache.GetDestinationCapacity() {
            resetAmount = cache.GetDestinationCapacity() - cache.GetSnapshotDestinationCapacity()
        }

        cache.DestinationCapacity = resetAmount + cache.GetPendingDestinationCapacity()
        cache.DestinationCapacityChanged = true
    }
}


func (cache *InfusionCache) SetCalculatedPlayerCapacity() {
    // Update the Capacity record on the Player
    if cache.GetSnapshotPlayerCapacity() != cache.GetPendingPlayerCapacity() {

        var resetAmount uint64
        if cache.GetSnapshotPlayerCapacity() < cache.GetPlayerCapacity() {
            resetAmount = cache.GetPlayerCapacity() - cache.GetSnapshotPlayerCapacity()
        }

        cache.PlayerCapacity = resetAmount + cache.GetPendingPlayerCapacity()
        cache.PlayerCapacityChanged = true
    }
}

func (cache *InfusionCache) SetRatio(ratio uint64) {
    if (!cache.InfusionLoaded) { cache.LoadInfusion() }

    cache.Infusion.Ratio = ratio

    cache.SetCalculatedPower()
    cache.SetCalculatedDestinationCapacity()
    cache.SetCalculatedPlayerCapacity()

    cache.InfusionChanged = true
    cache.Changed()

}

func (cache *InfusionCache) SetCommission(commission math.LegacyDec) {
    if (!cache.InfusionLoaded) { cache.LoadInfusion() }

    cache.Infusion.Commission = commission

    cache.SetCalculatedDestinationCapacity()
    cache.SetCalculatedPlayerCapacity()

    cache.InfusionChanged = true
    cache.Changed()
}

func (cache *InfusionCache) SetFuel(fuel uint64) () {
    if (!cache.InfusionLoaded) { cache.LoadInfusion() }

    cache.Infusion.Fuel = fuel

    cache.SetCalculatedPower()
    cache.SetCalculatedDestinationCapacity()
    cache.SetCalculatedPlayerCapacity()

    cache.InfusionChanged = true
    cache.Changed()
}


func (cache *InfusionCache) AddFuel(additionalFuel uint64) () {
    if (!cache.InfusionLoaded) { cache.LoadInfusion() }

    cache.Infusion.Fuel = cache.Infusion.Fuel + additionalFuel

    cache.SetCalculatedPower()
    cache.SetCalculatedDestinationCapacity()
    cache.SetCalculatedPlayerCapacity()

    cache.InfusionChanged = true
    cache.Changed()
}

func (cache *InfusionCache) SetFuelAndCommission(fuel uint64, commission math.LegacyDec) () {
    if (!cache.InfusionLoaded) { cache.LoadInfusion() }

    cache.Infusion.Fuel = fuel
    cache.Infusion.Commission = commission

    cache.SetCalculatedPower()
    cache.SetCalculatedDestinationCapacity()
    cache.SetCalculatedPlayerCapacity()

    cache.InfusionChanged = true
    cache.Changed()
}

func (cache *InfusionCache) SetDefusing(defusing uint64) () {
    if (!cache.InfusionLoaded) { cache.LoadInfusion() }

    cache.Infusion.Defusing = defusing

    cache.InfusionChanged = true
    cache.Changed()
}

// Set the Owner data manually
func (cache *InfusionCache) ManualLoadOwner(owner *PlayerCache) {
    cache.Owner = owner
    cache.OwnerLoaded = true
}


