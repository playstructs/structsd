package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	"strconv"
	"strings"
)

// GetSquadCount get the total number of squad
func (k Keeper) GetSquadCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.SquadCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0  {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetSquadCount set the total number of squad
func (k Keeper) SetSquadCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.SquadCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendSquad appends a squad in the store with a new id and update the count
func (k Keeper) AppendSquad(
	ctx sdk.Context,
	guildId uint64,
    entrySubstationId uint64,
	squadJoinType uint64,
	leader uint64,
	creator string,
) (squad types.Squad) {
    squad = types.CreateEmptySquad()

	// Create the squad
	count := k.GetSquadCount(ctx)

	// Set the ID of the appended value
	squad.Id = count
	squad.SetGuildId(guildId)
	squad.SetCreator(creator)
	squad.SetLeader(leader)
	squad.SetSquadJoinType(squadJoinType)
	squad.SetEntrySubstationId(entrySubstationId)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadKey))
	appendedValue := k.cdc.MustMarshal(&squad)
	store.Set(GetSquadIDBytes(squad.Id), appendedValue)

	// Update squad count
	k.SetSquadCount(ctx, count+1)
    k.SquadPermissionAdd(ctx, squad.Id, leader, types.SquadPermissionAll)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventSquad{Squad: &squad})

	return squad
}

// SetSquad set a specific squad in the store
func (k Keeper) SetSquad(ctx sdk.Context, squad types.Squad) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadKey))
	b := k.cdc.MustMarshal(&squad)
	store.Set(GetSquadIDBytes(squad.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventSquad{Squad: &squad})
}

// GetSquad returns a squad from its id
func (k Keeper) GetSquad(ctx sdk.Context, id uint64) (val types.Squad, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadKey))
	b := store.Get(GetSquadIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveSquad removes a squad from the store
func (k Keeper) RemoveSquad(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadKey))
	store.Delete(GetSquadIDBytes(id))

	_ = ctx.EventManager().EmitTypedEvent(&types.EventSquadDelete{SquadId: id})
}

// GetAllSquad returns all squad
func (k Keeper) GetAllSquad(ctx sdk.Context) (list []types.Squad) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Squad
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetSquadIDBytes returns the byte representation of the ID
func GetSquadIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetSquadIDFromBytes returns ID in uint64 format from a byte array
func GetSquadIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}



func (k Keeper) SquadSetRegisterRequest(ctx sdk.Context, squad types.Squad, player types.Player) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadRegistrationKey))

    	bz := make([]byte, 8)
    	binary.BigEndian.PutUint64(bz, squad.Id)

    	store.Set(GetPlayerIDBytes(player.Id), bz)
}

func (k Keeper) SquadApproveRegisterRequest(ctx sdk.Context, squad types.Squad, player types.Player) {

    registrationSquad, registrationFound := k.SquadGetRegisterRequest(ctx, player)
    if ((registrationFound) && (registrationSquad.Id == squad.Id)) {
            // look up destination substation
            substation, substationFound := k.GetSubstation(ctx, squad.EntrySubstationId, true)

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

            // Add player to the squad
            player.SetSquad(squad.Id)
            k.SetPlayer(ctx, player)

            store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadRegistrationKey))
            store.Delete(GetPlayerIDBytes(player.Id))
    }

}

func (k Keeper) SquadDenyRegisterRequest(ctx sdk.Context, squad types.Squad, player types.Player) {
    registrationSquad, registrationFound := k.SquadGetRegisterRequest(ctx, player)
    if ((registrationFound) && (registrationSquad.Id == squad.Id)) {
            store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadRegistrationKey))
            store.Delete(GetPlayerIDBytes(player.Id))
    }
}

func (k Keeper) SquadGetRegisterRequest(ctx sdk.Context, player types.Player) (squad types.Squad, found bool) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadRegistrationKey))

    	bz := store.Get(GetPlayerIDBytes(player.Id))

    	// Substation Capacity Not in Memory: no element
    	if bz == nil {
    		return types.Squad{}, false
    	}

    	squad, found = k.GetSquad(ctx, binary.BigEndian.Uint64(bz))

    	return squad, found

}



// GetSquadPermissionIDBytes returns the byte representation of the squad and player id pair
func GetSquadPermissionIDBytes(squadId uint64, playerId uint64) []byte {
	squadIdString  := strconv.FormatUint(squadId, 10)
	playerIdString := strconv.FormatUint(playerId, 10)

	return []byte(squadIdString + "-" + playerIdString)
}


func (k Keeper) SquadGetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte) (types.SquadPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadPermissionKey))

	bz := store.Get(permissionRecord)

	// Substation Capacity Not in Memory: no element
	if bz == nil {
		return types.SquadPermissionless
	}

	load := types.SquadPermission(binary.BigEndian.Uint64(bz))

	return load
}

func (k Keeper) SquadSetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte, permissions types.SquadPermission) (types.SquadPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadPermissionKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(permissions))

	store.Set(permissionRecord, bz)

    keys := strings.Split(string(permissionRecord), "-")
    _ = ctx.EventManager().EmitTypedEvent(&types.EventSquadPermission{Body: &types.EventPermissionBodyKeyPair{ObjectId: keys[0], PlayerId: keys[1], Value: uint64(permissions)}})

	return permissions
}

func (k Keeper) SquadPermissionClearAll(ctx sdk.Context, squadId uint64, playerId uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadPermissionKey))
	permissionId := GetSquadPermissionIDBytes(squadId, playerId)
	store.Delete(permissionId)

    keys := strings.Split(string(permissionId), "-")
    _ = ctx.EventManager().EmitTypedEvent(&types.EventSquadPermission{Body: &types.EventPermissionBodyKeyPair{ObjectId: keys[0], PlayerId: keys[1], Value: uint64(0)}})

}

func (k Keeper) SquadPermissionAdd(ctx sdk.Context, squadId uint64, playerId uint64, flag types.SquadPermission) types.SquadPermission {
    permissionRecord := GetSquadPermissionIDBytes(squadId, playerId)

    currentPermission := k.SquadGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.SquadSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission | flag)
	return newPermissions
}

func (k Keeper) SquadPermissionRemove(ctx sdk.Context, squadId uint64, playerId uint64, flag types.SquadPermission) types.SquadPermission {
    permissionRecord := GetSquadPermissionIDBytes(squadId, playerId)

    currentPermission := k.SquadGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.SquadSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission &^ flag)
	return newPermissions
}

func (k Keeper) SquadPermissionHasAll(ctx sdk.Context, squadId uint64, playerId uint64, flag types.SquadPermission) bool {
    permissionRecord := GetSquadPermissionIDBytes(squadId, playerId)

    currentPermission := k.SquadGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag == flag
}

func (k Keeper) SquadPermissionHasOneOf(ctx sdk.Context, squadId uint64, playerId uint64, flag types.SquadPermission) bool {
    permissionRecord := GetSquadPermissionIDBytes(squadId, playerId)

    currentPermission := k.SquadGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag != 0
}
