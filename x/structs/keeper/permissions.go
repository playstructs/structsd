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

	var keysToDelete [][]byte
	for ; iterator.Valid(); iterator.Next() {
		keysToDelete = append(keysToDelete, append([]byte(nil), iterator.Key()...))
	}
	iterator.Close()

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	for _, key := range keysToDelete {
		store.Delete(key)
		list = append(list, string(key))
		_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPermission{&types.PermissionRecord{PermissionId: string(key), Value: 0}})
	}

	return
}

/*
 * Permission Guild Rank System
 *
 * Each (objectId, guildId) pair stores a single register of PermissionBitCount uint64 slots.
 * Slot i holds the worst-allowed rank for permission bit i (0 = no record).
 */

func GuildRankRegisterKey(objectId string, guildId string) []byte {
	return []byte(types.PermissionGuildRank + objectId + "/" + guildId)
}

func GuildRankObjectPrefix(objectId string) []byte {
	return []byte(types.PermissionGuildRank + objectId + "/")
}

func (k Keeper) ReadGuildRankRegister(ctx context.Context, objectId string, guildId string) [types.PermissionBitCount]uint64 {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	var register [types.PermissionBitCount]uint64
	data := store.Get(GuildRankRegisterKey(objectId, guildId))
	if data == nil || len(data) < 8 {
		return register
	}
	storedSlots := len(data) / 8
	if storedSlots > types.PermissionBitCount {
		storedSlots = types.PermissionBitCount
	}
	for i := 0; i < storedSlots; i++ {
		register[i] = binary.BigEndian.Uint64(data[i*8 : i*8+8])
	}
	return register
}

func (k Keeper) WriteGuildRankRegister(ctx context.Context, objectId string, guildId string, register [types.PermissionBitCount]uint64) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := GuildRankRegisterKey(objectId, guildId)

	allZero := true
	for i := 0; i < types.PermissionBitCount; i++ {
		if register[i] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		store.Delete(key)
		return
	}

	data := make([]byte, types.PermissionRegisterSize)
	for i := 0; i < types.PermissionBitCount; i++ {
		binary.BigEndian.PutUint64(data[i*8:i*8+8], register[i])
	}
	store.Set(key, data)
}

// SetGuildRankPermission writes the register and emits events for changed bits only.
func (k Keeper) SetGuildRankPermission(ctx context.Context, objectId string, guildId string, register [types.PermissionBitCount]uint64, changedBits types.Permission) {
	k.WriteGuildRankRegister(ctx, objectId, guildId, register)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	for bit := 0; bit < types.PermissionBitCount; bit++ {
		if uint64(changedBits)&(1<<bit) != 0 {
			_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildRankPermission{
				GuildRankPermissionRecord: &types.GuildRankPermissionRecord{
					ObjectId:    objectId,
					GuildId:     guildId,
					Permissions: 1 << bit,
					Rank:        register[bit],
				},
			})
		}
	}
}

// SetGuildRankPermissionStoreOnly writes guild rank permissions without events. Used in InitGenesis.
func (k Keeper) SetGuildRankPermissionStoreOnly(ctx context.Context, objectId string, guildId string, permissionType types.Permission, worstAllowedRank uint64) {
	register := k.ReadGuildRankRegister(ctx, objectId, guildId)
	for bit := 0; bit < types.PermissionBitCount; bit++ {
		if uint64(permissionType)&(1<<bit) != 0 {
			register[bit] = worstAllowedRank
		}
	}
	k.WriteGuildRankRegister(ctx, objectId, guildId, register)
}

// ClearPermissionGuildRankByObject deletes all guild rank registers for the given objectId.
func (k Keeper) ClearPermissionGuildRankByObject(ctx context.Context, objectId string) {
	if objectId == "" {
		return
	}
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PermissionGuildRank))
	prefixBytes := []byte(objectId + "/")
	iterator := storetypes.KVStorePrefixIterator(store, prefixBytes)

	type entry struct {
		keyCopy []byte
		guildId string
		register [types.PermissionBitCount]uint64
	}
	var entries []entry

	for ; iterator.Valid(); iterator.Next() {
		kCopy := append([]byte(nil), iterator.Key()...)
		guildId := strings.TrimPrefix(string(kCopy), string(prefixBytes))
		var reg [types.PermissionBitCount]uint64
		data := iterator.Value()
		if len(data) >= 8 {
			storedSlots := len(data) / 8
			if storedSlots > types.PermissionBitCount {
				storedSlots = types.PermissionBitCount
			}
			for i := 0; i < storedSlots; i++ {
				reg[i] = binary.BigEndian.Uint64(data[i*8 : i*8+8])
			}
		}
		entries = append(entries, entry{keyCopy: kCopy, guildId: guildId, register: reg})
	}
	iterator.Close()

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	for _, e := range entries {
		store.Delete(e.keyCopy)
		for bit := 0; bit < types.PermissionBitCount; bit++ {
			if e.register[bit] != 0 {
				_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildRankPermission{
					GuildRankPermissionRecord: &types.GuildRankPermissionRecord{
						ObjectId:    objectId,
						GuildId:     e.guildId,
						Permissions: 1 << bit,
						Rank:        0,
					},
				})
			}
		}
	}
}

// GetAllGuildRankPermissionExport iterates all guild rank registers for genesis export.
func (k Keeper) GetAllGuildRankPermissionExport(ctx context.Context) (list []*types.GuildRankPermissionRecord) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PermissionGuildRank))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := string(iterator.Key())
		data := iterator.Value()
		if len(data) < 8 {
			continue
		}

		slash := strings.Index(key, "/")
		if slash < 0 {
			continue
		}
		objectId := key[:slash]
		guildId := key[slash+1:]

		storedSlots := len(data) / 8
		if storedSlots > types.PermissionBitCount {
			storedSlots = types.PermissionBitCount
		}
		for bit := 0; bit < storedSlots; bit++ {
			rank := binary.BigEndian.Uint64(data[bit*8 : bit*8+8])
			if rank != 0 {
				list = append(list, &types.GuildRankPermissionRecord{
					ObjectId:    objectId,
					GuildId:     guildId,
					Permissions: 1 << bit,
					Rank:        rank,
				})
			}
		}
	}
	return
}
