package keeper

import (
	"structs/x/structs/types"
)

type AllocationCache struct {
    CC              *CurrentContext
    AllocationId    string
	Allocation      types.Allocation
	Loaded          bool
	Changed         bool
	Deleted         bool

	PowerAttributeId            string
	SourceCapacityAttributeId   string
	SourceLoadAttributeId       string

}



func (cache *AllocationCache) Commit() {
    if cache.Changed {

    	cache.CC.k.logger.Info("Updating Allocation From Cache", "allocationId", cache.ID())

    	if cache.Deleted {
    	    cache.CC.k.RemoveAllocation(cache.CC.ctx, cache.ID())
    	} else {
    		cache.CC.k.SetAllocationOnly(cache.CC.ctx, cache.Allocation)
    	}

        cache.Changed = false

    }

}

func (cache *AllocationCache) IsChanged() bool {
	return cache.Changed
}

func (cache *AllocationCache) ID() string {
	return cache.AllocationId
}

func (cache *AllocationCache) LoadAllocation() bool {
	allocation, allocationFound := cache.CC.k.GetAllocation(cache.CC.ctx, cache.AllocationId)

	if allocationFound {
		cache.Allocation = allocation
		cache.Loaded = true
	}

	return allocationFound
}

func (cache *AllocationCache) GetAllocation() types.Allocation {
	if !cache.Loaded {
		cache.LoadAllocation()
	}
	return cache.Allocation
}

func (cache *AllocationCache) IsAutomated() bool {
    return cache.GetAllocation().Type == types.AllocationType_automated
}

func (cache *AllocationCache) IsDynamic() bool {
    return cache.GetAllocation().Type == types.AllocationType_dynamic
}

func (cache *AllocationCache) IsProviderAgreement() bool {
    return cache.GetAllocation().Type == types.AllocationType_providerAgreement
}

func (cache *AllocationCache) IsStatic() bool {
    return cache.GetAllocation().Type == types.AllocationType_static
}

func (cache *AllocationCache) GetPower() (uint64) {
    return cache.CC.GetGridAttribute(cache.PowerAttributeId)
}

func (cache *AllocationCache) GetObjectCapacity(objectId string) (uint64) {
    objectCapacityAttributeId := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId)
    return cache.CC.GetGridAttribute(objectCapacityAttributeId)
}

func (cache *AllocationCache) GetObjectLoad(objectId string) (uint64) {
    objectLoadAttributeId := GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId)
    return cache.CC.GetGridAttribute(objectLoadAttributeId)
}

func (cache *AllocationCache) SetAllocationSourceObjectId(sourceObjectId string) (bool) {
    if ! cache.Loaded {
        if ! cache.LoadAllocation() {
            return false
        }
    }

    cache.Allocation.SourceObjectId = sourceObjectId
    cache.Changed = true
    return true
}

func (cache *AllocationCache) SetAllocationDestinationId(destinationId string) (bool) {
    if ! cache.Loaded {
        if ! cache.LoadAllocation() {
            return false
        }
    }

    cache.Allocation.DestinationId = destinationId
    cache.Changed = true
    return true
}

func (cache *AllocationCache) SetAllocationController(address string) (bool) {
    if ! cache.Loaded {
        if ! cache.LoadAllocation() {
            return false
        }
    }

    cache.Allocation.Controller = address
    cache.Changed = true
    return true
}


func (cache *AllocationCache) SetSource(sourceObjectId string) (error) {

	if cache.IsAutomated() {
        // Automated Allocations must be the only allocation on a source
        // All allocations must have > 0 power, so this should show accurately
        _, destinationAutoResizeAllocationFound := cache.CC.k.GetAutoResizeAllocationBySource(cache.CC.ctx, sourceObjectId)
        if destinationAutoResizeAllocationFound || cache.GetObjectLoad(sourceObjectId) > 0 {
            return types.NewAllocationError(sourceObjectId, "automated_conflict")
        }

        cache.CC.k.SetAutoResizeAllocationSource(cache.CC.ctx, cache.ID(), sourceObjectId)
    }

    cache.SetAllocationSourceObjectId(sourceObjectId)
    return nil
}

func (cache *AllocationCache) SetAutomatedPower() (uint64, error) {
    if !cache.IsAutomated() {
        return 0, types.NewAllocationError(cache.ID(), "incorrect_type")
    }

    newPower := cache.GetObjectCapacity(cache.GetAllocation().SourceObjectId)
    return cache.SetPower(newPower)
}

func (cache *AllocationCache) SetDynamicPower(newPower uint64) (uint64, error) {
    if !cache.IsDynamic() {
        return 0, types.NewAllocationError(cache.ID(), "incorrect_type")
    }

    if (newPower == 0) {
        return 0, types.NewAllocationError(cache.GetAllocation().SourceObjectId, "new_power_zero")
    }

    sourceLoad := cache.CC.GetGridAttribute(cache.SourceLoadAttributeId)
    sourceCapacity := cache.CC.GetGridAttribute(cache.SourceCapacityAttributeId)
    availableCapacity := sourceCapacity - sourceLoad
    if (availableCapacity < newPower) {
        return 0, types.NewAllocationError(cache.GetAllocation().SourceObjectId, "capacity_exceeded").WithCapacity(availableCapacity, newPower)
    }

    return cache.SetPower(newPower)
}

func (cache *AllocationCache) SetInitialPower(newPower uint64) (uint64, error) {
    if cache.IsAutomated() {
        return 0, types.NewAllocationError(cache.ID(), "incorrect_type")
    }

    if (newPower == 0) {
        return 0, types.NewAllocationError(cache.GetAllocation().SourceObjectId, "new_power_zero")
    }

    sourceLoad := cache.CC.GetGridAttribute(cache.SourceLoadAttributeId)
    sourceCapacity := cache.CC.GetGridAttribute(cache.SourceCapacityAttributeId)
    availableCapacity := sourceCapacity - sourceLoad
    if (availableCapacity < newPower) {
        return 0, types.NewAllocationError(cache.GetAllocation().SourceObjectId, "capacity_exceeded").WithCapacity(availableCapacity, newPower)
    }

    return cache.SetPower(newPower)
}


func (cache *AllocationCache) SetPower(newPower uint64) (uint64, error) {
    previousPower := cache.GetPower()
    destinationId := cache.GetAllocation().DestinationId

    if previousPower != newPower {
        if destinationId != "" {
            destinationCapacityAttributeId := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, destinationId)
            cache.CC.SetGridAttributeDelta(destinationCapacityAttributeId, previousPower, newPower)

            // Update Connection Capacity
            cache.CC.UpdateSubstationConnectionCapacity(destinationId)

            if (previousPower > newPower) {
                // Add Destination to the Grid Queue
                cache.CC.k.AppendGridCascadeQueue(cache.CC.ctx, destinationId)
            }
        }
        cache.CC.SetGridAttributeDelta(cache.PowerAttributeId, previousPower, newPower)
        cache.CC.SetGridAttributeDelta(cache.SourceLoadAttributeId, previousPower, newPower)
    }

    return newPower, nil
}


func (cache *AllocationCache) SetDestination(objectId string) (error) {

    previousDestinationId := cache.GetAllocation().DestinationId

    if (previousDestinationId != objectId) {

        cache.CC.k.RemoveAllocationDestinationIndex(cache.CC.ctx, previousDestinationId, cache.ID())

        // Deal with the previous Destination first
        if (previousDestinationId != "") {
            // Decrease the Capacity of the old Destination
            previousDestinationCapacityAttributeId := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, previousDestinationId)
            cache.CC.SetGridAttributeDecrement(previousDestinationCapacityAttributeId, cache.GetPower())

            // Deal with the player connection capacity
            cache.CC.UpdateSubstationConnectionCapacity(previousDestinationId)

            // Add old Destination to the Grid Queue
            cache.CC.k.AppendGridCascadeQueue(cache.CC.ctx, previousDestinationId)

        }

        // Deal with the new Destination
        if (objectId != "") {

            if !cache.CC.GetSubstation(objectId).LoadSubstation() {
                return types.NewAllocationError(cache.ID(), "unacceptable_destination")
            }


            if cache.GetPower() > 0 {
                // Increment the Capacity of the new Destination
                destinationCapacityAttributeId := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId)
                cache.CC.SetGridAttributeIncrement(destinationCapacityAttributeId, cache.GetPower())

                // Deal with the player connection capacity
                cache.CC.UpdateSubstationConnectionCapacity(objectId)
            }

            // Update the Destination Allocation Index
            cache.CC.k.SetAllocationDestinationIndex(cache.CC.ctx, objectId, cache.ID())
        }

        cache.SetAllocationDestinationId(objectId)
    }

    return nil
}

func (cache *AllocationCache) Destroy() (error) {

    if !cache.LoadAllocation() {
        return types.NewAllocationError(cache.ID(), "unknown_allocation")
    }

    power := cache.GetPower()

    // Decrease the Load of the Source
    cache.CC.SetGridAttributeDecrement(cache.SourceLoadAttributeId, power)

    // Update Connection Capacity
    cache.CC.UpdateSubstationConnectionCapacity(cache.GetAllocation().SourceObjectId)

    // Decrease the Capacity of the Destination
    if (cache.GetAllocation().DestinationId != ""){
        destinationCapacityId := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, cache.GetAllocation().DestinationId)
        cache.CC.SetGridAttributeDecrement(destinationCapacityId, power)

        // Update Connection Capacity
        cache.CC.UpdateSubstationConnectionCapacity(cache.GetAllocation().DestinationId)


        // Add Destination to the Grid Queue
        cache.CC.k.AppendGridCascadeQueue(cache.CC.ctx, cache.GetAllocation().DestinationId)

        // Remove the destination index
        cache.CC.k.RemoveAllocationDestinationIndex(cache.CC.ctx, cache.GetAllocation().DestinationId, cache.ID())
    }

    // Clear the AutoResize hook on the source
    if cache.IsAutomated() {
        cache.CC.k.ClearAutoResizeAllocationBySource(cache.CC.ctx, cache.ID())
    }

    cache.CC.k.RemoveAllocationSourceIndex(cache.CC.ctx, cache.GetAllocation().SourceObjectId, cache.ID())


    // Check for a related Agreement and close it
    // TODO change to CC
    agreement := cache.CC.GetAgreement(GetObjectID(types.ObjectType_agreement, cache.GetAllocation().Index))
    if agreement.LoadAgreement() {
        agreement.PrematureCloseByAllocation()
    }

    cache.Changed = true
    cache.Deleted = true
    return nil
}


func (cache *AllocationCache) CanBeUpdatedBySource() bool {
    // TODO
    return true
}

func (cache *AllocationCache) CanBeUpdatedByController() bool {
    // TODO
    return true
}

