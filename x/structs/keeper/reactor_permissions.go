package keeper

import (
	"encoding/binary"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	"strconv"
	"strings"


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

	load := types.ReactorPermission(binary.BigEndian.Uint64(bz))

	return load
}

func (k Keeper) ReactorSetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte, permissions types.ReactorPermission) (types.ReactorPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorPermissionKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(permissions))

	store.Set(permissionRecord, bz)

	keys := strings.Split(string(permissionRecord), "-")
	_ = ctx.EventManager().EmitTypedEvent(&types.EventReactorPermission{Body: &types.EventPermissionBodyKeyPair{ObjectId: keys[0], PlayerId: keys[1], Value: uint64(permissions)}})

	return permissions
}

func (k Keeper) ReactorPermissionClearAll(ctx sdk.Context, reactorId uint64, playerId uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorPermissionKey))

	permissionId := GetReactorPermissionIDBytes(reactorId, playerId)
	store.Delete(permissionId)

    keys := strings.Split(string(permissionId), "-")
    _ = ctx.EventManager().EmitTypedEvent(&types.EventReactorPermission{Body: &types.EventPermissionBodyKeyPair{ObjectId: keys[0], PlayerId: keys[1], Value: uint64(0)}})

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
