package keeper

import (
	"encoding/binary"
	"context"

	"cosmossdk.io/store/prefix"
    "github.com/cosmos/cosmos-sdk/runtime"

	sdk "github.com/cosmos/cosmos-sdk/types"
	storetypes "cosmossdk.io/store/types"

	"structs/x/structs/types"
)


func (k Keeper) GetPlayerIndexFromAddress(ctx context.Context, address string) (uint64) {
    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressPlayerKey))

	bz := store.Get(types.KeyPrefix(address))

	// Address Not in Memory: no element
	if bz == nil  {
		return 0
	}

	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetPlayerIndexForAddress(ctx context.Context, address string, playerIndex uint64)  {
    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressPlayerKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, playerIndex)

	store.Set(types.KeyPrefix(address), bz)

    ctxSDK := sdk.UnwrapSDKContext(ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAddressAssociation{&types.AddressAssociation{Address: address, PlayerIndex: playerIndex, RegistrationStatus: types.RegistrationStatus_approved}})
}

// GetAllAddressExport returns all player addresses
func (k Keeper) GetAllAddressExport(ctx context.Context) (list []*types.AddressRecord) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressPlayerKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		list = append(list, &types.AddressRecord{Address: string(iterator.Key()), PlayerIndex: binary.BigEndian.Uint64(iterator.Value())})
	}

	return
}

func (k Keeper) AddressSetRegisterRequest(ctx context.Context, player types.Player, address string) {
    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressRegistrationKey))

    bz := make([]byte, 8)
    binary.BigEndian.PutUint64(bz, player.Index)

    store.Set(types.KeyPrefix(address), bz)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAddressAssociation{&types.AddressAssociation{Address: address, PlayerIndex: player.Index, RegistrationStatus: types.RegistrationStatus_proposed}})
}

func (k Keeper) AddressApproveRegisterRequest(ctx context.Context, player types.Player, address string, permissions types.Permission) {

    registrationPlayer, registrationFound := k.AddressGetRegisterRequest(ctx, address)
    if ((registrationFound) && (registrationPlayer.Index == player.Index)) {

            addressPermissionId := GetAddressPermissionIDBytes(address)
            k.PermissionAdd(ctx, addressPermissionId, permissions)

            store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressRegistrationKey))

            store.Delete(types.KeyPrefix(address))
    }

 	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAddressAssociation{&types.AddressAssociation{Address: address, PlayerIndex: player.Index, RegistrationStatus: types.RegistrationStatus_approved}})
}

func (k Keeper) AddressDenyRegisterRequest(ctx context.Context, player types.Player, address string) {
    registrationPlayer, registrationFound := k.AddressGetRegisterRequest(ctx, address)
    if ((registrationFound) && (registrationPlayer.Id == player.Id)) {
            store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressRegistrationKey))

            store.Delete(types.KeyPrefix(address))
    }

 	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAddressAssociation{&types.AddressAssociation{Address: address, PlayerIndex: player.Index, RegistrationStatus: types.RegistrationStatus_denied}})

}

func (k Keeper) AddressGetRegisterRequest(ctx context.Context, address string) (player types.Player, found bool) {
        store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressRegistrationKey))

    	bz := store.Get(types.KeyPrefix(address))

    	// Substation Capacity Not in Memory: no element
    	if bz == nil {
    		return types.Player{}, false
    	}

    	player, found = k.GetPlayerFromIndex(ctx, binary.BigEndian.Uint64(bz), false)

    	return player, found

}

