package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

)


func GetAllocationID(sourceObjectId string, index uint64) (id string) {
    id = fmt.Sprintf("%s-%d", sourceObjectId, index)
	return
}


// AppendAllocation appends a allocation in the store with a new id
//
// This process also impacts the energy grid
func (k Keeper) AppendAllocation(
	ctx sdk.Context,
	allocation types.Allocation,
) (allocationId string, err error) {
    // Set the ID of the appended value

    allocation.index = k.GetGridAttribute(ctx, GetGridAttributeID(types.GridAttributeType_allocationPointerEnd, allocation.SourceObjectId))
    allocation.id = GetAllocationID(allocation.SourceObjectId, allocation.index)

    allocationSourceCapacity := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.SourceObjectId))
	if (allocation.Type == types.AllocationType_automated) {

	    // Automated Allocations must be the only allocation on a source
	    sourceAllocations := k.GetAllocationsFromSource(ctx, allocation.SourceObjectId)
	    if (len(sourceAllocations) > 0) {
	        return allocation.Id, sdkerrors.Wrapf(types.ErrAllocationAppend, "Allocation Source (%s) cannot have an automated Allocation with other allocations in place", allocation.SourceObjectId)
	    }

	    // Update the Power definition to be the capacity of the source
	    allocation.Power =  allocationSourceCapacity

	    // Add the AutoResize flag
	    k.SetAutoResizeAllocationSource(ctx, allocation.id, allocation.SourceObjectId)

    } else {
        sourceLoad = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, allocation.SourceObjectId))
        availableCapacity := allocationSourceCapacity - sourceLoad
        if (availableCapacity < allocation.Power) {
            return allocation.Id, sdkerrors.Wrapf(types.ErrAllocationAppend, "Allocation Source (%s) does not have the capacity (%d) for the power (%d) defined in this allocation",  allocation.SourceObjectId, availableCapacity, allocation.Power)
        }
    }

    // By this point, the function should be sure of it's success as stores will be written to

    // Update the Source Load
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, allocation.SourceObjectId), allocation.Power)

    // Set the Allocation Power
    k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id), newPower)

    // If a destination is already set for the allocation, update the capacity details there too
    //
    // Permission checks on this connection should be done in the calling function
    if (allocation.DestinationObjectId > 0) {
        k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.DestinationObjectId), allocation.Power)
    }

    // Increase the Index Pointer
    k.SetGridAttributeIncrement(ctx, GetGridAttributeID(types.GridAttributeType_allocationPointerEnd, allocation.SourceObjectId), 1)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	appendedValue := k.cdc.MustMarshal(&allocation)
	store.Set(allocation.Id, appendedValue)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})

	return allocation.Id, nil
}

// SetAllocation set a specific allocation in the store
// Update the grid accordingly for both sources and destinations
func (k Keeper) SetAllocation(ctx sdk.Context, allocation types.Allocation) {

	previousAllocation, previousAllocationFound := k.GetAllocation(ctx, allocation.Id)
	if (!previousAllocationFound) {
	    // This should be an append, not a set.
	    return
	}

	if (previousAllocation.SourceObjectId != allocation.SourceObjectId) {
	    // Should never change the Source of an Allocation
	    return
	}

	if (previousAllocation.Index != allocation.Index) {
	    // Should never change the SourceId of an Allocation
	    return
	}

    if (previousAllocation.Type != allocation.Type) {
        // Allocation Type should never change
        return
    }


    // Power not actually populated in the object yet
    previousAllocation.Power = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id))

    if (allocation.Type == types.AllocationType_automated) {
        allocation.Power = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.SourceObjectId))

    } else if (allocation.Type == types.AllocationType_static) {
        allocation.Power = previousAllocation.Power
    }

    if (previousAllocation.Power != allocation.Power) {
        k.SetGridAttributeDelta(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, allocation.Id), previousAllocation.Power, allocation.Power)
    }

    /* Possible Destination Changes
         * No Destination Change
            * Update Power Delta

         * No Destination   -> Destination
            * no update to old destination
            * increment on new destination

         * Destination B    -> Destination C
            * decrement on old destination with old power
            * increment on new destination with new power

         * Destination      -> No Destination
            * decrement of old power on previous destination
     */

    destinationCapacityId          := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.DestinationObjectId)
    previousDestinationCapacityId  := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, previousAllocation.DestinationObjectId)

    if (previousAllocation.DestinationObjectId == allocation.DestinationObjectId) {
        if ((allocation.DestinationObjectId != nil) && (previousAllocation.Power != allocation.Power)) {

            k.SetGridAttributeDelta(ctx, destinationCapacityId, previousAllocation.Power, allocation.Power)

            if (previousAllocation.Power > allocation.Power) {
                // Add Destination to the Grid Queue
                k.AppendGridCascadeQueue(ctx, allocation.DestinationObjectId)
            }
        }

    } else {

        // Deal with the previous Destination first
        if (previousAllocation.DestinationId > 0) {
            // Decrease the Capacity of the old Destination
            k.SetGridAttributeDecrement(ctx, previousDestinationCapacityId, previousAllocation.Power)
            // Add old Destination to the Grid Queue
            k.AppendGridCascadeQueue(ctx, previousAllocation.DestinationObjectId)

        }

        // Deal with the new Destination
        if (allocation.DestinationObjectId != nil){
            // We know that destination is greater than zero here because they're not equal to previousAllocation

            // Increment the Capacity of the new Destination
            k.SetGridAttributeIncrement(ctx, destinationCapacityId, allocation.Power)
        }
    }

    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
    b := k.cdc.MustMarshal(&allocation)
    store.Set(allocation.Id, b)

    _ = ctx.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})

}



// RemoveAllocation removes a allocation from the store
func (k Keeper) RemoveAllocation(ctx sdk.Context, allocationId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	store.Delete([]byte(allocationId))

	_ = ctx.EventManager().EmitTypedEvent(&types.EventAllocationDelete{AllocationId: allocationId})
}

// DestroyAllocation updates grid attributes before calling RemoveAllocation
func (k Keeper) DestroyAllocation(ctx sdk.Context, allocationId string) (destroyed bool){
    allocation, allocationFound := k.GetAllocation(ctx, allocationId)

    if allocationFound {
        allocation.Power = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id))

        // Decrease the Load of the Source
        k.SetGridAttributeDecrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, GetObjectID(allocation.SourceType, allocation.SourceId)), allocation.Power)

        // Decrease the Capacity of the Destination
        if (allocation.DestinationId > 0){
            destinationCapacityId := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.DestinationId)

            k.SetGridAttributeDecrement(ctx, destinationCapacityId, allocation.Power)
            // Add Destination to the Grid Queue
            k.AppendGridCascadeQueue(ctx, allocation.DestinationId)
        }

        // Clear the AutoResize hook on the source
        if (allocation.Type == types.AllocationType_automated ) {
            k.ClearAutoResizeAllocationBySource(ctx sdk.Context, allocationId)
        }

    	k.RemoveAllocation(ctx, allocation.Id)

    	destroyed = true
    } else {
        destroyed = false
    }

    return
}

// GetAllocation returns a allocation from its id
func (k Keeper) GetAllocation(ctx sdk.Context, allocationId string) (val types.Allocation, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	b := store.Get([]byte(allocationId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// GetAllAllocation returns all allocation
func (k Keeper) GetAllAllocation(ctx sdk.Context) (list []types.Allocation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Allocation
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}


// GetAllocationsFromSource returns all allocation relating to a source
func (k Keeper) GetAllocationsFromSource(ctx sdk.Context, sourceObjectId string) (list []types.Allocation) {

    allocationPointer    = k.GetGridAttribute(ctx, GetGridAttributeID(types.GridAttributeType_allocationPointerStart, sourceObjectId))
    allocationPointerEnd = k.GetGridAttribute(ctx, GetGridAttributeID(types.GridAttributeType_allocationPointerEnd, sourceObjectId))

    // Iterate through the allocationPointer until we successfully delete an allocation
    for (allocationPointer < allocationPointerEnd) {
        allocation, AllocationFound = k.GetAllocation(ctx, GetAllocationID(sourceObjectId, allocationPointer))
        allocationPointer           = allocationPointer + 1

        if allocationFound {
            list = append(list, allocation)
        }
    }

    return
}

/*
 * Helper functions for the Allocation Auto-Resizing Capabilities
 *
 * This allows for Allocations to be automatically updated when
 * the capacity of the source is updated elsewhere.
 *
 * Some rules:
 * - The Allocation must be defined as automatic during creation.
 * - The Allocation must be the other allocation on the source.
 * - If the source is a Substation, it must not allow player connections.
 *
 */

func (k Keeper) SetAutoResizeAllocationSource(ctx sdk.Context, allocationId string, sourceObjectId string) {
  	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationAutoResizeKey))

  	store.Set([]byte(sourceObjectId), []byte(allocationId))
}

func (k Keeper) GetAutoResizeAllocationBySource(ctx sdk.Context, sourceObjectId string) (allocationId string, allocationFound bool)  {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationAutoResizeKey))
    	allocationId = store.Get([]byte(sourceObjectId))
    	if allocationId == nil {
    		return "", false
    	}
    	return string(allocationId), true
}


func (k Keeper) ClearAutoResizeAllocationBySource(ctx sdk.Context, sourceObjectId string) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationAutoResizeKey))
    	store.Delete([]byte(sourceObjectId))
}


func (k Keeper) AutoResizeAllocation(ctx sdk.Context, allocationId string, sourceId string, oldPower uint64, newPower uint64) {
    allocation, allocationFound := k.GetAllocation(ctx, allocationId)

    // Update Allocation Power
    k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocationId), newPower)

    // Update Source Load
    k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, sourceId), newPower)

    // Update Destination Capacity
    k.SetGridAttributeDelta(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.DestinationId), oldPower, newPower)
    if (oldPower > newPower) {
        k.AppendGridCascadeQueue(ctx, allocation.DestinationId)
    }

}
