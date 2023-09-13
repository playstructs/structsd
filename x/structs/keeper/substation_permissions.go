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



// GetSubstationPermissionIDBytes returns the byte representation of the Substation and player id pair
func GetSubstationPermissionIDBytes(substationId uint64, playerId uint64) []byte {
	substationIdString  := strconv.FormatUint(substationId, 10)
	playerIdString := strconv.FormatUint(playerId, 10)

	return []byte(substationIdString + "-" + playerIdString)
}


func (k Keeper) SubstationGetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte) (types.SubstationPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationPermissionKey))

	bz := store.Get(permissionRecord)

	// Substation Capacity Not in Memory: no element
	if bz == nil {
		return types.SubstationPermissionless
	}

	load := types.SubstationPermission(binary.BigEndian.Uint64(bz))

	return load
}

func (k Keeper) SubstationSetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte, permissions types.SubstationPermission) (types.SubstationPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationPermissionKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(permissions))

	store.Set(permissionRecord, bz)

    keys := strings.Split(string(permissionRecord), "-")
    _ = ctx.EventManager().EmitTypedEvent(&types.EventSubstationPermission{Body: &types.EventPermissionBodyKeyPair{ObjectId: keys[0], PlayerId: keys[1], Value: uint64(permissions)}})


	return permissions
}

func (k Keeper) SubstationPermissionClearAll(ctx sdk.Context, SubstationId uint64, playerId uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationPermissionKey))
	store.Delete(GetSubstationPermissionIDBytes(SubstationId, playerId))
}

func (k Keeper) SubstationPermissionAdd(ctx sdk.Context, substationId uint64, playerId uint64, flag types.SubstationPermission) types.SubstationPermission {
    permissionRecord := GetSubstationPermissionIDBytes(substationId, playerId)

    currentPermission := k.SubstationGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.SubstationSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission | flag)
	return newPermissions
}

func (k Keeper) SubstationPermissionRemove(ctx sdk.Context, substationId uint64, playerId uint64, flag types.SubstationPermission) types.SubstationPermission {
    permissionRecord := GetSubstationPermissionIDBytes(substationId, playerId)

    currentPermission := k.SubstationGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.SubstationSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission &^ flag)
	return newPermissions
}

func (k Keeper) SubstationPermissionHasAll(ctx sdk.Context, substationId uint64, playerId uint64, flag types.SubstationPermission) bool {
    permissionRecord := GetSubstationPermissionIDBytes(substationId, playerId)

    currentPermission := k.SubstationGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag == flag
}

func (k Keeper) SubstationPermissionHasOneOf(ctx sdk.Context, substationId uint64, playerId uint64, flag types.SubstationPermission) bool {
    permissionRecord := GetSubstationPermissionIDBytes(substationId, playerId)

    currentPermission := k.SubstationGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag != 0
}


