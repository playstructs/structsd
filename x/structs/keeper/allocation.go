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
