package keeper

import (
	"structs/x/structs/types"
)

// GetAllocation returns an Allocation by ID, caching the result.
func (cc *CurrentContext) GetAllocation(allocationId string) (*AllocationCache, bool) {
	if cache, exists := cc.allocations[allocationId]; exists {
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
		sourceCapAttrId := GetGridAttributeIDByObjectId(
			types.GridAttributeType_capacity, allocation.SourceObjectId)
		importedPower = cc.GetGridAttribute(sourceCapAttrId)
	}

	if importedPower == 0 {
		return
	}

	cc.SetGridAttribute(cache.PowerAttributeId, importedPower)
	cc.SetGridAttributeIncrement(cache.SourceLoadAttributeId, importedPower)

	destCapAttrId := GetGridAttributeIDByObjectId(
		types.GridAttributeType_capacity, allocation.DestinationId)
	cc.SetGridAttributeIncrement(destCapAttrId, importedPower)

	cc.UpdateSubstationConnectionCapacity(allocation.DestinationId)
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


func (cc *CurrentContext) NewAllocation(
	allocationType types.AllocationType,
	sourceObjectId string,
	destinationId string,
	creator string,
	controller string,
	power uint64,
) (*AllocationCache, error) {
    // Set the ID of the appended value

    allocation := types.Allocation{}

    allocation.Index = cc.k.GetAllocationCount(cc.ctx)
    cc.k.SetAllocationCount(cc.ctx, allocation.Index + 1)

    allocation.Id               = GetObjectID(types.ObjectType_allocation, allocation.Index)
    allocation.Type             = allocationType
    allocation.Creator          = creator
    allocation.Controller       = controller

    allocationPowerAttributeId  := GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id)
    sourceCapacityAttributeId   := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceObjectId)
    sourceLoadAttributeId       := GetGridAttributeIDByObjectId(types.GridAttributeType_load, sourceObjectId)

    cc.allocations[allocation.Id] = &AllocationCache{
        CC: cc,
        AllocationId: allocation.Id,
        Allocation:  allocation,
        Loaded: true,
        Changed: true,

        PowerAttributeId: allocationPowerAttributeId,
        SourceCapacityAttributeId: sourceCapacityAttributeId,
        SourceLoadAttributeId: sourceLoadAttributeId,
    }

    sourceErr := cc.allocations[allocation.Id].SetSource(sourceObjectId)
    if sourceErr != nil {
        return &AllocationCache{}, sourceErr
    }

    cc.allocations[allocation.Id].SetDestination(destinationId)

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
