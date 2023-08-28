package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strconv"
)



// GetPlayerPermissionIDBytes returns the byte representation of the Player and player id pair
func GetPlayerPermissionIDBytes(substationId uint64, playerId uint64) []byte {
	substationIdString  := strconv.FormatUint(substationId, 10)
	playerIdString := strconv.FormatUint(playerId, 10)

	return []byte(substationIdString + "-" + playerIdString)
}


func (k Keeper) PlayerGetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte) (types.PlayerPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerPermissionKey))

	bz := store.Get(permissionRecord)

	// Player Capacity Not in Memory: no element
	if bz == nil {
		return types.PlayerPermissionless
	}

	load := types.PlayerPermission(binary.BigEndian.Uint64(bz))

	return load
}

func (k Keeper) PlayerSetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte, permissions types.PlayerPermission) (types.PlayerPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerPermissionKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(permissions))

	store.Set(permissionRecord, bz)

	return permissions
}

func (k Keeper) PlayerPermissionClearAll(ctx sdk.Context, PlayerId uint64, playerId uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerPermissionKey))
	store.Delete(GetPlayerPermissionIDBytes(PlayerId, playerId))
}

func (k Keeper) PlayerPermissionAdd(ctx sdk.Context, targetPlayerId uint64, playerId uint64, flag types.PlayerPermission) types.PlayerPermission {
    permissionRecord := GetPlayerPermissionIDBytes(targetPlayerId, playerId)

    currentPermission := k.PlayerGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.PlayerSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission | flag)
	return newPermissions
}

func (k Keeper) PlayerPermissionRemove(ctx sdk.Context, targetPlayerId uint64, playerId uint64, flag types.PlayerPermission) types.PlayerPermission {
    permissionRecord := GetPlayerPermissionIDBytes(targetPlayerId, playerId)

    currentPermission := k.PlayerGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.PlayerSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission &^ flag)
	return newPermissions
}

func (k Keeper) PlayerPermissionHasAll(ctx sdk.Context, targetPlayerId uint64, playerId uint64, flag types.PlayerPermission) bool {
    permissionRecord := GetPlayerPermissionIDBytes(targetPlayerId, playerId)

    currentPermission := k.PlayerGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag == flag
}

func (k Keeper) PlayerPermissionHasOneOf(ctx sdk.Context, targetPlayerId uint64, playerId uint64, flag types.PlayerPermission) bool {
    permissionRecord := GetPlayerPermissionIDBytes(targetPlayerId, playerId)

    currentPermission := k.PlayerGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag != 0
}


