package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

)

// GetAllocationCount get the total number of allocation
func (k Keeper) GetAllocationCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.AllocationCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0 {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetAllocationCount set the total number of allocation
func (k Keeper) SetAllocationCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.AllocationCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// GetAllocationIDString returns the byte representation of the ID, based on Source details
func GetAllocationIDString(sourceType types.ObjectType, sourceId uint64, index uint64) (allocationId string) {
    allocationId := fmt.Sprintf("%d-%d-%d", sourceType, sourceId, index)
	return
}

// GetAllocationIDBytes returns the byte representation of the ID, based on Source details
func GetAllocationIDBytes(sourceType types.ObjectType, sourceId uint64, index uint64) []byte {
    id := fmt.Sprintf("%d-%d-%d", sourceType, sourceId, index)
	return []byte(id)
}

func GetAllocationIDBytesByObjectId(objectId []byte, index uint64) []byte {
    id := fmt.Sprintf("%s-%d", string(objectId), index)
	return []byte(id)
}


// AppendAllocation appends a allocation in the store with a new id
//
// This process also impacts the energy grid
func (k Keeper) AppendAllocation(
	ctx sdk.Context,
	allocation types.Allocation,
) (id string, count uint64, err error) {
	// Create the allocation
	count = k.GetAllocationCount(ctx)

    // Set the ID of the appended value
    allocation.index = k.GetGridAttribute(ctx, GetGridAttributeIDBytes(types.GridAttributeType_allocationPointerEnd, allocation.SourceType, allocation.SourceId))
    allocation.id = GetAllocationIDString(allocation.SourceType, allocation.SourceId, allocation.index)


    allocationSourceIdBytes := GetObjectIDBytes(allocation.SourceType, allocation.SourceId)
    allocationSourceCapacity := k.GetGridAttribute(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_capacity, allocationSourceIdBytes))
	if (allocation.Type == types.AllocationType_automated) {

	    // Automated Allocations must be the only allocation on a source
	    sourceAllocations := k.GetAllocationsFromSource(ctx, allocation.SourceType, allocation.SourceId)
	    if (len(sourceAllocations) > 0) {
	        return allocation.Id, count, sdkerrors.Wrapf(types.ErrAllocationAppend, "Allocation Source (%d) cannot have an automated Allocation with other allocations in place", allocation.SourceId)
	    }

	    // Update the Power definition to be the capacity of the source
	    allocation.Power =  allocationSourceCapacity

	    // Add the AutoResize flag
	    k.SetAutoResizeAllocationSource(ctx, []byte(allocation.id), allocationSourceIdBytes)

    } else {
        sourceLoad = k.GetGridAttribute(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_load, allocationSourceIdBytes))
        availableCapacity := allocationSourceCapacity - sourceLoad
        if (availableCapacity < allocation.Power) {
            return allocation.Id, count, sdkerrors.Wrapf(types.ErrAllocationAppend, "Allocation Source (%d) does not have the capacity (%d) for the power (%d) defined in this allocation", allocation.SourceId, availableCapacity, allocation.Power)
        }
    }

    // By this point, the function should be sure of it's success as stores will be written to

    // Update the Source Load
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_load, allocationSourceIdBytes), allocation.Power)

    // Set the Allocation Power
    k.SetGridAttribute(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_power, []byte(allocation.Id)), newPower)

    // If a destination is already set for the allocation, update the capacity details there too
    //
    // Permission checks on this connection should be done in the calling function
    if (allocation.DestinationId > 0) {
        allocationDestinationIdBytes := GetObjectIDBytes(types.ObjectType_substation, allocation.DestinationId)
        k.SetGridAttributeIncrement(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_capacity, allocationDestinationIdBytes), allocation.Power)
    }

    // Increase the Index Pointer
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDBytes(types.GridAttributeType_allocationPointerEnd, allocation.SourceType, allocation.SourceId), 1)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	appendedValue := k.cdc.MustMarshal(&allocation)
	store.Set([]byte(allocation.Id), appendedValue)

	// Update allocation count
	k.SetAllocationCount(ctx, count+1)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})

	return allocation.Id, count, nil
}

// SetAllocation set a specific allocation in the store
// Update the grid accordingly for both sources and destinations
func (k Keeper) SetAllocation(ctx sdk.Context, allocation types.Allocation) {

	previousAllocation, previousAllocationFound := k.GetAllocation(ctx, allocation.Id)
	if (!previousAllocationFound) {
	    // This should be an append, not a set.
	    return
	}

	if (previousAllocation.SourceType != allocation.SourceType) {
	    // Should never change the SourceType of an Allocation
	    return
	}

	if (previousAllocation.SourceId != allocation.SourceId) {
	    // Should never change the SourceId of an Allocation
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

    allocationSourceIdBytes := GetObjectIDBytes(allocation.SourceType, allocation.SourceId)

    // Power not actually populated in the object yet
    previousAllocation.Power = k.GetGridAttribute(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_power, []byte(allocation.Id)))

    if (allocation.Type == types.AllocationType_automated) {
        allocation.Power = k.GetGridAttribute(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_capacity, allocationSourceIdBytes))

    } else if (allocation.Type == types.AllocationType_static) {
        allocation.Power = previousAllocation.Power
    }

    if (previousAllocation.Power != allocation.Power) {
        k.SetGridAttributeDelta(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_load, []byte(allocation.Id)), previousAllocation.Power, allocation.Power)
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

    destinationIdBytes                  := GetObjectIDBytes(types.ObjectType_substation, allocation.DestinationId)
    destinationCapacityIdBytes          := GetGridAttributeIDBytesByObjectId(types.GridAttributeType_capacity, destinationIdBytes)

    previousDestinationIdBytes          := GetObjectIDBytes(types.ObjectType_substation, previousAllocation.DestinationId)
    previousDestinationCapacityIdBytes  := GetGridAttributeIDBytesByObjectId(types.GridAttributeType_capacity, previousDestinationIdBytes)

    if (previousAllocation.DestinationId == allocation.DestinationId) {
        if ((allocation.DestinationId > 0) && (previousAllocation.Power != allocation.Power)) {

            k.SetGridAttributeDelta(ctx, destinationCapacityIdBytes, previousAllocation.Power, allocation.Power)

            if (previousAllocation.Power > allocation.Power) {
                // Add Destination to the Grid Queue
                k.AppendGridCascadeQueue(ctx, destinationIdBytes)
            }
        }

    } else {

        // Deal with the previous Destination first
        if (previousAllocation.DestinationId > 0) {
            // Decrease the Capacity of the old Destination
            k.SetGridAttributeDecrement(ctx, previousDestinationCapacityIdBytes, previousAllocation.Power)
            // Add old Destination to the Grid Queue
            k.AppendGridCascadeQueue(ctx, previousDestinationIdBytes)

        }

        // Deal with the new Destination
        if (allocation.DestinationId > 0){
            // We know that destination is greater than zero here because they're not equal to previousAllocation

            // Increment the Capacity of the new Destination
            k.SetGridAttributeIncrement(ctx, destinationCapacityIdBytes, allocation.Power)
        }
    }

    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
    b := k.cdc.MustMarshal(&allocation)
    store.Set([]byte(allocation.Id), b)

    _ = ctx.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})

}



// RemoveAllocation removes a allocation from the store
func (k Keeper) RemoveAllocation(ctx sdk.Context, id string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	store.Delete([]byte(id))

	_ = ctx.EventManager().EmitTypedEvent(&types.EventAllocationDelete{AllocationId: id})
}

// DestroyAllocation updates grid attributes before calling RemoveAllocation
func (k Keeper) DestroyAllocation(ctx sdk.Context, allocationId []byte) (destroyed bool){
    allocation, allocationFound := k.GetAllocation(ctx, string(allocationId))

    if allocationFound {
        allocation.Power = k.GetGridAttribute(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_power, []byte(allocation.Id)))

        // Decrease the Load of the Source
        k.SetGridAttributeDecrement(ctx,GetGridAttributeIDBytesByGridQueueId(types.GridAttributeType_load, GetObjectIDBytes(allocation.SourceType, allocation.SourceId)), allocation.Power)

        // Decrease the Capacity of the Destination
        if (allocation.DestinationId > 0){
            destinationIdBytes := GetObjectIDBytes(types.ObjectType_substation, allocation.DestinationId)
            destinationCapacityIdBytes := GetGridAttributeIDBytesByGridQueueId(types.GridAttributeType_capacity, destinationIdBytes)

            k.SetGridAttributeDecrement(ctx, destinationIdBytes, allocation.Power)
            // Add Destination to the Grid Queue
            k.AppendGridCascadeQueue(ctx, destinationIdBytes)
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
func (k Keeper) GetAllocation(ctx sdk.Context, id string) (val types.Allocation, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	b := store.Get([]byte(id))
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
func (k Keeper) GetAllocationsFromSource(ctx sdk.Context, sourceType types.ObjectType, sourceId uint64) (list []types.Allocation) {

     allocationPointer      = k.GetGridAttributeAllocationPointerStart(ctx, queueId)
     allocationPointerEnd   = k.GetGridAttributeAllocationPointerEnd(ctx, queueId)

      // Iterate through the allocationPointer until we successfully delete an allocation
      for (allocationPointer < allocationPointerEnd) {
        allocation, AllocationFound = k.GetAllocation(ctx, GetAllocationIDBytes(sourceType, sourceId, allocationPointer))
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

func (k Keeper) SetAutoResizeAllocationSource(ctx sdk.Context, allocationIdBytes []byte, sourceObjectId []byte) {
  	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationAutoResizeKey))

  	store.Set(sourceObjectId, allocationIdBytes)
}

func (k Keeper) GetAutoResizeAllocationBySource(ctx sdk.Context, sourceObjectId []byte) (allocationIdBytes []byte, allocationFound bool)  {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationAutoResizeKey))
    	allocationIdBytes = store.Get(sourceObjectId)
    	if allocationIdBytes == nil {
    		return allocationIdBytes, false
    	}
    	return allocationIdBytes, true
}


func (k Keeper) ClearAutoResizeAllocationBySource(ctx sdk.Context, sourceObjectId []byte) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationAutoResizeKey))
    	store.Delete(sourceObjectId)
}


func (k Keeper) AutoResizeAllocation(ctx sdk.Context, allocationIdBytes []byte, sourceIdBytes []byte, oldPower uint64, newPower uint64) {
    allocation, allocationFound := k.GetAllocation(ctx, string(allocationIdBytes))

    // Update Allocation Power
    k.SetGridAttribute(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_power, allocationIdBytes), newPower)

    // Update Source Load
    k.SetGridAttribute(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_load, sourceIdBytes), newPower)

    // Update Destination Capacity
    destinationIdBytes := GetObjectIDBytes(types.ObjectType_substation, allocation.DestinationId)
    k.SetGridAttributeDelta(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_capacity, destinationIdBytes), oldPower, newPower)
    if (oldPower > newPower) {
        k.AppendGridCascadeQueue(ctx, destinationIdBytes)
    }

}
