package keeper

import (
	"encoding/binary"
    "context"

    "github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"structs/x/structs/types"


)

func AllocationSourceKeyPrefix(sourceId string) []byte {
	return []byte(types.AllocationSourceKey + sourceId + "/")
}

func AllocationDestinationKeyPrefix(destinationId string) []byte {
	return []byte(types.AllocationDestinationKey + destinationId + "/")
}


func (k Keeper) SetAllocationSourceIndex(ctx context.Context, sourceId string, allocationId string) (err error) {
    sourceIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AllocationSourceKeyPrefix(sourceId))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, 1)

	sourceIndexStore.Set([]byte(allocationId), bz)

	return err
}

func (k Keeper) RemoveAllocationSourceIndex(ctx context.Context, sourceId string, allocationId string) (err error) {
    sourceIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AllocationSourceKeyPrefix(sourceId))
	sourceIndexStore.Delete([]byte(allocationId))

	return err
}


func (k Keeper) GetAllAllocationIdBySourceIndex(ctx context.Context, sourceId string) (list []string) {
	sourceIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AllocationSourceKeyPrefix(sourceId))
	iterator := storetypes.KVStorePrefixIterator(sourceIndexStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		list = append(list, string(iterator.Key()))
	}

    return
}

func (k Keeper) GetAllAllocationBySourceIndex(ctx context.Context, sourceId string) (list []types.Allocation) {
	sourceIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AllocationSourceKeyPrefix(sourceId))
	iterator := storetypes.KVStorePrefixIterator(sourceIndexStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		val, found := k.GetAllocation(ctx, string(iterator.Key()))
		if found {
		    list = append(list, val)
    	}
    }
    return
}



func (k Keeper) SetAllocationDestinationIndex(ctx context.Context, destinationId string, allocationId string) (err error) {
    destinationIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AllocationDestinationKeyPrefix(destinationId))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, 1)

	destinationIndexStore.Set([]byte(allocationId), bz)

	return err
}

func (k Keeper) RemoveAllocationDestinationIndex(ctx context.Context, destinationId string, allocationId string) (err error) {
    destinationIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AllocationDestinationKeyPrefix(destinationId))
	destinationIndexStore.Delete([]byte(allocationId))

	return err
}

func (k Keeper) GetAllAllocationIdByDestinationIndex(ctx context.Context, destinationId string) (list []string) {
	destinationIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AllocationDestinationKeyPrefix(destinationId))
	iterator := storetypes.KVStorePrefixIterator(destinationIndexStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		list = append(list, string(iterator.Key()))
	}

    return
}

func (k Keeper) GetAllAllocationByDestinationIndex(ctx context.Context, destinationId string) (list []types.Allocation) {
	destinationIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AllocationDestinationKeyPrefix(destinationId))
	iterator := storetypes.KVStorePrefixIterator(destinationIndexStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		val, found := k.GetAllocation(ctx, string(iterator.Key()))
		if found {
		    list = append(list, val)
    	}
    }
    return
}



// GetAllocationCount get the total number of allocations
func (k Keeper) GetAllocationCount(ctx context.Context) uint64 {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.AllocationCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0  {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetAllocationCount set the total number of allocations
func (k Keeper) SetAllocationCount(ctx context.Context, count uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.AllocationCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
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

func (k Keeper) SetAutoResizeAllocationSource(ctx context.Context, allocationId string, sourceObjectId string) {
  	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationAutoResizeKey))

  	store.Set([]byte(sourceObjectId), []byte(allocationId))
}

func (k Keeper) GetAutoResizeAllocationBySource(ctx context.Context, sourceObjectId string) (allocationId string, allocationFound bool)  {
    	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationAutoResizeKey))
    	allocationId = string(store.Get([]byte(sourceObjectId)))
    	if allocationId == "" {
    		return "", false
    	}
    	return string(allocationId), true
}


func (k Keeper) ClearAutoResizeAllocationBySource(ctx context.Context, sourceObjectId string) {
    	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationAutoResizeKey))
    	store.Delete([]byte(sourceObjectId))
}


func (k Keeper) AutoResizeAllocation(ctx context.Context, allocationId string, sourceId string, oldPower uint64, newPower uint64) {
    allocation, _ := k.GetAllocation(ctx, allocationId)

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

