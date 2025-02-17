package keeper

import (
	//"encoding/binary"
    "context"

    "github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//sdkerrors "cosmossdk.io/errors"

)

// AppendProvider appends a provider in the store with a new id
// TODO
func (k Keeper) AppendProvider(
	ctx context.Context,
	provider types.Provider,
) (err error) {

	return  nil
}

func (k Keeper) SetProviderOnly(ctx context.Context, provider types.Provider) (types.Provider, error){

    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderKey))
    b := k.cdc.MustMarshal(&provider)
    store.Set([]byte(provider.Id), b)

	//ctxSDK := sdk.UnwrapSDKContext(ctx)
    //_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventProvider{Provider: &provider})

    return provider,  nil

}



// ImportProvider set a specific provider in the store
// Assumes Grid updates happen elsewhere
func (k Keeper) ImportProvider(ctx context.Context, provider types.Provider){
    //k.SetProviderSourceIndex(ctx, provider.SourceObjectId, provider.Id)
    //k.SetProviderDestinationIndex(ctx, provider.DestinationId, provider.Id)

    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderKey))
    b := k.cdc.MustMarshal(&provider)
    store.Set([]byte(provider.Id), b)

	//ctxSDK := sdk.UnwrapSDKContext(ctx)
    //_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventProvider{Provider: &provider})
}



// RemoveProvider removes a provider from the store
func (k Keeper) RemoveProvider(ctx context.Context, providerId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderKey))
	store.Delete([]byte(providerId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ObjectId: providerId})
}

// DestroyProvider updates grid attributes before calling RemoveProvider
func (k Keeper) DestroyProvider(ctx context.Context, providerId string) (destroyed bool){
    provider, providerFound := k.GetProvider(ctx, providerId)

    _ = provider
    if providerFound {
    	destroyed = true
    } else {
        destroyed = false
    }

    return
}


// GetProvider returns a provider from its id
func (k Keeper) GetProvider(ctx context.Context, providerId string) (val types.Provider, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderKey))
	b := store.Get([]byte(providerId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// GetAllProvider returns all provider
func (k Keeper) GetAllProvider(ctx context.Context) (list []types.Provider) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Provider
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		list = append(list, val)
	}

	return
}


