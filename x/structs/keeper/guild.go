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
	if bz == nil || binary.BigEndian.Uint64(bz) == 0  {
		return types.KeeperStartValue
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
	//guild types.Guild,
	endpoint string,
	reactor types.Reactor,
	player types.Player,
) (guild types.Guild) {
    guild = types.CreateEmptyGuild()

	// Create the guild
	count := k.GetGuildCount(ctx)

	// Set the ID of the appended value
	guild.Id = count
	guild.SetEndpoint(endpoint)
	guild.SetCreator(player.Creator)
	guild.SetOwner(player.Id)
	guild.SetPrimaryReactorId(reactor.Id)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildKey))
	appendedValue := k.cdc.MustMarshal(&guild)
	store.Set(GetGuildIDBytes(guild.Id), appendedValue)

	// Update guild count
	k.SetGuildCount(ctx, count+1)
    k.GuildPermissionAdd(ctx, guild.Id, player.Id, types.GuildPermissionAll)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: guild.Id, ObjectType: types.ObjectType_guild})

	return guild
}

// SetGuild set a specific guild in the store
func (k Keeper) SetGuild(ctx sdk.Context, guild types.Guild) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildKey))
	b := k.cdc.MustMarshal(&guild)
	store.Set(GetGuildIDBytes(guild.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: guild.Id, ObjectType: types.ObjectType_guild})
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



func (k Keeper) GuildSetRegisterRequest(ctx sdk.Context, guild types.Guild, player types.Player) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildRegistrationKey))

    	bz := make([]byte, 8)
    	binary.BigEndian.PutUint64(bz, guild.Id)

    	store.Set(GetPlayerIDBytes(player.Id), bz)
}

func (k Keeper) GuildApproveRegisterRequest(ctx sdk.Context, guild types.Guild, player types.Player) {

    registrationGuild, registrationFound := k.GuildGetRegisterRequest(ctx, player)
    if ((registrationFound) && (registrationGuild.Id == guild.Id)) {
            // look up destination substation
            substation, substationFound := k.GetSubstation(ctx, guild.EntrySubstationId, true)

            // If the player is already connected to a substation then leave them
            // Maybe add an option to force migration later
            if (player.SubstationId == 0) {
                if (substationFound) {
                    // Check if the substation has room
                    if substation.HasPlayerCapacity() {
                        // Connect Player to Substation
                        k.SubstationConnectPlayer(ctx, substation, player)
                    }
                }
            }

            // Add player to the guild
            player.SetGuild(guild.Id)
            k.SetPlayer(ctx, player)

            store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildRegistrationKey))
            store.Delete(GetPlayerIDBytes(player.Id))
    }

}

func (k Keeper) GuildDenyRegisterRequest(ctx sdk.Context, guild types.Guild, player types.Player) {
    registrationGuild, registrationFound := k.GuildGetRegisterRequest(ctx, player)
    if ((registrationFound) && (registrationGuild.Id == guild.Id)) {
            store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildRegistrationKey))
            store.Delete(GetPlayerIDBytes(player.Id))
    }
}

func (k Keeper) GuildGetRegisterRequest(ctx sdk.Context, player types.Player) (guild types.Guild, found bool) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildRegistrationKey))

    	bz := store.Get(GetPlayerIDBytes(player.Id))

    	// Substation Capacity Not in Memory: no element
    	if bz == nil {
    		return types.Guild{}, false
    	}

    	guild, found = k.GetGuild(ctx, binary.BigEndian.Uint64(bz))

    	return guild, found

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

	load := types.GuildPermission(binary.BigEndian.Uint64(bz))

	return load
}

func (k Keeper) GuildSetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte, permissions types.GuildPermission) (types.GuildPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildPermissionKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(permissions))

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
