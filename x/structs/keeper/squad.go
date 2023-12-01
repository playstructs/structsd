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
	creator string,
	guildId uint64,
	leader uint64,
	squadJoinType uint64,
    entrySubstationId uint64,
) (squad types.Squad) {
    squad = types.CreateEmptySquad()

	// Create the squad
	count := k.GetSquadCount(ctx)

	// Set the ID of the appended value
	squad.Id = count
	squad.SetCreator(creator)
	squad.SetGuildId(guildId)
	squad.SetLeader(leader)
	squad.SetSquadJoinType(squadJoinType)
	squad.SetEntrySubstationId(entrySubstationId)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadKey))
	appendedValue := k.cdc.MustMarshal(&squad)
	store.Set(GetSquadIDBytes(squad.Id), appendedValue)

	// Update squad count
	k.SetSquadCount(ctx, count+1)

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



func (k Keeper) SquadSetLeaderProposalRequest(ctx sdk.Context, squad types.Squad, player types.Player) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadLeaderProposalKey))

    	bz := make([]byte, 8)
    	binary.BigEndian.PutUint64(bz, player.Id)

    	store.Set(GetSquadIDBytes(squad.Id), bz)
}

func (k Keeper) SquadApproveLeaderProposalRequest(ctx sdk.Context, squad types.Squad, player types.Player) {

    // Add player to the squad
    player.SetSquad(squad.Id)
    k.SetPlayer(ctx, player)

    squad.SetLeader(player.Id)
    k.SetSquad(ctx, squad)

    k.SquadPermissionAdd(ctx, squad.Id, player.Id, types.SquadPermissionAll)

    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadLeaderProposalKey))
    store.Delete(GetSquadIDBytes(squad.Id))
}

func (k Keeper) SquadDenyLeaderProposalRequest(ctx sdk.Context, squad types.Squad, player types.Player) {
    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadLeaderProposalKey))
    store.Delete(GetSquadIDBytes(squad.Id))
}

func (k Keeper) SquadDeleteLeaderProposalRequest(ctx sdk.Context, squad types.Squad) {
    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadLeaderProposalKey))
    store.Delete(GetSquadIDBytes(squad.Id))
}

func (k Keeper) SquadGetLeaderProposalRequest(ctx sdk.Context, squad types.Squad) (player types.Player, found bool) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadLeaderProposalKey))

    	bz := store.Get(GetSquadIDBytes(squad.Id))

    	// Substation Capacity Not in Memory: no element
    	if bz == nil {
    		return types.Player{}, false
    	}

    	player, found = k.GetPlayer(ctx, binary.BigEndian.Uint64(bz))

    	return player, found

}

// GetSquadIDPlayerIDBytes returns the byte representation of the squad id and player id pair
func GetSquadIDPlayerIDBytes(squadId uint64, playerId uint64) []byte {
	squadIdString  := strconv.FormatUint(squadId, 10)
	playerIdString := strconv.FormatUint(playerId, 10)

	return []byte(squadIdString + "-" + playerIdString)
}


func (k Keeper) SquadSetInvite(ctx sdk.Context, squad types.Squad, player types.Player) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadInviteKey))

    	bz := make([]byte, 8)
    	binary.BigEndian.PutUint64(bz, types.SquadInviteStatus_Pending)

    	store.Set(GetSquadIDPlayerIDBytes(squad.Id, player.Id), bz)
}

func (k Keeper) SquadApproveInvite(ctx sdk.Context, squad types.Squad, player types.Player) {

    // Add player to the squad
    player.SetSquad(squad.Id)
    k.SetPlayer(ctx, player)

    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadInviteKey))
    store.Delete(GetSquadIDPlayerIDBytes(squad.Id, player.Id))
}

func (k Keeper) SquadDenyInvite(ctx sdk.Context, squad types.Squad, player types.Player) {
    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadInviteKey))
    store.Delete(GetSquadIDPlayerIDBytes(squad.Id, player.Id))
}

func (k Keeper) SquadDeleteInvite(ctx sdk.Context, squad types.Squad) {
    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadInviteKey))
    store.Delete(GetSquadIDPlayerIDBytes(squad.Id, player.Id))
}

func (k Keeper) SquadGetInvite(ctx sdk.Context, squad types.Squad, player types.Player) (squadInviteStatus uint64, bool) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadInviteKey))

    	bz := store.Get(GetSquadIDPlayerIDBytes(squad.Id, player.Id))

    	// Invitation not in the keeper: no element
    	if bz == nil {
    		return types.SquadInviteStatus_Invalid, false
    	}

        // should be returning SquadInviteStatus_Pending
    	return types.binary.BigEndian.Uint64(bz), true

}


func (k Keeper) SquadSetJoinRequest(ctx sdk.Context, squad types.Squad, player types.Player) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadJoinRequestKey))

    	bz := make([]byte, 8)
    	binary.BigEndian.PutUint64(bz, types.SquadInviteStatus_Pending)

    	store.Set(GetSquadIDPlayerIDBytes(squad.Id, player.Id), bz)
}

func (k Keeper) SquadApproveJoinRequest(ctx sdk.Context, squad types.Squad, player types.Player) {

    // Add player to the squad
    player.SetSquad(squad.Id)
    k.SetPlayer(ctx, player)

    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadJoinRequestKey))
    store.Delete(GetSquadIDPlayerIDBytes(squad.Id, player.Id))
}

func (k Keeper) SquadDenyInvite(ctx sdk.Context, squad types.Squad, player types.Player) {
    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadJoinRequestKey))
    store.Delete(GetSquadIDPlayerIDBytes(squad.Id, player.Id))
}

func (k Keeper) SquadDeleteInvite(ctx sdk.Context, squad types.Squad) {
    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadJoinRequestKey))
    store.Delete(GetSquadIDPlayerIDBytes(squad.Id, player.Id))
}

func (k Keeper) SquadGetInvite(ctx sdk.Context, squad types.Squad, player types.Player) (squadInviteStatus uint64, bool) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SquadJoinRequestKey))

    	bz := store.Get(GetSquadIDPlayerIDBytes(squad.Id, player.Id))

    	// Invitation not in the keeper: no element
    	if bz == nil {
    		return types.SquadInviteStatus_Invalid, false
    	}

        // should be returning SquadInviteStatus_Pending
    	return types.binary.BigEndian.Uint64(bz), true

}


/*
 * Join Requests
 * ID based on Player ID
 *
 * Requires a Squad and Player
 * A player can only have one join request
 *
 */

// SquadSetJoinRequest
// SquadApproveJoinRequest
// SquadDenyJoinRequest
// SquadDeleteJoinRequest
// SquadGetJoinRequest


/*
 * Join Invite
 *
 * ID Based on Squad and Player
 *    ex: <squadId>-<playerId> = 1
 *
 * Require a Squad and Player
 * A player can only have multiple invites
 */

// SquadSetJoinInvite
// SquadApproveJoinInvite
// SquadDenyJoinInvite
// SquadDeleteJoinInvite
// SquadGetJoinInvite


// GetSquadPermissionIDBytes returns the byte representation of the squad and player id pair
func GetSquadPlayerIDBytes(squadId uint64, playerId uint64) []byte {
	squadIdString  := strconv.FormatUint(squadId, 10)
	playerIdString := strconv.FormatUint(playerId, 10)

	return []byte(squadIdString + "-" + playerIdString)
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
