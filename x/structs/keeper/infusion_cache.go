package keeper

import (
    "structs/x/structs/types"

    "cosmossdk.io/math"
)

type InfusionCache struct {
    DestinationType types.ObjectType
    DestinationId   string
    Address         string

    CC *CurrentContext

    Changed        bool
    Deleted        bool
    InfusionLoaded bool
    Infusion       types.Infusion

    DestinationFuelAttributeId     string
    DestinationCapacityAttributeId string
}

// =========================================================================
// Committable
// =========================================================================

func (cache *InfusionCache) Commit() {
    if cache.Changed {
        if cache.Deleted {
            cache.CC.k.RemoveInfusion(cache.CC.ctx, cache.DestinationId, cache.Address)
        } else {
            cache.CC.k.SetInfusion(cache.CC.ctx, cache.Infusion)
        }
        cache.Changed = false
    }

    if cache.IsEmpty() {
        cache.CC.k.AppendInfusionDestructionQueue(cache.CC.ctx, cache.GetInfusionId())
    }
}

func (cache *InfusionCache) IsChanged() bool { return cache.Changed }

func (cache *InfusionCache) ID() string {
    return cache.DestinationId + "/" + cache.Address
}

// =========================================================================
// Loading
// =========================================================================


func (cache *InfusionCache) LoadInfusion() bool {
    cache.Infusion, cache.InfusionLoaded = cache.CC.k.GetInfusion(
        cache.CC.ctx, cache.DestinationId, cache.Address,
    )

    if !cache.InfusionLoaded {
        cache.Infusion = types.Infusion{
            DestinationType: cache.DestinationType,
            DestinationId:   cache.DestinationId,
            Address:         cache.Address,
            PlayerId:        cache.GetOwnerId(),
            Commission:      math.LegacyZeroDec(),
        }
        cache.InfusionLoaded = true
    }

    return cache.InfusionLoaded
}

// =========================================================================
// Getters
// =========================================================================

func (cache *InfusionCache) GetOwnerId() string {
    if !cache.InfusionLoaded {
        return GetObjectID(types.ObjectType_player, cache.CC.GetPlayerIndexFromAddress(cache.Address))
    }
    return cache.Infusion.PlayerId
}

func (cache *InfusionCache) GetInfusion() types.Infusion { if !cache.InfusionLoaded { cache.LoadInfusion() }; return cache.Infusion }
func (cache *InfusionCache) GetInfusionId() string       { if !cache.InfusionLoaded { cache.LoadInfusion() }; return cache.DestinationId + "-" + cache.Address }
func (cache *InfusionCache) GetPower() uint64             { if !cache.InfusionLoaded { cache.LoadInfusion() }; return cache.Infusion.Power }
func (cache *InfusionCache) GetFuel() uint64              { if !cache.InfusionLoaded { cache.LoadInfusion() }; return cache.Infusion.Fuel }
func (cache *InfusionCache) GetDefusing() uint64          { if !cache.InfusionLoaded { cache.LoadInfusion() }; return cache.Infusion.Defusing }

func (cache *InfusionCache) GetOwner() *PlayerCache {
    player, _ := cache.CC.GetPlayer(cache.GetOwnerId())
    return player
}

func (cache *InfusionCache) IsEmpty() bool {
    if !cache.InfusionLoaded { cache.LoadInfusion() }

    return cache.Infusion.Power == 0 && cache.Infusion.Defusing == 0
}

// =========================================================================
// Internal: apply grid deltas + propagation
// =========================================================================

// applyGridDeltas applies the difference between old and new contributions
// to the three affected grid attributes, and propagates auto-resize/cascade.
// This mirrors how AllocationCache.SetPower handles grid changes inline.
func (cache *InfusionCache) applyGridDeltas(oldFuel, oldDestCap, oldPlayerCap uint64) {
    newFuel := cache.Infusion.Fuel
    _, newDestCap, newPlayerCap := cache.Infusion.GetPowerDistribution()

    // Destination fuel
    if oldFuel != newFuel {
        cache.CC.SetGridAttributeDelta(cache.DestinationFuelAttributeId, oldFuel, newFuel)
    }

    // Destination capacity
    if oldDestCap != newDestCap {
        cache.CC.SetGridAttributeDelta(cache.DestinationCapacityAttributeId, oldDestCap, newDestCap)

        destAllocId, found := cache.CC.k.GetAutoResizeAllocationBySource(cache.CC.ctx, cache.DestinationId)
        if found {
            totalCap := cache.CC.GetGridAttribute(cache.DestinationCapacityAttributeId)
            cache.CC.AutoResizeAllocation(destAllocId, totalCap)
        } else if newDestCap < oldDestCap {
            cache.CC.k.AppendGridCascadeQueue(cache.CC.ctx, cache.DestinationId)
        }
    }

    // Player capacity
    if oldPlayerCap != newPlayerCap {
        playerId := cache.GetOwnerId()
        playerCapAttrId := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, playerId)
        cache.CC.SetGridAttributeDelta(playerCapAttrId, oldPlayerCap, newPlayerCap)

        playerAllocId, found := cache.CC.k.GetAutoResizeAllocationBySource(cache.CC.ctx, playerId)
        if found {
            totalCap := cache.CC.GetGridAttribute(playerCapAttrId)
            cache.CC.AutoResizeAllocation(playerAllocId, totalCap)
        } else if newPlayerCap < oldPlayerCap {
            cache.CC.k.AppendGridCascadeQueue(cache.CC.ctx, playerId)
        }
    }
}

// =========================================================================
// Setters
// =========================================================================

func (cache *InfusionCache) SetRatio(ratio uint64) {
    if !cache.InfusionLoaded { cache.LoadInfusion() }

    oldFuel := cache.Infusion.Fuel
    _, oldDestCap, oldPlayerCap := cache.Infusion.GetPowerDistribution()

    cache.Infusion.Ratio = ratio
    cache.Infusion.Recalculate()

    cache.applyGridDeltas(oldFuel, oldDestCap, oldPlayerCap)
    cache.Changed = true
}

func (cache *InfusionCache) SetCommission(commission math.LegacyDec) {
    if !cache.InfusionLoaded { cache.LoadInfusion() }

    oldFuel := cache.Infusion.Fuel
    _, oldDestCap, oldPlayerCap := cache.Infusion.GetPowerDistribution()

    cache.Infusion.Commission = commission

    cache.applyGridDeltas(oldFuel, oldDestCap, oldPlayerCap)
    cache.Changed = true
}

func (cache *InfusionCache) SetFuel(fuel uint64) {
    if !cache.InfusionLoaded { cache.LoadInfusion() }

    oldFuel := cache.Infusion.Fuel
    _, oldDestCap, oldPlayerCap := cache.Infusion.GetPowerDistribution()

    cache.Infusion.Fuel = fuel
    cache.Infusion.Recalculate()

    cache.applyGridDeltas(oldFuel, oldDestCap, oldPlayerCap)
    cache.Changed = true
}

func (cache *InfusionCache) AddFuel(additionalFuel uint64) {
    if !cache.InfusionLoaded { cache.LoadInfusion() }

    oldFuel := cache.Infusion.Fuel
    _, oldDestCap, oldPlayerCap := cache.Infusion.GetPowerDistribution()

    cache.Infusion.Fuel += additionalFuel
    cache.Infusion.Recalculate()

    cache.applyGridDeltas(oldFuel, oldDestCap, oldPlayerCap)
    cache.Changed = true
}

func (cache *InfusionCache) SetFuelAndCommission(fuel uint64, commission math.LegacyDec) {
    if !cache.InfusionLoaded { cache.LoadInfusion() }

    oldFuel := cache.Infusion.Fuel
    _, oldDestCap, oldPlayerCap := cache.Infusion.GetPowerDistribution()

    cache.Infusion.Fuel = fuel
    cache.Infusion.Commission = commission
    cache.Infusion.Recalculate()

    cache.applyGridDeltas(oldFuel, oldDestCap, oldPlayerCap)
    cache.Changed = true
}

func (cache *InfusionCache) SetDefusing(defusing uint64) {
    if !cache.InfusionLoaded { cache.LoadInfusion() }

    cache.Infusion.Defusing = defusing
    cache.Changed = true
}

func (cache *InfusionCache) Destroy() {
    if !cache.InfusionLoaded { cache.LoadInfusion() }

    // Capture current contributions
    oldFuel := cache.Infusion.Fuel
    _, oldDestCap, oldPlayerCap := cache.Infusion.GetPowerDistribution()

    // Zero everything out
    cache.Infusion.Fuel = 0
    cache.Infusion.Power = 0
    cache.Infusion.Defusing = 0

    // Apply grid deltas (old → 0 for all three attributes)
    cache.applyGridDeltas(oldFuel, oldDestCap, oldPlayerCap)

    // Remove the infusion record
    cache.Deleted = true
    cache.Changed = true
}