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



// GetPermissionIDBytes returns the byte representation of the Player and player id pair
// TODO FROM HERE, so all of it

func GetPermissionIDBytes(objectId []byte, playerId uint64) []byte {
	substationIdString  := strconv.FormatUint(substationId, 10)
	playerIdString := strconv.FormatUint(playerId, 10)

	return []byte(substationIdString + "@" + playerIdString)
}


func (k Keeper) GetPermissionsByBytes(ctx sdk.Context, permissionRecord []byte) (types.Permission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PermissionKey))

	bz := store.Get(permissionRecord)

	// Player Capacity Not in Memory: no element
	if bz == nil {
		return types.Permissionless
	}

	load := types.Permission(binary.BigEndian.Uint64(bz))

	return load
}

func (k Keeper) SetPermissionsByBytes(ctx sdk.Context, permissionRecord []byte, permissions types.Permission) (types.Permission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PermissionKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(permissions))

	store.Set(permissionRecord, bz)

    keys := strings.Split(string(permissionRecord), "@")
    _ = ctx.EventManager().EmitTypedEvent(&types.EventPermission{Body: &types.EventPermissionBodyKeyPair{ObjectId: keys[0], PlayerId: keys[1], Value: uint64(permissions)}})

	return permissions
}

func (k Keeper) PermissionClearAll(ctx sdk.Context, PlayerId uint64, playerId uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerPermissionKey))

	permissionId := GetPlayerPermissionIDBytes(PlayerId, playerId)
	store.Delete(permissionId)

    keys := strings.Split(string(permissionId), "@")
    _ = ctx.EventManager().EmitTypedEvent(&types.EventPlayerPermission{Body: &types.EventPermissionBodyKeyPair{ObjectId: keys[0], PlayerId: keys[1], Value: uint64(0)}})

}

func (k Keeper) PermissionAdd(ctx sdk.Context, targetPlayerId uint64, playerId uint64, flag types.PlayerPermission) types.PlayerPermission {
    permissionRecord := GetPlayerPermissionIDBytes(targetPlayerId, playerId)

    currentPermission := k.PlayerGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.PlayerSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission | flag)
	return newPermissions
}

func (k Keeper) PermissionRemove(ctx sdk.Context, targetPlayerId uint64, playerId uint64, flag types.PlayerPermission) types.PlayerPermission {
    permissionRecord := GetPlayerPermissionIDBytes(targetPlayerId, playerId)

    currentPermission := k.PlayerGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.PlayerSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission &^ flag)
	return newPermissions
}

func (k Keeper) PermissionHasAll(ctx sdk.Context, targetPlayerId uint64, playerId uint64, flag types.Permission) bool {
    permissionRecord := GetPermissionIDBytes(targetPlayerId, playerId)

    currentPermission := k.GetPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag == flag
}

func (k Keeper) PermissionHasOneOf(ctx sdk.Context, targetPlayerId uint64, playerId uint64, flag types.Permission) bool {
    permissionRecord := GetPermissionIDBytes(targetPlayerId, playerId)

    currentPermission := k.GetPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag != 0
}


