package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
    //sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"

)
func GetStructTypeIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetStructTypeCount get the total number of struct types
func (k Keeper) GetStructTypeCount(ctx context.Context) uint64 {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.StructTypeCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0  {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetStructTypeCount set the total number of struct types
func (k Keeper) SetStructTypeCount(ctx context.Context, count uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.StructTypeCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendStructType appends a struct type in the store with a new id and update the count
func (k Keeper) AppendStructType(
	ctx context.Context,
	structType types.StructType,
) (types.StructType) {
 	ctxSDK := sdk.UnwrapSDKContext(ctx)

	// Create the struct
	count := k.GetStructTypeCount(ctx)

	// Set the ID of the appended value
	structType.Id = count

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.StructTypeKey))
	appendedValue := k.cdc.MustMarshal(&structType)
	store.Set(GetStructTypeIDBytes(structType.Id), appendedValue)

	// Update struct count
	k.SetStructTypeCount(ctx, count+1)

    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventStructType{StructType: &structType})

	return structType
}

// SetStructType set a specific struct type in the store
func (k Keeper) SetStructType(ctx context.Context, structType types.StructType) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.StructTypeKey))
	b := k.cdc.MustMarshal(&structType)
	store.Set(GetStructTypeIDBytes(structType.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventStructType{StructType: &structType})
}

// GetStructType returns a struct type from its id
func (k Keeper) GetStructType(ctx context.Context, structTypeId uint64) (val types.StructType, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.StructTypeKey))
	b := store.Get(GetStructTypeIDBytes(structTypeId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}



// GetAllStructType returns all struct types
func (k Keeper) GetAllStructType(ctx context.Context) (list []types.StructType) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.StructTypeKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.StructType
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		list = append(list, val)
	}

	return
}



