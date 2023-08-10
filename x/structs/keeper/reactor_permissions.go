package keeper

import (
	"encoding/binary"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	"strconv"
)



// GetReactorPermissionIDBytes returns the byte representation of the reactor and player id pair
func GetReactorPermissionIDBytes(reactorId uint64, playerId uint64) []byte {
	reactorIdString  := strconv.FormatUint(reactorId, 10)
	playerIdString := strconv.FormatUint(playerId, 10)

	return []byte(reactorIdString + "-" + playerIdString)
}


func (k Keeper) ReactorGetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte) (types.ReactorPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorPermissionKey))

	bz := store.Get(permissionRecord)

	// Substation Capacity Not in Memory: no element
	if bz == nil {
		return types.ReactorPermissionless
	}

	load := types.ReactorPermission(binary.BigEndian.Uint16(bz))

	return load
}

func (k Keeper) ReactorSetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte, permissions types.ReactorPermission) (types.ReactorPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorPermissionKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint16(bz, uint16(permissions))

	store.Set(permissionRecord, bz)

	return permissions
}

func (k Keeper) ReactorPermissionClearAll(ctx sdk.Context, reactorId uint64, playerId uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorPermissionKey))
	store.Delete(GetReactorPermissionIDBytes(reactorId, playerId))
}

func (k Keeper) ReactorPermissionAdd(ctx sdk.Context, reactorId uint64, playerId uint64, flag types.ReactorPermission) types.ReactorPermission {
    permissionRecord := GetReactorPermissionIDBytes(reactorId, playerId)

    currentPermission := k.ReactorGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.ReactorSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission | flag)
	return newPermissions
}

func (k Keeper) ReactorPermissionRemove(ctx sdk.Context, reactorId uint64, playerId uint64, flag types.ReactorPermission) types.ReactorPermission {
    permissionRecord := GetReactorPermissionIDBytes(reactorId, playerId)

    currentPermission := k.ReactorGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.ReactorSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission &^ flag)
	return newPermissions
}

func (k Keeper) ReactorPermissionHasAll(ctx sdk.Context, reactorId uint64, playerId uint64, flag types.ReactorPermission) bool {
    permissionRecord := GetReactorPermissionIDBytes(reactorId, playerId)

    currentPermission := k.ReactorGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag == flag
}

func (k Keeper) ReactorPermissionHasOneOf(ctx sdk.Context, reactorId uint64, playerId uint64, flag types.ReactorPermission) bool {
    permissionRecord := GetReactorPermissionIDBytes(reactorId, playerId)

    currentPermission := k.ReactorGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag != 0
}