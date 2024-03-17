package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//sdkerrors "cosmossdk.io/errors"

	"fmt"
	"strings"
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

func (k Keeper) GetPermissionsByBytes(ctx context.Context, permissionId []byte) (types.Permission) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PermissionKey))

	bz := store.Get(permissionId)

	// Player Capacity Not in Memory: no element
	if bz == nil {
		return types.Permissionless
	}

	load := types.Permission(binary.BigEndian.Uint64(bz))

	return load
}

func (k Keeper) SetPermissionsByBytes(ctx context.Context, permissionId []byte, permissions types.Permission) (types.Permission) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PermissionKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(permissions))

	store.Set(permissionId, bz)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPermission{PermissionId: string(permissionId), Value: uint64(permissions)})

	return permissions
}

func (k Keeper) PermissionClearAll(ctx context.Context, permissionId []byte) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PermissionKey))

	store.Delete(permissionId)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPermission{PermissionId: string(permissionId), Value: 0})

}

func (k Keeper) PermissionAdd(ctx context.Context, permissionId []byte, flag types.Permission) types.Permission {
    currentPermission := k.GetPermissionsByBytes(ctx, permissionId)
    newPermissions := k.SetPermissionsByBytes(ctx, permissionId, currentPermission | flag)
	return newPermissions
}

func (k Keeper) PermissionRemove(ctx context.Context, permissionId []byte, flag types.Permission) types.Permission {
    currentPermission := k.GetPermissionsByBytes(ctx, permissionId)
    newPermissions := k.SetPermissionsByBytes(ctx, permissionId, currentPermission &^ flag)
	return newPermissions
}

func (k Keeper) PermissionHasAll(ctx context.Context, permissionId []byte, flag types.Permission) bool {
    currentPermission := k.GetPermissionsByBytes(ctx, permissionId)
	return currentPermission&flag == flag
}

func (k Keeper) PermissionHasOneOf(ctx context.Context, permissionId []byte, flag types.Permission) bool {
    currentPermission := k.GetPermissionsByBytes(ctx, permissionId)
	return currentPermission&flag != 0
}


func (k Keeper) GetPermissionsByObject(ctx context.Context, objectId string) (list []types.PermissionRecord) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PermissionKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
        extractedId := strings.Split(string(iterator.Key()), "@")
        if (extractedId[0] == objectId) {
            list = append(list, types.PermissionRecord{PermissionId: string(iterator.Key()), Value: binary.BigEndian.Uint64(iterator.Value())})
		}
	}
	return
}


func (k Keeper) GetPermissionsByPlayer(ctx context.Context, playerId string) (list []types.PermissionRecord) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PermissionKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
        extractedId := strings.Split(string(iterator.Key()), "@")
        if (extractedId[1] == playerId) {
            list = append(list, types.PermissionRecord{PermissionId: string(iterator.Key()), Value: binary.BigEndian.Uint64(iterator.Value())})
		}
	}
	return
}