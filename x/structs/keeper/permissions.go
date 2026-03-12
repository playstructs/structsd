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
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPermission{&types.PermissionRecord{PermissionId: string(permissionId), Value: uint64(permissions)}})

	return permissions
}

func (k Keeper) PermissionClearAll(ctx context.Context, permissionId []byte) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PermissionKey))

	store.Delete(permissionId)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPermission{&types.PermissionRecord{PermissionId: string(permissionId), Value: 0}})

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

// GetAllPermissionExport returns all grid attributes
func (k Keeper) GetAllPermissionExport(ctx context.Context) (list []*types.PermissionRecord) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PermissionKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		list = append(list, &types.PermissionRecord{PermissionId: string(iterator.Key()), Value: binary.BigEndian.Uint64(iterator.Value())})
	}

	return
}

// ClearPermissionByObject deletes all permission entries for the given objectId
// (all keys objectId@*). Uses prefix iteration so cost is O(players with permissions on this object).
func (k Keeper) ClearPermissionByObject(ctx context.Context, objectId string) (list []string) {
	if objectId == "" {
		return
	}
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PermissionKey))
	prefixBytes := []byte(objectId + "@")
	iterator := storetypes.KVStorePrefixIterator(store, prefixBytes)
	defer iterator.Close()

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		store.Delete(key)
		list = append(list, string(key))
		_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPermission{&types.PermissionRecord{PermissionId: string(key), Value: 0}})
	}

	return
}

/*
 * Permission Guild Rank System
 *
 */


func GuildRankPermissionID(objectId string, guildId string, permission types.Permission) string {
	return fmt.Sprintf("%s/%s/%d", objectId, guildId, permission)
}

func GuildRankKeyPrefix(objectId string, guildId string) []byte {
    return []byte(types.PermissionGuildRank + objectId + "/" + guildId + "/")
}

func ObjectOnlyGuildRankKeyPrefix(objectId string) []byte {
    return []byte(types.PermissionGuildRank + objectId + "/" )
}

// SetHighestGuildRankPermissionStoreOnly writes the guild rank permission to the store without emitting an event. Used in InitGenesis.
func (k Keeper) SetHighestGuildRankPermissionStoreOnly(ctx context.Context, objectId string, guildId string, permissionType types.Permission, highestRank uint64) {
	guildRankStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), GuildRankKeyPrefix(objectId, guildId))
	binaryPermission := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryPermission, uint64(permissionType))
	binaryRank := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryRank, highestRank)
	guildRankStore.Set(binaryPermission, binaryRank)
}

func (k Keeper) SetHighestGuildRankPermission(ctx context.Context, objectId string, guildId string, permissionType types.Permission, highestRank uint64) (err error) {
	guildRankStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), GuildRankKeyPrefix(objectId, guildId))

	binaryPermission := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryPermission, uint64(permissionType))

	binaryRank := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryRank, highestRank)

	guildRankStore.Set(binaryPermission, binaryRank)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildRankPermission{
        GuildRankPermissionRecord: &types.GuildRankPermissionRecord{
            ObjectId:    objectId,
            GuildId:     guildId,
            Permissions:  uint64(permissionType),
            Rank:         highestRank,
        },
    })

    return err
}

func (k Keeper) RemoveGuildRankPermission(ctx context.Context, objectId string, guildId string, permissionType types.Permission) (err error) {
    guildRankStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), GuildRankKeyPrefix(objectId, guildId))

    binaryPermission := make([]byte, 8)
    binary.BigEndian.PutUint64(binaryPermission, uint64(permissionType))

    guildRankStore.Delete(binaryPermission)

    ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildRankPermission{
        GuildRankPermissionRecord: &types.GuildRankPermissionRecord{
            ObjectId:    objectId,
            GuildId:     guildId,
            Permissions:  uint64(permissionType),
            Rank:         0, // 0 indicates removal for indexers
        },
    })

    return err
}


func (k Keeper) GetHighestGuildRankForPermission(ctx context.Context, objectId string, guildId string, permissionType types.Permission) (uint64, bool) {
	guildRankStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), GuildRankKeyPrefix(objectId, guildId))

	binaryPermission := make([]byte, 8)
	binary.BigEndian.PutUint64(binaryPermission, uint64(permissionType))

	binaryHighestRank := guildRankStore.Get(binaryPermission)

	if binaryHighestRank == nil {
		return 0, false
	}

	highestRank := binary.BigEndian.Uint64(binaryHighestRank)
	return highestRank, true
}

func (k Keeper) GetAllGuildRankPermissions(ctx context.Context, objectId string, guildId string) (list [][]byte) {
    guildRankStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)),  GuildRankKeyPrefix(objectId, guildId))
    iterator := storetypes.KVStorePrefixIterator(guildRankStore, []byte{})

    defer iterator.Close()

    for ; iterator.Valid(); iterator.Next() {
        list = append(list, iterator.Key())
    }

    return
}

func (k Keeper) ClearAllGuildRankPermissions(ctx context.Context, objectId string, guildId string, list [][]byte) {
	guildRankStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), GuildRankKeyPrefix(objectId, guildId))
	for _, key := range list {
		guildRankStore.Delete(key)
	}
}

// ClearPermissionGuildRankByObject deletes all guild rank permission entries for the given objectId.
// Must only be called when the object is actually being deleted (e.g. from ClearPermissionsForObject in the same flow as object deletion).
// It must not be exposed to arbitrary callers or used for "revoke all guild access" without going through object-deletion semantics.
func (k Keeper) ClearPermissionGuildRankByObject(ctx context.Context, objectId string) (list []string) {
	if objectId == "" {
		return
	}
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PermissionGuildRank))
	prefixBytes := []byte(objectId + "/")
	iterator := storetypes.KVStorePrefixIterator(store, prefixBytes)
	defer iterator.Close()

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		store.Delete(key)
		list = append(list, string(key))

		// Decode key: objectId + "/" + guildId + "/" + 8-byte permission; rest = guildId + "/" + 8 bytes
		if len(key) > len(prefixBytes) {
			rest := key[len(prefixBytes):]
			if len(rest) >= 9 { // at least "x/"+ 8 bytes
				guildId := string(rest[:len(rest)-9]) // guildId + "/"
				guildId = strings.TrimSuffix(guildId, "/")
				permBytes := rest[len(rest)-8:]
				permVal := binary.BigEndian.Uint64(permBytes)
				_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildRankPermission{
					GuildRankPermissionRecord: &types.GuildRankPermissionRecord{
						ObjectId:    objectId,
						GuildId:     guildId,
						Permissions: permVal,
						Rank:        0, // removal
					},
				})
			}
		}
	}
	return
}

// GetAllGuildRankPermissionExport iterates all guild rank permission entries for genesis export.
func (k Keeper) GetAllGuildRankPermissionExport(ctx context.Context) (list []*types.GuildRankPermissionRecord) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PermissionGuildRank))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		value := iterator.Value()
		if len(key) < 10 || len(value) != 8 {
			continue
		}
		// key: objectId + "/" + guildId + "/" + 8-byte permission
		firstSlash := strings.Index(string(key), "/")
		if firstSlash < 0 {
			continue
		}
		lastSlash := len(key) - 9 // position of "/" before 8-byte permission
		if lastSlash <= firstSlash {
			continue
		}
		objectId := string(key[:firstSlash])
		guildId := string(key[firstSlash+1 : lastSlash])
		permVal := binary.BigEndian.Uint64(key[len(key)-8:])
		rank := binary.BigEndian.Uint64(value)
		list = append(list, &types.GuildRankPermissionRecord{
			ObjectId:    objectId,
			GuildId:     guildId,
			Permissions: permVal,
			Rank:        rank,
		})
	}
	return
}


