package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// GetAddressCount get the total number of address
func (k Keeper) GetAddressCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.AddressCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetAddressCount set the total number of address
func (k Keeper) SetAddressCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.AddressCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendAddress appends a address in the store with a new id and update the count
func (k Keeper) AppendAddress(
	ctx sdk.Context,
	address types.Address,
) uint64 {
	// Create the address
	count := k.GetAddressCount(ctx)

	// Set the ID of the appended value
	address.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressKey))
	appendedValue := k.cdc.MustMarshal(&address)
	store.Set(GetAddressIDBytes(address.Id), appendedValue)

	// Update address count
	k.SetAddressCount(ctx, count+1)

	return count
}

// SetAddress set a specific address in the store
func (k Keeper) SetAddress(ctx sdk.Context, address types.Address) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressKey))
	b := k.cdc.MustMarshal(&address)
	store.Set(GetAddressIDBytes(address.Id), b)
}

// GetAddress returns a address from its id
func (k Keeper) GetAddress(ctx sdk.Context, id uint64) (val types.Address, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressKey))
	b := store.Get(GetAddressIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveAddress removes a address from the store
func (k Keeper) RemoveAddress(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressKey))
	store.Delete(GetAddressIDBytes(id))
}

// GetAllAddress returns all address
func (k Keeper) GetAllAddress(ctx sdk.Context) (list []types.Address) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Address
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetAddressIDBytes returns the byte representation of the ID
func GetAddressIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetAddressIDFromBytes returns ID in uint64 format from a byte array
func GetAddressIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}



func (k Keeper) GetPlayerIdFromAddress(ctx sdk.Context, address string) (uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressPlayerKey))

	bz := store.Get([]byte(address))

	// Substation Capacity Not in Memory: no element
	if bz == nil {
		return 0
	}

	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetPlayerIdForAddress(ctx sdk.Context, address string, playerId uint64)  {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressPlayerKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, playerId)

	store.Set([]byte(address), bz)

}
