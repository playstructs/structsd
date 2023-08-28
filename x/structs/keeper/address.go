package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)



func (k Keeper) GetPlayerIdFromAddress(ctx sdk.Context, address string) (uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressPlayerKey))

	bz := store.Get(types.KeyPrefix(address))

	// Address Not in Memory: no element
	if bz == nil  {
		return 0
	}

	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetPlayerIdForAddress(ctx sdk.Context, address string, playerId uint64)  {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressPlayerKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, playerId)

	store.Set(types.KeyPrefix(address), bz)

}


func (k Keeper) AddressSetRegisterRequest(ctx sdk.Context, player types.Player, address string) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressRegistrationKey))

    	bz := make([]byte, 8)
    	binary.BigEndian.PutUint64(bz, player.Id)

    	store.Set(types.KeyPrefix(address), bz)

}

func (k Keeper) AddressApproveRegisterRequest(ctx sdk.Context, player types.Player, address string, permissions types.AddressPermission) {

    registrationPlayer, registrationFound := k.AddressGetRegisterRequest(ctx, address)
    if ((registrationFound) && (registrationPlayer.Id == player.Id)) {
            k.AddressPermissionAdd(ctx, address, permissions)

            store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressRegistrationKey))
            store.Delete(types.KeyPrefix(address))
    }

}

func (k Keeper) AddressDenyRegisterRequest(ctx sdk.Context, player types.Player, address string) {
    registrationPlayer, registrationFound := k.AddressGetRegisterRequest(ctx, address)
    if ((registrationFound) && (registrationPlayer.Id == player.Id)) {
            store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressRegistrationKey))
            store.Delete(types.KeyPrefix(address))
    }
}

func (k Keeper) AddressGetRegisterRequest(ctx sdk.Context, address string) (player types.Player, found bool) {
    	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressRegistrationKey))

    	bz := store.Get(types.KeyPrefix(address))

    	// Substation Capacity Not in Memory: no element
    	if bz == nil {
    		return types.Player{}, false
    	}

    	player, found = k.GetPlayer(ctx, binary.BigEndian.Uint64(bz))

    	return player, found

}


func (k Keeper) AddressGetPlayerPermissions(ctx sdk.Context, address string) (types.AddressPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressPermissionKey))

	bz := store.Get(types.KeyPrefix(address))

	// Substation Capacity Not in Memory: no element
	if bz == nil {
		return types.AddressPermissionless
	}

	load := types.AddressPermission(binary.BigEndian.Uint64(bz))

	return load
}

func (k Keeper) AddressSetPlayerPermissions(ctx sdk.Context, address string, permissions types.AddressPermission) (types.AddressPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressPermissionKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(permissions))

	store.Set(types.KeyPrefix(address), bz)

	return permissions
}

func (k Keeper) AddressPermissionClearAll(ctx sdk.Context, address string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressPermissionKey))
	store.Delete(types.KeyPrefix(address))
}

func (k Keeper) AddressPermissionAdd(ctx sdk.Context, address string, flag types.AddressPermission) types.AddressPermission {

    currentPermission := k.AddressGetPlayerPermissions(ctx, address)
    newPermissions := k.AddressSetPlayerPermissions(ctx, address, currentPermission | flag)
	return newPermissions
}

func (k Keeper) AddressPermissionRemove(ctx sdk.Context, address string, flag types.AddressPermission) types.AddressPermission {

    currentPermission := k.AddressGetPlayerPermissions(ctx, address)
    newPermissions := k.AddressSetPlayerPermissions(ctx, address, currentPermission &^ flag)
	return newPermissions
}

func (k Keeper) AddressPermissionHasAll(ctx sdk.Context, address string, flag types.AddressPermission) bool {
    currentPermission := k.AddressGetPlayerPermissions(ctx, address)

	return currentPermission&flag == flag
}

func (k Keeper) AddressPermissionHasOneOf(ctx sdk.Context, address string, flag types.AddressPermission) bool {
    currentPermission := k.AddressGetPlayerPermissions(ctx, address)

	return currentPermission&flag != 0
}
