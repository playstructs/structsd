package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// GetReactorCount get the total number of reactor
func (k Keeper) GetReactorCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.ReactorCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetReactorCount set the total number of reactor
func (k Keeper) SetReactorCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.ReactorCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendReactor appends a reactor in the store with a new id and update the count
func (k Keeper) AppendReactor(
	ctx sdk.Context,
	reactor types.Reactor,
) uint64 {
	// Create the reactor
	count := k.GetReactorCount(ctx)

	// Set the ID of the appended value
	reactor.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	appendedValue := k.cdc.MustMarshal(&reactor)
	store.Set(GetReactorIDBytes(reactor.Id), appendedValue)

	// Update reactor count
	k.SetReactorCount(ctx, count+1)

	return count
}

// SetReactor set a specific reactor in the store
func (k Keeper) SetReactor(ctx sdk.Context, reactor types.Reactor) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	b := k.cdc.MustMarshal(&reactor)
	store.Set(GetReactorIDBytes(reactor.Id), b)
}

// GetReactor returns a reactor from its id
func (k Keeper) GetReactor(ctx sdk.Context, id uint64) (val types.Reactor, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	b := store.Get(GetReactorIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveReactor removes a reactor from the store
func (k Keeper) RemoveReactor(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	store.Delete(GetReactorIDBytes(id))
}

// GetAllReactor returns all reactor
func (k Keeper) GetAllReactor(ctx sdk.Context) (list []types.Reactor) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Reactor
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetReactorIDBytes returns the byte representation of the ID
func GetReactorIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetReactorIDFromBytes returns ID in uint64 format from a byte array
func GetReactorIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}
