package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// GetAllocationProposalCount get the total number of allocationProposal
func (k Keeper) GetAllocationProposalCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.AllocationProposalCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetAllocationProposalCount set the total number of allocationProposal
func (k Keeper) SetAllocationProposalCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.AllocationProposalCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendAllocationProposal appends a allocationProposal in the store with a new id and update the count
func (k Keeper) AppendAllocationProposal(
	ctx sdk.Context,
	allocationProposal types.AllocationProposal,
) uint64 {
	// Create the allocationProposal
	count := k.GetAllocationProposalCount(ctx)

	// Set the ID of the appended value
	allocationProposal.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationProposalKey))
	appendedValue := k.cdc.MustMarshal(&allocationProposal)
	store.Set(GetAllocationProposalIDBytes(allocationProposal.Id), appendedValue)

	// Update allocationProposal count
	k.SetAllocationProposalCount(ctx, count+1)

	return count
}

// SetAllocationProposal set a specific allocationProposal in the store
func (k Keeper) SetAllocationProposal(ctx sdk.Context, allocationProposal types.AllocationProposal) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationProposalKey))
	b := k.cdc.MustMarshal(&allocationProposal)
	store.Set(GetAllocationProposalIDBytes(allocationProposal.Id), b)
}

// GetAllocationProposal returns a allocationProposal from its id
func (k Keeper) GetAllocationProposal(ctx sdk.Context, id uint64) (val types.AllocationProposal, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationProposalKey))
	b := store.Get(GetAllocationProposalIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveAllocationProposal removes a allocationProposal from the store
func (k Keeper) RemoveAllocationProposal(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationProposalKey))
	store.Delete(GetAllocationProposalIDBytes(id))
}

// GetAllAllocationProposal returns all allocationProposal
func (k Keeper) GetAllAllocationProposal(ctx sdk.Context) (list []types.AllocationProposal) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AllocationProposalKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.AllocationProposal
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetAllocationProposalIDBytes returns the byte representation of the ID
func GetAllocationProposalIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetAllocationProposalIDFromBytes returns ID in uint64 format from a byte array
func GetAllocationProposalIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}
