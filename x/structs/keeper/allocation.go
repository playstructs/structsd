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
	if bz == nil {
		return 0
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

// AppendAllocation appends a allocation in the store with a new id and update the count
func (k Keeper) AppendAllocation(
	ctx sdk.Context,
	allocation types.Allocation,
) uint64 {
	// Create the allocation
	count := k.GetAllocationCount(ctx)

	// Set the ID of the appended value
	allocation.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	appendedValue := k.cdc.MustMarshal(&allocation)
	store.Set(GetAllocationIDBytes(allocation.Id), appendedValue)

	// Update allocation count
	k.SetAllocationCount(ctx, count+1)

	return count
}

// SetAllocation set a specific allocation in the store
func (k Keeper) SetAllocation(ctx sdk.Context, allocation types.Allocation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	b := k.cdc.MustMarshal(&allocation)
	store.Set(GetAllocationIDBytes(allocation.Id), b)
}

// GetAllocation returns a allocation from its id
func (k Keeper) GetAllocation(ctx sdk.Context, id uint64) (val types.Allocation, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	b := store.Get(GetAllocationIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveAllocation removes a allocation from the store
func (k Keeper) RemoveAllocation(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	store.Delete(GetAllocationIDBytes(id))
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


// GetAllSubstationAllocationIn returns all allocation
func (k Keeper) GetAllSubstationAllocationIn(ctx sdk.Context, substationId uint64) (list []types.Allocation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Allocation
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		if (val.DestinationId == substationId) {
		    list = append(list, val)
		}
	}

	return
}

// GetAllSubstationAllocationIn returns all allocation
func (k Keeper) GetAllSubstationAllocationPackagesIn(ctx sdk.Context, substationId uint64) (list []types.AllocationPackage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

    var status uint64
    var allocationPackage types.AllocationPackage

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Allocation
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		if (val.DestinationId == substationId) {
		    status = k.GetAllocationStatus(ctx, val.Id)
		    allocationPackage = types.AllocationPackage{Allocation: &val, Status: status,}
		    list = append(list, allocationPackage)
		}
	}

	return
}




// GetAllReactorAllocations returns all allocation relating to a reactor
func (k Keeper) GetAllReactorAllocations(ctx sdk.Context, reactorId uint64) (list []types.Allocation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Allocation
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		if (val.SourceId == reactorId) {
		    list = append(list, val)
		}
	}

	return
}



// GetAllocationIDBytes returns the byte representation of the ID
func GetAllocationIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetAllocationIDFromBytes returns ID in uint64 format from a byte array
func GetAllocationIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

// GetAllocationStatus returns the current power being generated by a Reactor
func (k Keeper) GetAllocationStatus(ctx sdk.Context, id uint64) uint64 {
    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationStatusKey))
	status := store.Get(GetAllocationIDBytes(id))

	// Status doesn't exist: no element
	if status == nil {
		return types.AllocationStatus_Offline
	}

	// Parse bytes
	return binary.BigEndian.Uint64(status)
}

// GetAllOnlineAllocation returns an array of all online allocations
// Mainly used for genesis export
func (k Keeper) GetAllOnlineAllocation(ctx sdk.Context) []uint64 {
    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationStatusKey))

    var status uint64
    var allocationId uint64
    var online []uint64


    iterator := sdk.KVStorePrefixIterator(store, []byte{})

    defer iterator.Close()

    for ; iterator.Valid(); iterator.Next() {
        status = binary.BigEndian.Uint64(iterator.Value())

        if (status == types.AllocationStatus_Online) {
            allocationIdB := iterator.Key()
            allocationId = GetAllocationIDFromBytes(allocationIdB)
            online = append(online, allocationId)
        }

    }

    return online

}


// SetReactorPower updates the cached state of available power in a reactor
func (k Keeper) SetAllocationStatus(ctx sdk.Context, id uint64, status uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationStatusKey))
	bz := make([]byte, 8)

	if ((status == types.AllocationStatus_Online) || (status == types.AllocationStatus_Offline)) {
    	binary.BigEndian.PutUint64(bz, status)
    	store.Set(GetReactorIDBytes(id), bz)
    }
}
