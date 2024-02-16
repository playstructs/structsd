package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

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


// AppendAllocation appends a allocation in the store with a new id and update the count
func (k Keeper) AppendAllocation(
	ctx sdk.Context,
	allocation types.Allocation,
) uint64 {
	// Create the allocation
	count := k.GetAllocationCount(ctx)


    // Set the ID of the appended value
    allocation.index = k.GetGridAttribute(ctx, GetGridAttributeIDBytes(types.GridAttributeType_allocationPointerEnd, allocation.SourceType, allocation.SourceId))
    allocation.id = GetAllocationIDString(allocation.SourceType, allocation.SourceId, allocation.index)

    // Increase the Index Pointer
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDBytes(types.GridAttributeType_allocationPointerEnd, allocation.SourceType, allocation.SourceId), 1)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	appendedValue := k.cdc.MustMarshal(&allocation)
	store.Set([]byte(allocation.Id), appendedValue)

	// Update allocation count
	k.SetAllocationCount(ctx, count+1)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})

	return count
}

// SetAllocation set a specific allocation in the store
// Update the grid accordingly for both sources and destinations
func (k Keeper) SetAllocation(ctx sdk.Context, allocation types.Allocation) {

	previousAllocation, previousAllocationFound := k.GetAllocation(ctx, allocation.Id)
	if (!previousAllocationFound) {
	    // This should be an append, not a set.
	    return
	}

	if (previousAllocation.SourceId != allocation.SourceId) {
	    // Should never change the SourceId of an Allocation
	    return
	}


    if (previousAllocation.Power != allocation.Power) {
        // Alter the Capacity of the Allocation Source
        k.SetGridAttributeDelta(ctx, GetGridAttributeIDBytesByGridQueueId(types.GridAttributeType_load, GetObjectIDBytes(allocation.SourceType, allocation.SourceId)), previousAllocation.Power, allocation.Power)
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

    destinationCapacityIDBytes          := GetGridAttributeIDBytesByObjectId(types.GridAttributeType_capacity, GetObjectIDBytes(types.ObjectType_substation, allocation.DestinationId))
    previousDestinationCapacityIdBytes  := GetGridAttributeIDBytesByObjectId(types.GridAttributeType_capacity, GetObjectIDBytes(types.ObjectType_substation, previousAllocation.DestinationId))

    if (previousAllocation.DestinationId == allocation.DestinationId) {
        if ((allocation.DestinationId > 0) && (previousAllocation.Power != allocation.Power)) {

            k.SetGridAttributeDelta(ctx, destinationCapacityIDBytes, previousAllocation.Power, allocation.Power)

            if (previousAllocation.Power > allocation.Power) {
                // Add Destination to the Grid Queue
                k.AppendGridCascadeQueue(ctx, destinationCapacityIDBytes)
            }
        }

    } else {

        // Deal with the previous Destination first
        if (previousAllocation.DestinationId > 0) {
            // Decrease the Capacity of the old Destination
            k.SetGridAttributeDecrement(ctx, previousDestinationCapacityIdBytes, previousAllocation.Power)
            // Add old Destination to the Grid Queue
            k.AppendGridCascadeQueue(ctx, previousDestinationCapacityIdBytes)

        }

        // Deal with the new Destination
        if (allocation.DestinationId > 0){
            // We know that destination is greater than zero here because they're not equal to previousAllocation

            // Increment the Capacity of the new Destination
            k.SetGridAttributeIncrement(ctx, destinationCapacityIDBytes, allocation.Power)
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
    allocation, allocationFound := k.GetAllocation(ctx, allocationId)

    if allocationFound {
        // Decrease the Load of the Source
        k.SetGridAttributeDecrement(ctx,GetGridAttributeIDBytesByGridQueueId(types.GridAttributeType_load, GetObjectIDBytes(allocation.SourceType, allocation.SourceId)), allocation.Power)

        // Decrease the Capacity of the Destination
        destinationIdBytes := GetGridAttributeIDBytesByGridQueueId(types.GridAttributeType_capacity, GetObjectIDBytes(ObjectType_substation, allocation.DestinationId))
        k.SetGridAttributeDecrement(ctx, destinationIdBytes, allocation.Power)
        // Add Destination to the Grid Queue
        k.AppendGridCascadeQueue(ctx, destinationIdBytes)

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


