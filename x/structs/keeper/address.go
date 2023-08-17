package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)



func (k Keeper) GetPlayerIdFromAddress(ctx sdk.Context, address string) (uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressPlayerKey))

	bz := store.Get(types.KeyPrefix(address))

	// Address Not in Memory: no element
	if bz == nil  {
		return 0
	}

	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetPlayerIdForAddress(ctx sdk.Context, address string, playerId uint64)  {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressPlayerKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, playerId)

	store.Set(types.KeyPrefix(address), bz)

}



func (k Keeper) AddressGetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte) (types.AddressPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressPermissionKey))

	bz := store.Get(permissionRecord)

	// Substation Capacity Not in Memory: no element
	if bz == nil {
		return types.AddressPermissionless
	}

	load := types.AddressPermission(binary.BigEndian.Uint16(bz))

	return load
}

func (k Keeper) AddressSetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte, permissions types.AddressPermission) (types.AddressPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressPermissionKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint16(bz, uint16(permissions))

	store.Set(permissionRecord, bz)

	return permissions
}

func (k Keeper) AddressPermissionClearAll(ctx sdk.Context, address string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressPermissionKey))
	store.Delete(types.KeyPrefix(address))
}

func (k Keeper) AddressPermissionAdd(ctx sdk.Context, address string, flag types.AddressPermission) types.AddressPermission {
    permissionRecord := types.KeyPrefix(address)

    currentPermission := k.AddressGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.AddressSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission | flag)
	return newPermissions
}

func (k Keeper) AddressPermissionRemove(ctx sdk.Context, address string, flag types.AddressPermission) types.AddressPermission {
    permissionRecord := types.KeyPrefix(address)

    currentPermission := k.AddressGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.AddressSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission &^ flag)
	return newPermissions
}

func (k Keeper) AddressPermissionHasAll(ctx sdk.Context, address string, flag types.AddressPermission) bool {
    permissionRecord := types.KeyPrefix(address)

    currentPermission := k.AddressGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag == flag
}

func (k Keeper) AddressPermissionHasOneOf(ctx sdk.Context, address string, flag types.AddressPermission) bool {
    permissionRecord := types.KeyPrefix(address)

    currentPermission := k.AddressGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag != 0
}
