package keeper

import (
    "strings"
	"structs/x/structs/types"
)

// GetAllocation returns an Allocation by ID, caching the result.
/* GetAllocation returns the cache for an allocation and whether it exists.
 *
 * A destroyed allocation does not exist, even though its cache is still sitting
 * in the map and its record is still in the store: Destroy defers both removals
 * to CommitAll. Reporting it as found hands the caller a live-looking handle to
 * something already torn down, and every mutator on it re-applies deltas that
 * were applied once. Historically Destroy retained the power attribute too,
 * letting a second teardown consume surviving allocations' load. Clearing that
 * attribute now reclaims storage; the Deleted guard remains the authority that
 * prevents stale handles from mutating an object that no longer exists.
 *
 * Every caller reads this boolean as existence and none checks Deleted, which is
 * the same shape as the phantom PlayerCache: a context getter is a cache
 * allocator and says nothing about whether the object is real. Teardown callers
 * want "already gone" to be a skip rather than a second attempt, and handlers
 * want it to be a rejection - which is what they get.
 */
func (cc *CurrentContext) GetAllocation(allocationId string) (*AllocationCache, bool) {
	if cache, exists := cc.allocations[allocationId]; exists {
		if cache.Deleted {
			return cache, false
		}
		return cache, true
	}

	value, found := cc.k.GetAllocation(cc.ctx, allocationId)
	if !found {
		return &AllocationCache{}, false
	}

	cc.allocations[allocationId] = &AllocationCache{
	    CC: cc,
	    AllocationId: allocationId,
		Allocation:  value,
		Loaded: true,

		PowerAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocationId),
		SourceCapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, value.SourceObjectId),
        SourceLoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_load, value.SourceObjectId),

	}
	return cc.allocations[allocationId], true
}


func (cc *CurrentContext) GenesisImportAllocation(allocation types.Allocation, importedPower uint64) {
	cc.allocations[allocation.Id] = &AllocationCache{
		CC:           cc,
		AllocationId: allocation.Id,
		Allocation:   allocation,
		Loaded:       true,
		Changed:      true,

		PowerAttributeId:          GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id),
		SourceCapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.SourceObjectId),
		SourceLoadAttributeId:     GetGridAttributeIDByObjectId(types.GridAttributeType_load, allocation.SourceObjectId),
	}
	cache := cc.allocations[allocation.Id]

	cc.k.SetAllocationSourceIndex(cc.ctx, allocation.SourceObjectId, allocation.Id)
	cc.k.SetAllocationDestinationIndex(cc.ctx, allocation.DestinationId, allocation.Id)

	if allocation.Type == types.AllocationType_automated {
		cc.k.SetAutoResizeAllocationSource(cc.ctx, allocation.Id, allocation.SourceObjectId)
		importedPower = cc.GetEffectiveGridCapacity(allocation.SourceObjectId)
	}

	if importedPower == 0 {
		return
	}

	cc.SetGridAttribute(cache.PowerAttributeId, importedPower)
	cc.SetGridAttributeIncrement(cache.SourceLoadAttributeId, importedPower)
	cc.UpdateSubstationConnectionCapacity(allocation.SourceObjectId)

	destCapAttrId := GetGridAttributeIDByObjectId(
		types.GridAttributeType_capacity, allocation.DestinationId)
	cc.SetGridAttributeIncrement(destCapAttrId, importedPower)

	cc.UpdateSubstationConnectionCapacity(allocation.DestinationId)
}

/* GetAllocationsBySourceUpTo loads at most limit allocations for a source, and
 * reports whether the source has more and the number of index rows consumed.
 * Charge rowsRead even when a stale index row produced no allocation cache.
 *
 * GetAllAllocationBySource below materializes every allocation on a source: one
 * index read, one store read and one retained cache object each, all before the
 * caller can decide it has seen enough. That is fine on a transaction path,
 * where gas bounds it, and not fine in the EndBlocker, where nothing does - a
 * source can be fragmented into as many one-power allocations as its capacity
 * allows, and GridCascade would load the whole set to shed a handful of them.
 *
 * The limit bounds the iteration rather than the result, so a source carrying
 * stale index rows costs the same as one that does not. `more` is what lets the
 * caller tell "this source is done" from "I stopped early", which are opposite
 * conclusions: the first means the load cannot be shed, the second means come
 * back next block.
 */
func (cc *CurrentContext) GetAllocationsBySourceUpTo(objectId string, limit int) (allocations []*AllocationCache, more bool, rowsRead int) {
	if limit <= 0 {
		return nil, cc.k.SourceHasAllocations(cc.ctx, objectId), 0
	}

	allocationList, more := cc.k.GetAllocationIdsBySourceIndexUpTo(cc.ctx, objectId, limit)

	for _, allocationId := range allocationList {
		allocation, allocationFound := cc.GetAllocation(allocationId)
		if allocationFound {
			allocations = append(allocations, allocation)
		}
	}

	return allocations, more, len(allocationList)
}

func (cc *CurrentContext) GetAllAllocationBySource(objectId string) (allocations []*AllocationCache) {
    allocationList := cc.k.GetAllAllocationIdBySourceIndex(cc.ctx, objectId)

    for _, allocationId := range allocationList {
        allocation, allocationFound := cc.GetAllocation(allocationId)
        if allocationFound {
            allocations = append(allocations, allocation)
        }
    }
    return
}

func (cc *CurrentContext) GetAllAllocationByDestination(objectId string) (allocations []*AllocationCache) {
    allocationList := cc.k.GetAllAllocationIdByDestinationIndex(cc.ctx, objectId)

    for _, allocationId := range allocationList {
        allocation, allocationFound := cc.GetAllocation(allocationId)
        if allocationFound {
            allocations = append(allocations, allocation)
        }
    }
    return
}

// GetAllocationsByDestinationUpTo resolves a bounded incoming-edge prefix
// through this operation's caches. A stale, moved or deleted allocation still
// consumes its raw index-row budget but cannot count as live incoming supply.
func (cc *CurrentContext) GetAllocationsByDestinationUpTo(objectId string, limit int) (allocations []*AllocationCache, more bool, rowsRead int) {
	ids, more := cc.k.GetAllocationIdsByDestinationIndexUpTo(cc.ctx, objectId, limit)
	for _, id := range ids {
		allocation, found := cc.GetAllocation(id)
		if found && allocation.GetAllocation().DestinationId == objectId {
			allocations = append(allocations, allocation)
		}
	}
	return allocations, more, len(ids)
}


func (cc *CurrentContext) NewAllocation(
	allocationType types.AllocationType,
	sourceObjectId string,
	destinationId string,
	creator string,
	controller string,
	power uint64,
) (*AllocationCache, error) {
    if strings.HasPrefix(sourceObjectId, "4-") {
        if err := cc.GetSubstation(sourceObjectId).CheckAcceptingConnections(); err != nil { return nil, err }
    }
    // Proto3 enums accept arbitrary int32 values. Reject undeclared types
    // before reserving an id, adding a cache, or changing load/index rows.
    // Keeping this in the constructor also protects internal cache callers.
    if _, valid := types.AllocationType_name[int32(allocationType)]; !valid {
        return &AllocationCache{}, types.NewAllocationError(sourceObjectId, "invalid_type")
    }
    // Validate the destination before allocating an id or reserving source
    // state. Ignoring SetDestination's refusal would create an allocation that
    // consumes power but never reaches the requested destination.
    if destinationId != "" && cc.GetSubstation(destinationId).CheckSubstation() != nil {
        return &AllocationCache{}, types.NewAllocationError(sourceObjectId, "unacceptable_destination").WithDestination(destinationId)
    }
    // Set the ID of the appended value

    allocation := types.Allocation{}

    allocation.Index = cc.k.GetAllocationCount(cc.ctx)
    // Agreement IDs share this sequence. Wrapping would reuse retired IDs,
    // including agreements whose completed accounting has been reclaimed.
    if allocation.Index == ^uint64(0) {
        return &AllocationCache{}, types.NewAllocationError(sourceObjectId, "allocation_id_exhausted")
    }

    allocation.Id               = GetObjectID(types.ObjectType_allocation, allocation.Index)
    allocation.Type             = allocationType
    allocation.SourceObjectId   = sourceObjectId
    allocation.Creator          = creator
    allocation.Controller       = controller

    allocationPowerAttributeId  := GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id)
    sourceCapacityAttributeId   := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId)
    sourceLoadAttributeId       := GetGridAttributeIDByObjectId(types.GridAttributeType_load, sourceObjectId)

    cache := &AllocationCache{
        CC: cc,
        AllocationId: allocation.Id,
        Allocation:  allocation,
        Loaded: true,
        Changed: true,

        PowerAttributeId: allocationPowerAttributeId,
        SourceCapacityAttributeId: sourceCapacityAttributeId,
        SourceLoadAttributeId: sourceLoadAttributeId,
    }

    // Validate all refusal conditions before inserting a changed cache or
    // reserving counters/indexes. A caller may continue after an error and
    // eventually commit this CurrentContext.
    if err := cache.validateSource(sourceObjectId); err != nil {
        return &AllocationCache{}, err
    }
    if !cache.IsAutomated() {
        if err := cache.validateInitialPower(power); err != nil {
            return &AllocationCache{}, err
        }
    }

    cc.k.SetAllocationCount(cc.ctx, allocation.Index + 1)
    cc.allocations[allocation.Id] = cache
    cache.setValidatedSource(sourceObjectId)

    if err := cc.allocations[allocation.Id].SetDestination(destinationId); err != nil {
        return &AllocationCache{}, err
    }

    var setPowerErr error

    if cc.allocations[allocation.Id].IsAutomated() {
       _, setPowerErr = cc.allocations[allocation.Id].SetAutomatedPower()
    } else {
       _, setPowerErr = cc.allocations[allocation.Id].SetInitialPower(power)
    }

    if setPowerErr != nil {
        return &AllocationCache{}, setPowerErr
    }

	return cc.allocations[allocation.Id], nil
}


// DestroyMultipleAllocations tears down a batch of allocations, settling any
// agreement behind each. A failure on one is logged and the rest still go: the
// callers are grid teardown paths where stopping early would leave allocations
// pointing at capacity that no longer exists.
/* DisconnectMultipleAgreementAllocations is DestroyMultipleAllocations for the
 * allocations arriving at a substation being deleted.
 *
 * Destroying a providerAgreement allocation settles its agreement through
 * PrematureCloseByAllocation, which pays the *provider* cancellation penalty to
 * the consumer. That is the right policy when the provider walks away, and the
 * wrong one when the consumer does - AgreementClose applies the consumer
 * penalty. A consumer could pick the favourable one by pointing their agreement
 * allocation at a substation they own and deleting the substation instead of
 * closing the agreement.
 *
 * So an inbound agreement allocation is disconnected rather than destroyed. The
 * agreement survives with nowhere to send its power, exactly as it does between
 * AgreementOpen and the first connect, and the only ways to end it early remain
 * the two that price it correctly.
 *
 * Disconnect rather than refuse the deletion, because SubstationAllocationConnect
 * does not ask for rights on the destination: anyone may point an allocation at
 * anyone's substation, so refusing would let a stranger's agreement pin a
 * substation in place forever.
 *
 * Outbound allocations are unaffected. An agreement is sourced from the
 * provider's substation, so deleting that one is the provider ending service,
 * and the provider penalty it pays is the correct price.
 */
func (cc *CurrentContext) DisconnectMultipleAgreementAllocations(allocationIds []string) {
    for _, allocationId := range allocationIds {
        allocation, found := cc.GetAllocation(allocationId)
        if !found {
            continue
        }

        if allocation.IsProviderAgreement() {
            if err := allocation.SetDestination(""); err != nil {
                cc.k.logger.Error("Agreement allocation could not be disconnected", "allocationId", allocationId, "error", err)
            }
            continue
        }

        if err := allocation.Destroy(); err != nil {
            cc.k.logger.Error("Allocation could not be destroyed", "allocationId", allocationId, "error", err)
        }
    }
}

func (cc *CurrentContext) DestroyMultipleAllocations(allocationIds []string) {
    for _, allocationId := range allocationIds {
        allocation, found := cc.GetAllocation(allocationId)
        if found {
            if err := allocation.Destroy(); err != nil {
                cc.k.logger.Error("Allocation could not be destroyed", "allocationId", allocationId, "error", err)
            }
        }
    }
}


// AutoResizeAllocation resizes the automated allocation an auto-resize hook
// names, and reports whether that allocation was there at all.
//
// The two failures are kept apart deliberately. A missing allocation means the
// hook is stale and there is nothing tracking this source's capacity, so the
// caller has to fall back to shedding load. A resize that fails is a different
// thing entirely: the allocation exists and is still tracking, and treating that
// as a stale hook would destroy allocations over a transient error. Only the
// first returns found == false.
func (cc *CurrentContext) AutoResizeAllocation(allocationId string, newPower uint64) (found bool) {
    allocation, found := cc.GetAllocation(allocationId)
    if !found {
        return false
    }

    if _, err := allocation.SetPower(newPower); err != nil {
        cc.k.logger.Error("Auto-resize could not set allocation power", "allocationId", allocationId, "newPower", newPower, "error", err)
    }

    return true
}

// allocationSourceGuild identifies the guild whose resource is being allocated.
// A guild source owns itself; a guild-owned substation names it as owner.
func (cc *CurrentContext) allocationSourceGuild(source PermissionedObject) *GuildCache {
    if source == nil { return nil }
    if guild, ok := source.(*GuildCache); ok { return guild }
    kind, _, err := types.ParseAssetOwnerID(source.GetOwnerId())
    if err != nil || kind != types.ObjectType_guild { return nil }
    guild := cc.GetGuild(source.GetOwnerId())
    if guild.CheckGuild() != nil { return nil }
    return guild
}
