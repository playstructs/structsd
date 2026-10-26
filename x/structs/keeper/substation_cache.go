package keeper

import (

	//sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)


type SubstationCache struct {
    SubstationId string
    CC  *CurrentContext

    Ready bool
    Deleted bool
    Changed bool

    SubstationLoaded  bool
    Substation  types.Substation

    LoadAttributeId string
    CapacityAttributeId string
    ConnectionCountAttributeId string
    ConnectionCapacityAttributeId string

}



func (cache *SubstationCache) Commit() () {
    if cache.Changed {
        cache.CC.k.logger.Info("Updating Substation From Cache","substationId", cache.SubstationId)

        if cache.Deleted {
            cache.CC.k.ClearSubstation(cache.CC.ctx, cache.SubstationId)
        } else {
            cache.CC.k.SetSubstation(cache.CC.ctx, cache.Substation)
        }
        cache.Changed = false
    }

}

func (cache *SubstationCache) IsChanged() bool {
    return cache.Changed
}

func (cache *SubstationCache) ID() string {
    return cache.SubstationId
}


/* Separate Loading functions for each of the underlying containers */

// Load the core Substation data
func (cache *SubstationCache) LoadSubstation() (bool) {
    cache.Substation, cache.SubstationLoaded = cache.CC.k.GetSubstation(cache.CC.ctx, cache.SubstationId)
    return cache.SubstationLoaded
}

func (cache *SubstationCache) CheckSubstation() (error) {
    if (!cache.SubstationLoaded) {
        if !cache.LoadSubstation() {
           return types.NewObjectNotFoundError("substation", cache.SubstationId)
        }
    }
    return nil
}


/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

func (cache *SubstationCache) GetSubstation()   (types.Substation)  { if (!cache.SubstationLoaded) { cache.LoadSubstation() }; return cache.Substation }
func (cache *SubstationCache) GetSubstationId() (string)            { return cache.SubstationId }

func (cache *SubstationCache) GetOwnerId()      (string)            { if (!cache.SubstationLoaded) { cache.LoadSubstation() }; return cache.Substation.Owner }

func (cache *SubstationCache) GetConnectionCapacity() (uint64) {
    if cache.IsDeletionPending() { return 0 }
    return cache.CC.GetGridAttribute(cache.ConnectionCapacityAttributeId)
}

func (cache *SubstationCache) GetConnectionCount() (uint64) {
    return cache.CC.GetGridAttribute(cache.ConnectionCountAttributeId)
}

func (cache *SubstationCache) GetCapacity() (uint64) {
    return cache.CC.GetGridAttribute(cache.CapacityAttributeId)
}

// GetUsableCapacity excludes the station's fixed 10% operating consumption.
// GetCapacity and capacity grid events continue to report raw incoming power.
func (cache *SubstationCache) GetUsableCapacity() uint64 {
    return cache.CC.GetEffectiveGridCapacity(cache.SubstationId)
}

func (cache *SubstationCache) GetLoad() (uint64) {
    return cache.CC.GetGridAttribute(cache.LoadAttributeId)
}

// GetAvailableCapacity reports the power this substation still has to give.
//
// Load is allowed to exceed capacity: an allocation feeding a substation can
// shrink or vanish mid-block, and only GridCascade in the EndBlocker forces load
// back under capacity — later still if it cannot shed enough and gives up. In
// that window a plain capacity - load wraps to nearly 2^64, and the gates built
// on this read it as unlimited headroom rather than none, so an over-subscribed
// substation would sell power it does not have and deepen the brownout it is
// already in. Clamping matches PlayerCache.GetAvailableCapacity and the
// comparisons SetInitialPower and SetDynamicPower make before subtracting.
// Grid accounting owns available capacity; bounded GridCascade propagates
// upstream losses. Incoming contribution count cannot turn positive capacity off.
func (cache *SubstationCache) GetAvailableCapacity() (uint64) {
    if cache.IsDeletionPending() { return 0 }
    capacity := cache.GetUsableCapacity()
    load := cache.GetLoad()

    if load >= capacity {
        return 0
    }

    return capacity - load
}

func (cache *SubstationCache) GetOwner() (*PlayerCache) {
    owner, err := cache.CC.GetExistingAssetOwner(cache.GetOwnerId())
    if err != nil {
        return nil
    }
    player, _ := owner.(*PlayerCache)
    return player
}

// IsDeletionPending is persisted so closing survives restart and export.
func (cache *SubstationCache) IsDeletionPending() bool {
    return cache.GetSubstation().DeletionPending
}

func (cache *SubstationCache) CheckAcceptingConnections() error {
    if err := cache.CheckSubstation(); err != nil { return err }
    if cache.Deleted || cache.IsDeletionPending() {
        return types.NewAllocationError(cache.ID(), "substation_deleting")
    }
    return nil
}

func (cache *SubstationCache) Delete(migrationSubstationId string) {
    if cache.Deleted { return }
    if !cache.IsDeletionPending() {
        cache.Substation.DeletionPending = true
        cache.Substation.DeletionMigrationSubstationId = migrationSubstationId
        cache.Changed = true
        cache.CC.SetGridAttribute(cache.ConnectionCapacityAttributeId, 0)
    }
    // Keep small deletes immediate, but never charge a transaction for an
    // attacker-sized incoming prefix. The existing persistent FIFO finishes it.
    _, done := cache.ProcessDeletion(8)
    if !done { cache.CC.k.AppendGridCascadeQueue(cache.CC.ctx, cache.ID()) }
}

// ProcessDeletion charges each raw index row against the caller's shared budget.
// Index updates are immediate; records remain in this CurrentContext until commit.
func (cache *SubstationCache) ProcessDeletion(limit int) (used int, done bool) {
    if cache.Deleted { return 0, true }
    if !cache.IsDeletionPending() { return 0, false }
    ids, more := cache.CC.k.GetPlayerIdsBySubstationIndexUpTo(cache.CC.ctx, cache.ID(), limit)
    for _, id := range ids {
        used++
        player, err := cache.CC.GetExistingPlayer(id)
        if err != nil || player.GetSubstationId() != cache.ID() {
            cache.CC.k.RemoveSubstationPlayerIndex(cache.CC.ctx, cache.ID(), id)
            continue
        }
        target := cache.Substation.DeletionMigrationSubstationId
        if target == cache.ID() { target = "" }
        player.MigrateSubstation(target)
    }
    if more { return used, false }
    for _, incoming := range []bool{false, true} {
        if incoming {
            ids, more = cache.CC.k.GetAllocationIdsByDestinationIndexUpTo(cache.CC.ctx, cache.ID(), limit-used)
        } else {
            ids, more = cache.CC.k.GetAllocationIdsBySourceIndexUpTo(cache.CC.ctx, cache.ID(), limit-used)
        }
        for _, id := range ids {
            used++
            allocation, found := cache.CC.GetAllocation(id)
            if !found || (incoming && allocation.GetAllocation().DestinationId != cache.ID()) || (!incoming && allocation.GetSourceId() != cache.ID()) {
                if incoming { cache.CC.k.RemoveAllocationDestinationIndex(cache.CC.ctx, cache.ID(), id) } else { cache.CC.k.RemoveAllocationSourceIndex(cache.CC.ctx, cache.ID(), id) }
                continue
            }
            var err error
            // A consumer deleting its receiving station cannot claim the
            // provider's cancellation penalty. Only detach that incoming edge.
            if incoming && allocation.IsProviderAgreement() { err = allocation.SetDestination("") } else { err = allocation.Destroy() }
            if err != nil {
                cache.CC.k.logger.Error("Substation deletion deferred", "substationId", cache.ID(), "allocationId", id, "error", err)
                return used, false
            }
        }
        if more { return used, false }
    }
    cache.CC.ClearGridAttribute(cache.LoadAttributeId)
    cache.CC.ClearGridAttribute(cache.CapacityAttributeId)
    cache.CC.ClearGridAttribute(cache.ConnectionCountAttributeId)
    cache.CC.ClearGridAttribute(cache.ConnectionCapacityAttributeId)
    cache.CC.ClearPermissionsForObject(cache.ID())
    cache.Deleted = true
    cache.Changed = true
    return used, true
}

// Set the Owner Id data
func (cache *SubstationCache) SetOwnerId(owner string) {
    if (!cache.SubstationLoaded) { cache.LoadSubstation() }
    cache.Substation.Owner = owner
    cache.Changed = true
}


// Delete Permission
func (cache *SubstationCache) CanBeDeleteDBy(activePlayer *PlayerCache) (error) {
    return cache.CC.PermissionCheck(cache, activePlayer, types.PermDelete);
}

// Association Permission
func (cache *SubstationCache) CanManageConnectionsBy(activePlayer *PlayerCache) error {
    if err := cache.CheckAcceptingConnections(); err != nil { return err }
    return cache.CC.PermissionCheck(cache, activePlayer, types.PermSubstationConnection)
}

func (cache *SubstationCache) CanAllocateAsSourceBy(activePlayer *PlayerCache) error {
    if err := cache.CheckAcceptingConnections(); err != nil { return err }
    return cache.CC.PermissionCheck(cache, activePlayer, types.PermSourceAllocation)
}



func (cache *SubstationCache) ConnectionCountDecrement(amount uint64) {
    cache.CC.SetGridAttributeDecrement(cache.ConnectionCountAttributeId, amount)
    cache.CC.UpdateSubstationConnectionCapacity(cache.SubstationId)
}

func (cache *SubstationCache) ConnectionCountIncrement(amount uint64) {
    cache.CC.SetGridAttributeIncrement(cache.ConnectionCountAttributeId, amount)
    cache.CC.UpdateSubstationConnectionCapacity(cache.SubstationId)
}

func (cache *SubstationCache) PlayerIncrease() {
    cache.ConnectionCountIncrement(1)
}

func (cache *SubstationCache) PlayerDecrease(){
    cache.ConnectionCountDecrement(1)
}

func (cache *SubstationCache) GetName() string {
    if !cache.SubstationLoaded { cache.LoadSubstation() }
    return cache.Substation.Name
}

func (cache *SubstationCache) GetPfp() string {
    if !cache.SubstationLoaded { cache.LoadSubstation() }
    return cache.Substation.Pfp
}

func (cache *SubstationCache) SetName(name string) {
    if !cache.SubstationLoaded { cache.LoadSubstation() }
    cache.Substation.Name = name
    cache.Changed = true
}

func (cache *SubstationCache) SetPfp(pfp string) {
    if !cache.SubstationLoaded { cache.LoadSubstation() }
    cache.Substation.Pfp = pfp
    cache.Changed = true
}

func (cache *SubstationCache) CanUpdateUGCBy(activePlayer *PlayerCache) error {
    return cache.CC.UGCPermissionCheck(cache, activePlayer)
}
