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


	_ = ctx.EventManager().EmitTypedEvent(&types.EventAddressAssociation{Address: &types.EventAddressBody{Address: address, PlayerId: playerId}})


}


func (k Keeper) AddressSetRegisterRequest(ctx sdk.Context, player types.Player, address string) {
    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AddressRegistrationKey))

    bz := make([]byte, 8)
    binary.BigEndian.PutUint64(bz, player.Id)

    store.Set(types.KeyPrefix(address), bz)

    _ = ctx.EventManager().EmitTypedEvent(&types.EventAddressRegistrationRequest{Address: &types.EventAddressBody{Address: address, PlayerId: player.Id}})
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

