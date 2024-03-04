package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"fmt"
)



// GetObjectPermissionIDBytes returns the byte representation of the object and player id pair
func GetObjectPermissionIDBytes(objectId string, playerId string) []byte {
	 id := fmt.Sprintf("%s@%s", objectId, playerId)
	 return []byte(id)
}

// GetAddressPermissionIDBytes returns the byte representation of the Address, based on ObjectType
func GetAddressPermissionIDBytes(address string) []byte {
    id := fmt.Sprintf("%d-%s@0", types.ObjectType_address, address)
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

    //keys := strings.Split(string(permissionId), "@")
    //_ = ctx.EventManager().EmitTypedEvent(&types.EventPermission{Body: &types.EventPermissionBodyKeyPair{ObjectId: keys[0], PlayerId: keys[1], Value: uint64(permissions)}})

	return permissions
}

func (k Keeper) PermissionClearAll(ctx sdk.Context, permissionId []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PermissionKey))

	store.Delete(permissionId)

    //keys := strings.Split(string(permissionId), "@")
    //_ = ctx.EventManager().EmitTypedEvent(&types.EventPermission{Body: &types.EventPermissionBodyKeyPair{ObjectId: keys[0], PlayerId: keys[1], Value: uint64(0)}})

}

func (k Keeper) PermissionAdd(ctx sdk.Context, permissionId []byte, flag types.Permission) types.Permission {
    currentPermission := k.GetPermissionsByBytes(ctx, permissionId)
    newPermissions := k.SetPermissionsByBytes(ctx, permissionId, currentPermission | flag)
	return newPermissions
}

func (k Keeper) PermissionRemove(ctx sdk.Context, permissionId []byte, flag types.Permission) types.Permission {
    currentPermission := k.GetPermissionsByBytes(ctx, permissionId)
    newPermissions := k.SetPermissionsByBytes(ctx, permissionId, currentPermission &^ flag)
	return newPermissions
}

func (k Keeper) PermissionHasAll(ctx sdk.Context, permissionId []byte, flag types.Permission) bool {
    currentPermission := k.GetPermissionsByBytes(ctx, permissionId)
	return currentPermission&flag == flag
}

func (k Keeper) PermissionHasOneOf(ctx sdk.Context, permissionId []byte, flag types.Permission) bool {
    currentPermission := k.GetPermissionsByBytes(ctx, permissionId)
	return currentPermission&flag != 0
}


