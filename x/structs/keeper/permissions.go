package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strconv"
	"strings"
)



// GetPermissionIDBytes returns the byte representation of the object and player id pair
func GetPermissionIDBytes(objectId []byte, playerId uint64) []byte {
	 id := fmt.Sprintf("%s@%d", string(objectId), playerId)
	 return []byte(id)

}


func (k Keeper) GetPermissionsByBytes(ctx sdk.Context, permissionId []byte) (types.Permission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PermissionKey))

	bz := store.Get(permissionId)

	// Player Capacity Not in Memory: no element
	if bz == nil {
		return types.Permissionless
	}

	load := types.Permission(binary.BigEndian.Uint64(bz))

	return load
}

func (k Keeper) SetPermissionsByBytes(ctx sdk.Context, permissionId []byte, permissions types.Permission) (types.Permission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PermissionKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(permissions))

	store.Set(permissionId, bz)

    keys := strings.Split(string(permissionId), "@")
    _ = ctx.EventManager().EmitTypedEvent(&types.EventPermission{Body: &types.EventPermissionBodyKeyPair{ObjectId: keys[0], PlayerId: keys[1], Value: uint64(permissions)}})

	return permissions
}

func (k Keeper) PermissionClearAll(ctx sdk.Context, objectId []byte, playerId uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerPermissionKey))

	permissionId := GetPermissionIDBytes(objectId, playerId)
	store.Delete(permissionId)

    keys := strings.Split(string(permissionId), "@")
    _ = ctx.EventManager().EmitTypedEvent(&types.EventPlayerPermission{Body: &types.EventPermissionBodyKeyPair{ObjectId: keys[0], PlayerId: keys[1], Value: uint64(0)}})

}

func (k Keeper) PermissionAdd(ctx sdk.Context, objectId []byte, playerId uint64, flag types.Permission) types.PlayerPermission {
    permissionRecord := GetPermissionIDBytes(objectId, playerId)

    currentPermission := k.GetPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.SetPermissionsByBytes(ctx, permissionRecord, currentPermission | flag)
	return newPermissions
}

func (k Keeper) PermissionRemove(ctx sdk.Context, objectId []byte, playerId uint64, flag types.Permission) types.Permission {
    permissionRecord := GetPermissionIDBytes(objectId, playerId)

    currentPermission := k.GetPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.SetPermissionsByBytes(ctx, permissionRecord, currentPermission &^ flag)
	return newPermissions
}

func (k Keeper) PermissionHasAll(ctx sdk.Context, objectId []byte, playerId uint64, flag types.Permission) bool {
    permissionRecord := GetPermissionIDBytes(objectId, playerId)

    currentPermission := k.GetPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag == flag
}

func (k Keeper) PermissionHasOneOf(ctx sdk.Context, objectId []byte, playerId uint64, flag types.Permission) bool {
    permissionRecord := GetPermissionIDBytes(objectId, playerId)

    currentPermission := k.GetPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag != 0
}


