package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//"strconv"
	//"strings"
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
	substationId string,
	reactor types.Reactor,
	player types.Player,
) (guild types.Guild) {
    guild = types.CreateEmptyGuild()

	// Create the guild
	count := k.GetGuildCount(ctx)

	// Set the ID of the appended value
	guild.Id                = GetObjectID(types.ObjectType_guild, count)
	guild.Index             = count
	guild.Endpoint          = endpoint
	guild.Creator           = player.Creator
	guild.Owner             = player.Id
	guild.PrimaryReactorId  = reactor.Id
	guild.EntrySubstationId = substationId

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildKey))
	appendedValue := k.cdc.MustMarshal(&guild)
	store.Set([]byte(guild.Id), appendedValue)

	// Update guild count
	k.SetGuildCount(ctx, count+1)

	permissionId := GetObjectPermissionIDBytes(guild.Id, player.Id)
    k.PermissionAdd(ctx, permissionId, types.Permission(types.GuildPermissionAll))

	_ = ctx.EventManager().EmitTypedEvent(&types.EventGuild{Guild: &guild})

	return guild
}

// SetGuild set a specific guild in the store
func (k Keeper) SetGuild(ctx sdk.Context, guild types.Guild) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildKey))
	b := k.cdc.MustMarshal(&guild)
	store.Set([]byte(guild.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventGuild{Guild: &guild})
}

// GetGuild returns a guild from its id
func (k Keeper) GetGuild(ctx sdk.Context, guildId string) (val types.Guild, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildKey))
	b := store.Get([]byte(guildId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveGuild removes a guild from the store
func (k Keeper) RemoveGuild(ctx sdk.Context, guildId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildKey))
	store.Delete([]byte(guildId))

	_ = ctx.EventManager().EmitTypedEvent(&types.EventDelete{ObjectId: guildId})
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


func (k Keeper) GuildSetRegisterRequest(ctx sdk.Context, guild types.Guild, player types.Player) {
    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildRegistrationKey))

    bz := make([]byte, 8)
    binary.BigEndian.PutUint64(bz, guild.Index)

    store.Set([]byte(player.Id), bz)

    _ = ctx.EventManager().EmitTypedEvent(&types.EventGuildAssociation{GuildId: guild.Id, PlayerId: player.Id, RegistrationStatus: types.RegistrationStatus_proposed})
}

func (k Keeper) GuildApproveRegisterRequest(ctx sdk.Context, guild types.Guild, player types.Player) {

    registrationGuild, registrationFound := k.GuildGetRegisterRequest(ctx, player)
    if ((registrationFound) && (registrationGuild.Id == guild.Id)) {
            // look up destination substation
            substation, substationFound := k.GetSubstation(ctx, guild.EntrySubstationId, true)

            // If the player is already connected to a substation then leave them
            // Maybe add an option to force migration later
            if (player.SubstationId == "") {
                if (substationFound) {
                    // Check if the substation has room
                    //if substation.HasPlayerCapacity() {
                        // Connect Player to Substation
                        k.SubstationConnectPlayer(ctx, substation, player)
                    //}
                }
            }

            // Add player to the guild
            player.GuildId = guild.Id
            k.SetPlayer(ctx, player)

            store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildRegistrationKey))
            store.Delete([]byte(player.Id))

            _ = ctx.EventManager().EmitTypedEvent(&types.EventGuildAssociation{GuildId: guild.Id, PlayerId: player.Id, RegistrationStatus: types.RegistrationStatus_approved})

    }

}

func (k Keeper) GuildDenyRegisterRequest(ctx sdk.Context, guild types.Guild, player types.Player) {
    registrationGuild, registrationFound := k.GuildGetRegisterRequest(ctx, player)
    if ((registrationFound) && (registrationGuild.Id == guild.Id)) {
        store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildRegistrationKey))
        store.Delete([]byte(player.Id))

        _ = ctx.EventManager().EmitTypedEvent(&types.EventGuildAssociation{GuildId: guild.Id, PlayerId: player.Id, RegistrationStatus: types.RegistrationStatus_denied})
    }
}

func (k Keeper) GuildGetRegisterRequest(ctx sdk.Context, player types.Player) (guild types.Guild, found bool) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GuildRegistrationKey))

    	bz := store.Get([]byte(player.Id))

    	// Substation Capacity Not in Memory: no element
    	if bz == nil {
    		return types.Guild{}, false
    	}

    	guild, found = k.GetGuild(ctx, GetObjectID(types.ObjectType_guild, binary.BigEndian.Uint64(bz)))

    	return guild, found
}


