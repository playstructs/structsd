package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

)

// GetStructCount get the total number of struct
func (k Keeper) GetStructCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.StructCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0  {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetStructCount set the total number of struct
func (k Keeper) SetStructCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.StructCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendStruct appends a struct in the store with a new id and update the count
func (k Keeper) AppendStruct(
	ctx sdk.Context,
	//struct types.Struct,
	player types.Player,
	structType string,
	planet types.Planet,
	slot uint64,
) (structure types.Struct) {
    structure = types.CreateBaseStruct(structType )

	// Create the struct
	count := k.GetStructCount(ctx)

	// Set the ID of the appended value
	structure.Id = count
	structure.SetCreator(player.Creator)
	structure.SetOwner(player.Id)
	structure.SetSlot(slot)
	structure.SetBuildStartBlock(uint64(ctx.BlockHeight()))

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StructKey))
	appendedValue := k.cdc.MustMarshal(&structure)
	store.Set(GetStructIDBytes(structure.Id), appendedValue)

	// Update struct count
	k.SetStructCount(ctx, count+1)


	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: structure.Id, ObjectType: types.ObjectType_struct})

	return structure
}

// SetStruct set a specific struct in the store
func (k Keeper) SetStruct(ctx sdk.Context, structure types.Struct) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StructKey))
	b := k.cdc.MustMarshal(&structure)
	store.Set(GetStructIDBytes(structure.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: structure.Id, ObjectType: types.ObjectType_struct})
}

// GetStruct returns a struct from its id
func (k Keeper) GetStruct(ctx sdk.Context, id uint64) (val types.Struct, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StructKey))
	b := store.Get(GetStructIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// RemoveStruct removes a struct from the store
func (k Keeper) RemoveStruct(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StructKey))
	store.Delete(GetStructIDBytes(id))
}

// GetAllStruct returns all struct
func (k Keeper) GetAllStruct(ctx sdk.Context) (list []types.Struct) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StructKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Struct
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		list = append(list, val)
	}

	return
}

// GetStructIDBytes returns the byte representation of the ID
func GetStructIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetStructIDFromBytes returns ID in uint64 format from a byte array
func GetStructIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

