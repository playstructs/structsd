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


func (cc *CurrentContext) DestroyMultipleAllocations(allocationIds []string) {
    for _, allocationId := range allocationIds {
        allocation, found := cc.GetAllocation(allocationId)
        if found {
            allocation.Destroy()
        }
    }
}


func (cc *CurrentContext) AutoResizeAllocation(allocationId string, newPower uint64) {
    allocation, found := cc.GetAllocation(allocationId)
    if found {
        allocation.SetPower(newPower)
    }
}
