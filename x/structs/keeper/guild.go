package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	"strconv"
)

// GetGuildCount get the total number of guild
func (k Keeper) GetGuildCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.GuildCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetGuildCount set the total number of guild
func (k Keeper) SetGuildCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.GuildCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendGuild appends a guild in the store with a new id and update the count
func (k Keeper) AppendGuild(
	ctx sdk.Context,
	guild types.Guild,
) uint64 {
	// Create the guild
	count := k.GetGuildCount(ctx)

	// Set the ID of the appended value
	guild.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildKey))
	appendedValue := k.cdc.MustMarshal(&guild)
	store.Set(GetGuildIDBytes(guild.Id), appendedValue)

	// Update guild count
	k.SetGuildCount(ctx, count+1)

	return count
}

// SetGuild set a specific guild in the store
func (k Keeper) SetGuild(ctx sdk.Context, guild types.Guild) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildKey))
	b := k.cdc.MustMarshal(&guild)
	store.Set(GetGuildIDBytes(guild.Id), b)
}

// GetGuild returns a guild from its id
func (k Keeper) GetGuild(ctx sdk.Context, id uint64) (val types.Guild, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildKey))
	b := store.Get(GetGuildIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveGuild removes a guild from the store
func (k Keeper) RemoveGuild(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildKey))
	store.Delete(GetGuildIDBytes(id))
}

// GetAllGuild returns all guild
func (k Keeper) GetAllGuild(ctx sdk.Context) (list []types.Guild) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Guild
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetGuildIDBytes returns the byte representation of the ID
func GetGuildIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetGuildIDFromBytes returns ID in uint64 format from a byte array
func GetGuildIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}




// GetGuildPermissionIDBytes returns the byte representation of the guild and player id pair
func GetGuildPermissionIDBytes(guildId uint64, playerId uint64) []byte {
	guildIdString  := strconv.FormatUint(guildId, 10)
	playerIdString := strconv.FormatUint(playerId, 10)

	return []byte(guildIdString + "-" + playerIdString)
}


func (k Keeper) GuildGetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte) (types.GuildPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildPermissionKey))

	bz := store.Get(permissionRecord)

	// Substation Capacity Not in Memory: no element
	if bz == nil {
		return types.GuildPermissionless
	}

	load := types.GuildPermission(binary.BigEndian.Uint16(bz))

	return load
}

func (k Keeper) GuildSetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte, permissions types.GuildPermission) (types.GuildPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildPermissionKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint16(bz, uint16(permissions))

	store.Set(permissionRecord, bz)

	return permissions
}

func (k Keeper) GuildPermissionClearAll(ctx sdk.Context, guildId uint64, playerId uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildPermissionKey))
	store.Delete(GetGuildPermissionIDBytes(guildId, playerId))
}

func (k Keeper) GuildPermissionAdd(ctx sdk.Context, guildId uint64, playerId uint64, flag types.GuildPermission) types.GuildPermission {
    permissionRecord := GetGuildPermissionIDBytes(guildId, playerId)

    currentPermission := k.GuildGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.GuildSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission | flag)
	return newPermissions
}

func (k Keeper) GuildPermissionRemove(ctx sdk.Context, guildId uint64, playerId uint64, flag types.GuildPermission) types.GuildPermission {
    permissionRecord := GetGuildPermissionIDBytes(guildId, playerId)

    currentPermission := k.GuildGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.GuildSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission &^ flag)
	return newPermissions
}

func (k Keeper) GuildPermissionHasAll(ctx sdk.Context, guildId uint64, playerId uint64, flag types.GuildPermission) bool {
    permissionRecord := GetGuildPermissionIDBytes(guildId, playerId)

    currentPermission := k.GuildGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag == flag
}

func (k Keeper) GuildPermissionHasOneOf(ctx sdk.Context, guildId uint64, playerId uint64, flag types.GuildPermission) bool {
    permissionRecord := GetGuildPermissionIDBytes(guildId, playerId)

    currentPermission := k.GuildGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag != 0
}