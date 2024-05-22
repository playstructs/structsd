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

	// Add the Account keeper record
	playerAccAddress, _ := sdk.AccAddressFromBech32(address)
	playerAuthAccount := k.accountKeeper.GetAccount(ctx, playerAccAddress)
    if playerAuthAccount == nil {
        playerAuthAccount = k.accountKeeper.NewAccountWithAddress(ctx, playerAccAddress)
        k.accountKeeper.SetAccount(ctx, playerAuthAccount)
    }

    ctxSDK := sdk.UnwrapSDKContext(ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAddressAssociation{&types.AddressAssociation{Address: address, PlayerIndex: playerIndex, RegistrationStatus: types.RegistrationStatus_approved}})
}

func (k Keeper) RevokePlayerIndexForAddress(ctx context.Context, address string, playerIndex uint64)  {
    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressPlayerKey))
	store.Delete(types.KeyPrefix(address))

    ctxSDK := sdk.UnwrapSDKContext(ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAddressAssociation{&types.AddressAssociation{Address: address, PlayerIndex: playerIndex, RegistrationStatus: types.RegistrationStatus_revoked}})
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


func (k Keeper) AddressEmitActivity(ctx context.Context, address string) {
    ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAddressActivity{&types.AddressActivity{Address: address, BlockHeight: ctxSDK.BlockHeight(), BlockTime: ctxSDK.HeaderInfo().Time }})
}