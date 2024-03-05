package keeper

import (
	//"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"fmt"

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

    allocation.Index = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_allocationPointerEnd, allocation.SourceObjectId))
    allocation.Id = GetAllocationID(allocation.SourceObjectId, allocation.Index)

    allocationSourceCapacity := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.SourceObjectId))
	if (allocation.Type == types.AllocationType_automated) {

	    // Automated Allocations must be the only allocation on a source
	    sourceAllocations := k.GetAllocationsFromSource(ctx, allocation.SourceObjectId, false)
	    if (len(sourceAllocations) > 0) {
	        return allocation.Id, sdkerrors.Wrapf(types.ErrAllocationAppend, "Allocation Source (%s) cannot have an automated Allocation with other allocations in place", allocation.SourceObjectId)
	    }

	    // Update the Power definition to be the capacity of the source
	    allocation.Power =  allocationSourceCapacity

	    // Add the AutoResize flag
	    k.SetAutoResizeAllocationSource(ctx, allocation.Id, allocation.SourceObjectId)

    } else {
        sourceLoad := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, allocation.SourceObjectId))
        availableCapacity := allocationSourceCapacity - sourceLoad
        if (availableCapacity < allocation.Power) {
            return allocation.Id, sdkerrors.Wrapf(types.ErrAllocationAppend, "Allocation Source (%s) does not have the capacity (%d) for the power (%d) defined in this allocation",  allocation.SourceObjectId, availableCapacity, allocation.Power)
        }
    }

    // By this point, the function should be sure of it's success as stores will be written to

    // Update the Source Load
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, allocation.SourceObjectId), allocation.Power)
    // Update Connection Capacity
    k.UpdateGridConnectionCapacity(ctx, allocation.SourceObjectId)

    // Set the Allocation Power
    k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id), allocation.Power)

    // If a destination is already set for the allocation, update the capacity details there too
    //
    // Permission checks on this connection should be done in the calling function
    if (allocation.DestinationId != "") {
        k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.DestinationId), allocation.Power)

        // Update Connection Capacity
        k.UpdateGridConnectionCapacity(ctx, allocation.DestinationId)
    }

    // Increase the Index Pointer
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_allocationPointerEnd, allocation.SourceObjectId), 1)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	appendedValue := k.cdc.MustMarshal(&allocation)
	store.Set([]byte(allocation.Id), appendedValue)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})

	return allocation.Id, nil
}

// SetAllocation set a specific allocation in the store
// Update the grid accordingly for both sources and destinations
func (k Keeper) SetAllocation(ctx sdk.Context, allocation types.Allocation) (types.Allocation, error){

	previousAllocation, previousAllocationFound := k.GetAllocation(ctx, allocation.Id, true)
	if (!previousAllocationFound) {
	    // This should be an append, not a set.
	    return allocation, sdkerrors.Wrapf(types.ErrAllocationSet, "Trying to set an allocation that doesn't exist yet. Should have been an append?")
	}

	if (previousAllocation.SourceObjectId != allocation.SourceObjectId) {
	    // Should never change the Source of an Allocation
	    return allocation, sdkerrors.Wrapf(types.ErrAllocationSet, "Source Object (%s vs %s) should never change during an allocation update", previousAllocation.SourceObjectId, allocation.SourceObjectId)
	}

	if (previousAllocation.Index != allocation.Index) {
	    // Should never change the SourceId of an Allocation
	    return allocation, sdkerrors.Wrapf(types.ErrAllocationSet, "Allocation Index (%d vs %d) should never change during an allocation update", previousAllocation.Index, allocation.Index)
	}

    if (previousAllocation.Type != allocation.Type) {
        // Allocation Type should never change
	    return allocation, sdkerrors.Wrapf(types.ErrAllocationSet, "Allocation Type (%d vs %d) should never change during an allocation update", previousAllocation.Type, allocation.Type)
    }


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

    destinationCapacityId          := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.DestinationId)
    previousDestinationCapacityId  := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, previousAllocation.DestinationId)


    if (previousAllocation.DestinationId == allocation.DestinationId) {
        if ((allocation.DestinationId != "") && (previousAllocation.Power != allocation.Power)) {

            k.SetGridAttributeDelta(ctx, destinationCapacityId, previousAllocation.Power, allocation.Power)

            // Update Connection Capacity
            k.UpdateGridConnectionCapacity(ctx, allocation.DestinationId)

            if (previousAllocation.Power > allocation.Power) {
                // Add Destination to the Grid Queue
                k.AppendGridCascadeQueue(ctx, allocation.DestinationId)
            }
        }

    } else {

        // Deal with the previous Destination first
        if (previousAllocation.DestinationId != "") {
            // Decrease the Capacity of the old Destination
            k.SetGridAttributeDecrement(ctx, previousDestinationCapacityId, previousAllocation.Power)

            // Deal with the player connection capacity
            k.UpdateGridConnectionCapacity(ctx, previousAllocation.DestinationId)

            // Add old Destination to the Grid Queue
            k.AppendGridCascadeQueue(ctx, previousAllocation.DestinationId)

        }

        // Deal with the new Destination
        if (allocation.DestinationId != ""){
            // We know that destination is greater than zero here because they're not equal to previousAllocation

            // Increment the Capacity of the new Destination
            k.SetGridAttributeIncrement(ctx, destinationCapacityId, allocation.Power)
        }
    }

    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
    b := k.cdc.MustMarshal(&allocation)
    store.Set([]byte(allocation.Id), b)

    _ = ctx.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})

    return allocation, nil

}



// RemoveAllocation removes a allocation from the store
func (k Keeper) RemoveAllocation(ctx sdk.Context, allocationId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	store.Delete([]byte(allocationId))

	_ = ctx.EventManager().EmitTypedEvent(&types.EventAllocationDelete{AllocationId: allocationId})
}

// DestroyAllocation updates grid attributes before calling RemoveAllocation
func (k Keeper) DestroyAllocation(ctx sdk.Context, allocationId string) (destroyed bool){
    allocation, allocationFound := k.GetAllocation(ctx, allocationId, true)

    if allocationFound {
        // Decrease the Load of the Source
        k.SetGridAttributeDecrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, allocation.SourceObjectId), allocation.Power)
        // Update Connection Capacity
        k.UpdateGridConnectionCapacity(ctx, allocation.SourceObjectId)

        // Decrease the Capacity of the Destination
        if (allocation.DestinationId != ""){
            destinationCapacityId := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.DestinationId)

            // Update Connection Capacity
            k.UpdateGridConnectionCapacity(ctx, allocation.DestinationId)

            k.SetGridAttributeDecrement(ctx, destinationCapacityId, allocation.Power)
            // Add Destination to the Grid Queue
            k.AppendGridCascadeQueue(ctx, allocation.DestinationId)
        }

        // Clear the AutoResize hook on the source
        if (allocation.Type == types.AllocationType_automated ) {
            k.ClearAutoResizeAllocationBySource(ctx, allocationId)
        }

    	k.RemoveAllocation(ctx, allocation.Id)

    	destroyed = true
    } else {
        destroyed = false
    }

    return
}

// TODO could likely be done far more efficiently
// Currently makes separate writes for each update
func (k Keeper) DestroyAllAllocations(ctx sdk.Context, allocations []types.Allocation) {
     for _, allocation := range allocations {
        k.DestroyAllocation(ctx, allocation.Id)
     }
}


// GetAllocation returns a allocation from its id
func (k Keeper) GetAllocation(ctx sdk.Context, allocationId string, full bool) (val types.Allocation, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	b := store.Get([]byte(allocationId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

    if full {
        val.Power = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, val.Id))
    }

	return val, true
}

// GetAllAllocation returns all allocation
func (k Keeper) GetAllAllocation(ctx sdk.Context, full bool) (list []types.Allocation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Allocation
		k.cdc.MustUnmarshal(iterator.Value(), &val)

        if full {
            val.Power = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, val.Id))
        }

		list = append(list, val)
	}

	return
}


// GetAllocationsFromSource returns all allocation relating to a source
func (k Keeper) GetAllocationsFromSource(ctx sdk.Context, sourceObjectId string, full bool) (list []types.Allocation) {

    allocationPointer    := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_allocationPointerStart, sourceObjectId))
    allocationPointerEnd := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_allocationPointerEnd, sourceObjectId))

    // Iterate through the allocationPointer until we successfully delete an allocation
    for (allocationPointer < allocationPointerEnd) {
        allocation, allocationFound := k.GetAllocation(ctx, GetAllocationID(sourceObjectId, allocationPointer), full)
        allocationPointer           = allocationPointer + 1

        if allocationFound {
            list = append(list, allocation)
        }
    }

    return
}


// GetAllAllocation returns all allocation
func (k Keeper) GetAllAllocationsFromDestination(ctx sdk.Context, destinationId string, full bool) (list []types.Allocation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Allocation
		k.cdc.MustUnmarshal(iterator.Value(), &val)

        if (val.DestinationId == destinationId) {
            if full {
                val.Power = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, val.Id))
            }

    		list = append(list, val)
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
    	allocationId = string(store.Get([]byte(sourceObjectId)))
    	if allocationId == "" {
    		return "", false
    	}
    	return string(allocationId), true
}


func (k Keeper) ClearAutoResizeAllocationBySource(ctx sdk.Context, sourceObjectId string) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationAutoResizeKey))
    	store.Delete([]byte(sourceObjectId))
}


func (k Keeper) AutoResizeAllocation(ctx sdk.Context, allocationId string, sourceId string, oldPower uint64, newPower uint64) {
    allocation, _ := k.GetAllocation(ctx, allocationId, false)

    // Update Allocation Power
    k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocationId), newPower)

    // Update Source Load
    k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, sourceId), newPower)

    // Update Destination Capacity
    k.SetGridAttributeDelta(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.DestinationId), oldPower, newPower)

    // Update Connection Capacity
    k.UpdateGridConnectionCapacity(ctx, allocation.DestinationId)

    // Check to see if we need to check on the Destination
    if (oldPower > newPower) {
        k.AppendGridCascadeQueue(ctx, allocation.DestinationId)
    }

}

