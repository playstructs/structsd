package keeper

import (
	//"encoding/binary"
    "context"

    "github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	sdkerrors "cosmossdk.io/errors"

	"fmt"

)




// AppendAllocation appends a allocation in the store with a new id
//
// This process also impacts the energy grid
func (k Keeper) AppendAllocation(
	ctx context.Context,
	allocation types.Allocation,
	power uint64,
) (allocationId string, finalPower uint64, err error) {
    // Set the ID of the appended value

    allocation.Index = k.GetAllocationCount(ctx)
    k.SetAllocationCount(ctx, allocation.Index + 1)

    allocation.Id = GetObjectID(types.ObjectType_allocation, allocation.Index)

    allocationSourceCapacity := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.SourceObjectId))
	if (allocation.Type == types.AllocationType_automated) {

	    // Automated Allocations must be the only allocation on a source
	    // TODO - make a quicker lookup. This is going to get slow as allocation increase
	    sourceAllocations := k.GetAllocationsFromSource(ctx, allocation.SourceObjectId)
	    if (len(sourceAllocations) > 0) {
	        return allocation.Id, power, sdkerrors.Wrapf(types.ErrAllocationAppend, "Allocation Source (%s) cannot have an automated Allocation with other allocations in place", allocation.SourceObjectId)
	    }

	    // Update the Power definition to be the capacity of the source
	    power =  allocationSourceCapacity

	    // Add the AutoResize flag
	    k.SetAutoResizeAllocationSource(ctx, allocation.Id, allocation.SourceObjectId)

    } else {
        sourceLoad := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, allocation.SourceObjectId))
        availableCapacity := allocationSourceCapacity - sourceLoad
        if (availableCapacity < power) {
            return allocation.Id, power, sdkerrors.Wrapf(types.ErrAllocationAppend, "Allocation Source (%s) does not have the capacity (%d) for the power (%d) defined in this allocation",  allocation.SourceObjectId, availableCapacity, power)
        }
    }

    // By this point, the function should be sure of it's success as stores will be written to

    // Update the Source Load
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, allocation.SourceObjectId), power)
    // Update Connection Capacity
    k.UpdateGridConnectionCapacity(ctx, allocation.SourceObjectId)

    // Set the Allocation Power
    k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id), power)

    // If a destination is already set for the allocation, update the capacity details there too
    //
    // Permission checks on this connection should be done in the calling function
    if (allocation.DestinationId != "") {
        k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.DestinationId), power)

        // Update Connection Capacity
        k.UpdateGridConnectionCapacity(ctx, allocation.DestinationId)

        // Update the Destination Index
        k.SetAllocationDestinationIndex(ctx, allocation.DestinationId, allocation.Id)
    }

    k.SetAllocationSourceIndex(ctx, allocation.SourceObjectId, allocation.Id)

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationKey))
	appendedValue := k.cdc.MustMarshal(&allocation)
	store.Set([]byte(allocation.Id), appendedValue)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})

	return allocation.Id, power, nil
}

func (k Keeper) SetAllocationOnly(ctx context.Context, allocation types.Allocation) (types.Allocation, error){

    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationKey))
    b := k.cdc.MustMarshal(&allocation)
    store.Set([]byte(allocation.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})

    return allocation,  nil

}

// SetAllocation set a specific allocation in the store
// Update the grid accordingly for both sources and destinations
func (k Keeper) SetAllocation(ctx context.Context, allocation types.Allocation, newPower uint64) (types.Allocation, uint64, error){

	previousAllocation, previousAllocationFound := k.GetAllocation(ctx, allocation.Id)
	if (!previousAllocationFound) {
	    // This should be an append, not a set.
	    return allocation, newPower, sdkerrors.Wrapf(types.ErrAllocationSet, "Trying to set an allocation that doesn't exist yet. Should have been an append?")
	}

    previousPower := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id))

	if (previousAllocation.SourceObjectId != allocation.SourceObjectId) {
	    // Should never change the Source of an Allocation
	    return allocation, newPower, sdkerrors.Wrapf(types.ErrAllocationSet, "Source Object (%s vs %s) should never change during an allocation update", previousAllocation.SourceObjectId, allocation.SourceObjectId)
	}

	if (previousAllocation.Index != allocation.Index) {
	    // Should never change the SourceId of an Allocation
	    return allocation, newPower, sdkerrors.Wrapf(types.ErrAllocationSet, "Allocation Index (%d vs %d) should never change during an allocation update", previousAllocation.Index, allocation.Index)
	}

    if (previousAllocation.Type != allocation.Type) {
        // Allocation Type should never change
	    return allocation, newPower, sdkerrors.Wrapf(types.ErrAllocationSet, "Allocation Type (%d vs %d) should never change during an allocation update", previousAllocation.Type, allocation.Type)
    }


    if (allocation.Type == types.AllocationType_automated) {
        newPower = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.SourceObjectId))

    } else if (allocation.Type == types.AllocationType_static) {
        newPower = previousPower
    }

    if (previousPower != newPower) {
        k.SetGridAttributeDelta(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id), previousPower, newPower)
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
        if ((allocation.DestinationId != "") && (previousPower != newPower)) {

            k.SetGridAttributeDelta(ctx, destinationCapacityId, previousPower, newPower)

            // Update Connection Capacity
            k.UpdateGridConnectionCapacity(ctx, allocation.DestinationId)

            if (previousPower > newPower) {
                // Add Destination to the Grid Queue
                k.AppendGridCascadeQueue(ctx, allocation.DestinationId)
            }
        }

    } else {
        k.RemoveAllocationDestinationIndex(ctx, previousAllocation.DestinationId, allocation.Id)

        // Deal with the previous Destination first
        if (previousAllocation.DestinationId != "") {
            // Decrease the Capacity of the old Destination
            k.SetGridAttributeDecrement(ctx, previousDestinationCapacityId, previousPower)

            // Deal with the player connection capacity
            k.UpdateGridConnectionCapacity(ctx, previousAllocation.DestinationId)

            // Add old Destination to the Grid Queue
            k.AppendGridCascadeQueue(ctx, previousAllocation.DestinationId)

        }

        // Deal with the new Destination
        if (allocation.DestinationId != ""){
            // We know that destination is greater than zero here because they're not equal to previousAllocation

            // Increment the Capacity of the new Destination
            k.SetGridAttributeIncrement(ctx, destinationCapacityId, newPower)

            // Deal with the player connection capacity
            k.UpdateGridConnectionCapacity(ctx, allocation.DestinationId)

            // Update the Destination Allocation Index
            k.SetAllocationDestinationIndex(ctx, allocation.DestinationId, allocation.Id)

        }
    }

    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationKey))
    b := k.cdc.MustMarshal(&allocation)
    store.Set([]byte(allocation.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})

    return allocation, newPower,  nil

}



// ImportAllocation set a specific allocation in the store
// Assumes Grid updates happen elsewhere
func (k Keeper) ImportAllocation(ctx context.Context, allocation types.Allocation){
    k.SetAllocationSourceIndex(ctx, allocation.SourceObjectId, allocation.Id)
    k.SetAllocationDestinationIndex(ctx, allocation.DestinationId, allocation.Id)

    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationKey))
    b := k.cdc.MustMarshal(&allocation)
    store.Set([]byte(allocation.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})
}



// RemoveAllocation removes a allocation from the store
func (k Keeper) RemoveAllocation(ctx context.Context, allocationId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationKey))
	store.Delete([]byte(allocationId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ObjectId: allocationId})
}

// DestroyAllocation updates grid attributes before calling RemoveAllocation
func (k Keeper) DestroyAllocation(ctx context.Context, allocationId string) (destroyed bool){
    allocation, allocationFound := k.GetAllocation(ctx, allocationId)

    power := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocationId))


    if allocationFound {
        // Decrease the Load of the Source
        k.SetGridAttributeDecrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, allocation.SourceObjectId), power)
        // Update Connection Capacity
        k.UpdateGridConnectionCapacity(ctx, allocation.SourceObjectId)

        // Decrease the Capacity of the Destination
        if (allocation.DestinationId != ""){
            destinationCapacityId := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, allocation.DestinationId)

            // Update Connection Capacity
            k.UpdateGridConnectionCapacity(ctx, allocation.DestinationId)

            k.SetGridAttributeDecrement(ctx, destinationCapacityId, power)
            // Add Destination to the Grid Queue
            k.AppendGridCascadeQueue(ctx, allocation.DestinationId)
        }

        // Clear the AutoResize hook on the source
        if (allocation.Type == types.AllocationType_automated ) {
            k.ClearAutoResizeAllocationBySource(ctx, allocationId)
        }

    	k.RemoveAllocation(ctx, allocation.Id)

        k.RemoveAllocationSourceIndex(ctx, allocation.SourceObjectId, allocation.Id)
        k.RemoveAllocationDestinationIndex(ctx, allocation.DestinationId, allocation.Id)

    	destroyed = true
    } else {
        destroyed = false
    }

    return
}

// TODO could likely be done far more efficiently
// Currently makes separate writes for each update
func (k Keeper) DestroyAllAllocations(ctx context.Context, allocations []types.Allocation) {
     for _, allocation := range allocations {
        k.DestroyAllocation(ctx, allocation.Id)
     }
}


// GetAllocation returns a allocation from its id
func (k Keeper) GetAllocation(ctx context.Context, allocationId string) (val types.Allocation, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationKey))
	b := store.Get([]byte(allocationId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// GetAllAllocation returns all allocation
func (k Keeper) GetAllAllocation(ctx context.Context) (list []types.Allocation) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Allocation
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		list = append(list, val)
	}

	return
}


