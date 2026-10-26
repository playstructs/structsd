package keeper

import (
    "structs/x/structs/types"

    "cosmossdk.io/math"
)

type InfusionCache struct {
    InfusionId      string
    DestinationId   string
    Address         string

    CC *CurrentContext

    Changed        bool
    Deleted        bool
    InfusionLoaded bool
    Infusion       types.Infusion
    originalOwnerId string
    originalOwnerKnown bool
    cachedOwnerId string

    DestinationFuelAttributeId     string
    DestinationCapacityAttributeId string
}

// =========================================================================
// Committable
// =========================================================================

func (cache *InfusionCache) Commit() {
    if cache.Changed {
        if cache.Deleted {
            if cache.originalOwnerKnown {
                cache.CC.k.removeInfusionWithOwner(cache.CC.ctx, cache.DestinationId, cache.Address, cache.originalOwnerId)
            } else {
                cache.CC.k.RemoveInfusion(cache.CC.ctx, cache.DestinationId, cache.Address)
            }
        } else {
            if cache.originalOwnerKnown {
                cache.CC.k.setInfusionWithPreviousOwner(cache.CC.ctx, cache.Infusion, cache.originalOwnerId)
            } else {
                cache.CC.k.SetInfusion(cache.CC.ctx, cache.Infusion)
            }
        }
        cache.originalOwnerId = cache.Infusion.AssetOwnerID()
        if cache.Deleted { cache.originalOwnerId = "" }
        cache.originalOwnerKnown = true
        cache.Changed = false
    }

    // Only a live, still-present, now-empty record should be queued for
    // reclamation. Without the guards a just-deleted infusion re-enqueues
    // itself (its zero-value cache reads IsEmpty), and a phantom cache from a
    // probe of a nonexistent id (disownInfusion, ReactorInfusionDelegationRemoved)
    // enqueues a row that the destruction sweep can never clear, growing
    // EndBlock work forever.
    if !cache.Deleted && cache.CheckInfusion() == nil && cache.IsEmpty() {
        cache.CC.k.AppendInfusionDestructionQueue(cache.CC.ctx, cache.GetInfusionId())
    }
}

func (cache *InfusionCache) IsChanged() bool { return cache.Changed }

func (cache *InfusionCache) ID() string {
    return cache.InfusionId
}

// =========================================================================
// Loading
// =========================================================================


func (cache *InfusionCache) LoadInfusion() bool {
    cache.Infusion, cache.InfusionLoaded = cache.CC.k.GetInfusion(
        cache.CC.ctx, cache.DestinationId, cache.Address,
    )
    cache.originalOwnerId = cache.Infusion.AssetOwnerID()
    cache.originalOwnerKnown = true
    cache.indexCachedOwner()

    return cache.InfusionLoaded
}

// =========================================================================
// Getters
// =========================================================================

func (cache *InfusionCache) GetOwnerId() string {
    if !cache.InfusionLoaded { cache.LoadInfusion() }
    return cache.Infusion.AssetOwnerID()
}

func (cache *InfusionCache) CheckInfusion() error  {
    if !cache.InfusionLoaded {
        if !cache.LoadInfusion() {
            return types.NewObjectNotFoundError("infusion", cache.InfusionId)
        }
    }
    return nil
}

func (cache *InfusionCache) GetInfusion() types.Infusion { if !cache.InfusionLoaded { cache.LoadInfusion() }; return cache.Infusion }
func (cache *InfusionCache) GetInfusionId() string       { if !cache.InfusionLoaded { cache.LoadInfusion() }; return cache.DestinationId + "-" + cache.Address }
func (cache *InfusionCache) GetPower() uint64             { if !cache.InfusionLoaded { cache.LoadInfusion() }; return cache.Infusion.Power }
func (cache *InfusionCache) GetFuel() uint64              { if !cache.InfusionLoaded { cache.LoadInfusion() }; return cache.Infusion.Fuel }
func (cache *InfusionCache) GetDefusing() uint64          { if !cache.InfusionLoaded { cache.LoadInfusion() }; return cache.Infusion.Defusing }

func (cache *InfusionCache) GetOwner() (AssetOwner, error) {
    return cache.CC.GetExistingAssetOwner(cache.GetOwnerId())
}

// IsEmpty reports whether nothing of value remains in the infusion, making it
// safe to reclaim. Fuel is part of the test because Power alone can be zeroed
// while the underlying stake is still live: a reactor whose validator is jailed
// runs at a zero energy ratio, which drives Power to zero without touching the
// delegation. Reclaiming on Power alone would delete that record and strand the
// delegator's fuel.
func (cache *InfusionCache) IsEmpty() bool {
    if !cache.InfusionLoaded { cache.LoadInfusion() }

    return cache.Infusion.Fuel == 0 && cache.Infusion.Power == 0 && cache.Infusion.Defusing == 0
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
            found = cache.CC.AutoResizeAllocation(destAllocId, totalCap)

            // The hook named an allocation that is gone, so nothing is tracking
            // this capacity. Drop the hook and treat it as absent, or a decrease
            // silently skips the cascade below and leaves load unshed.
            if !found {
                cache.CC.k.logger.Error("Stale auto-resize hook on infusion destination", "destinationId", cache.DestinationId, "allocationId", destAllocId)
                cache.CC.k.ClearAutoResizeAllocationBySource(cache.CC.ctx, cache.DestinationId)
            }
        }

        if !found && newDestCap < oldDestCap {
            cache.CC.k.AppendGridCascadeQueue(cache.CC.ctx, cache.DestinationId)
        }
    }

    // Player capacity
    cache.applyPlayerCapacityDelta(cache.GetOwnerId(), oldPlayerCap, newPlayerCap)
}

// applyPlayerCapacityDelta moves one player's share of an infusion's power.
//
// Split out of applyGridDeltas because SetPlayerId needs the same work against
// two different players, withdrawing from the outgoing owner and crediting the
// incoming one. Keeping it in one place is what stops a re-home from skipping
// the auto-resize hook and the cascade the losing player is owed.
func (cache *InfusionCache) applyPlayerCapacityDelta(playerId string, oldPlayerCap uint64, newPlayerCap uint64) {
    if oldPlayerCap == newPlayerCap {
        return
    }

    if cache.CC.k.ReactorDiversityReady(cache.CC.ctx) {
        reactor := ""
        if cache.Infusion.DestinationType == types.ObjectType_reactor { reactor = cache.DestinationId }
        cache.CC.ApplyOwnerInfusionCapacityDelta(playerId, reactor, oldPlayerCap, newPlayerCap)
        return
    }

    playerCapAttrId := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, playerId)
    cache.CC.SetGridAttributeDelta(playerCapAttrId, oldPlayerCap, newPlayerCap)

    playerAllocId, found := cache.CC.k.GetAutoResizeAllocationBySource(cache.CC.ctx, playerId)
    if found {
        totalCap := cache.CC.GetGridAttribute(playerCapAttrId)
        found = cache.CC.AutoResizeAllocation(playerAllocId, totalCap)

        // See applyGridDeltas: a hook pointing at a destroyed allocation is no hook.
        if !found {
            cache.CC.k.logger.Error("Stale auto-resize hook on player", "playerId", playerId, "allocationId", playerAllocId)
            cache.CC.k.ClearAutoResizeAllocationBySource(cache.CC.ctx, playerId)
        }
    }

    if !found && newPlayerCap < oldPlayerCap {
        cache.CC.k.AppendGridCascadeQueue(cache.CC.ctx, playerId)
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

// SetPlayerId re-homes an infusion to the player that now owns its address, and
// moves the capacity it contributes along with it.
//
// An address can change hands: AddressRevoke clears the address index and
// AddressRegister will bind that address to a different player on a key proof.
// The infusion is keyed by (destination, address) and survives all of that, so
// without this its PlayerId stays pinned to whoever held the address first.
// Whoever controls the address controls the stake behind the infusion, so the
// capacity belongs to the current owner and the record has to follow the index.
//
// Ownership only affects the delegator's own share. The destination's fuel and
// commission capacity are keyed by destination, not by player, so they are
// deliberately untouched here.
func (cache *InfusionCache) SetPlayerId(playerId string) {
    cache.setInfusionOwner(playerId, "")
}

// SetOwnerId resolves an existing player or guild before moving its capacity.
func (cache *InfusionCache) SetOwnerId(ownerId string) error {
    if err := cache.CheckInfusion(); err != nil { return err }
    if _, err := cache.CC.GetExistingAssetOwner(ownerId); err != nil { return err }
    kind, _, _ := types.ParseAssetOwnerID(ownerId)
    if kind == types.ObjectType_player {
        cache.setInfusionOwner(ownerId, "")
    } else {
        cache.setInfusionOwner("", ownerId)
    }
    return nil
}

func (cache *InfusionCache) setInfusionOwner(playerId, ownerId string) {
    if !cache.InfusionLoaded { cache.LoadInfusion() }

    previousPlayerId := cache.Infusion.AssetOwnerID()
    nextOwnerId := ownerId
    if nextOwnerId == "" { nextOwnerId = playerId }
    if previousPlayerId == nextOwnerId {
        return
    }

    _, _, playerPower := cache.Infusion.GetPowerDistribution()

    // Withdraw before rewriting the field, or the credit below lands on the
    // player we were meant to take it from.
    if previousPlayerId != "" {
        cache.applyPlayerCapacityDelta(previousPlayerId, playerPower, 0)
    }

    cache.Infusion.PlayerId = playerId
    cache.Infusion.OwnerId = ownerId
    cache.indexCachedOwner()

    if nextOwnerId != "" {
        cache.applyPlayerCapacityDelta(nextOwnerId, 0, playerPower)
    }

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
    cache.indexCachedOwner()
    cache.Changed = true
}

func (cache *InfusionCache) CanAllocateAsSourceBy(activePlayer *PlayerCache) error {
    return types.NewAllocationError(cache.ID(), "unacceptable_source")
}
