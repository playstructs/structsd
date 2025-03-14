package keeper

import (
	"context"
	"encoding/binary"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
    authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"fmt"
)

// GetProviderCount get the total number of provider
func (k Keeper) GetProviderCount(ctx context.Context) uint64 {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.ProviderCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0 {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetProviderCount set the total number of provider
func (k Keeper) SetProviderCount(ctx context.Context, count uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.ProviderCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendProvider appends a provider in the store with a new id
func (k Keeper) AppendProvider(ctx context.Context, provider types.Provider) (types.Provider, error) {

	// Define the provider id
	count := k.GetProviderCount(ctx)

	// Set the ID of the appended value
	provider.Id = GetObjectID(types.ObjectType_provider, count)
	provider.Index = count

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderKey))
	appendedValue := k.cdc.MustMarshal(&provider)
	store.Set([]byte(provider.Id), appendedValue)

	k.SetProviderCount(ctx, count+1)

	ctxSDK := sdk.UnwrapSDKContext(ctx)

	// Set the Checkpoint to current block
    k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_checkpointBlock, provider.Id), uint64(ctxSDK.BlockHeight()))

    // Create the Collateral Pool
    fmt.Printf("Provider Collateral Pool: %s", types.ProviderCollateralPool + provider.Id)
    fmt.Printf("Provider Collateral Pool: %s", authtypes.NewModuleAddress(types.ProviderCollateralPool + provider.Id))

    providerCollateralAddress := authtypes.NewModuleAddress(types.ProviderCollateralPool + provider.Id)
    providerCollateralAccount := k.accountKeeper.NewAccountWithAddress(ctx, providerCollateralAddress)
    k.accountKeeper.SetAccount(ctx, providerCollateralAccount)

    // Create the Earnings Pool
    fmt.Printf("Provider Earnings Pool: %s", types.ProviderEarningsPool + provider.Id)
    fmt.Printf("Provider Earnings Pool: %s", authtypes.NewModuleAddress(types.ProviderEarningsPool + provider.Id))

    providerEarningsAddress := authtypes.NewModuleAddress(types.ProviderEarningsPool + provider.Id)
    providerEarningsAccount := k.accountKeeper.NewAccountWithAddress(ctx, providerEarningsAddress)
    k.accountKeeper.SetAccount(ctx, providerEarningsAccount)

	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventProvider{Provider: &provider})

	return provider, nil
}

func (k Keeper) SetProvider(ctx context.Context, provider types.Provider) (types.Provider, error) {

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderKey))
	b := k.cdc.MustMarshal(&provider)
	store.Set([]byte(provider.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventProvider{Provider: &provider})

	return provider, nil

}

// ImportProvider set a specific provider in the store
// Assumes Grid updates happen elsewhere
func (k Keeper) ImportProvider(ctx context.Context, provider types.Provider) {

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderKey))
	b := k.cdc.MustMarshal(&provider)
	store.Set([]byte(provider.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventProvider{Provider: &provider})
}

// RemoveProvider removes a provider from the store
func (k Keeper) RemoveProvider(ctx context.Context, providerId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderKey))
	store.Delete([]byte(providerId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ObjectId: providerId})
}

// DestroyProvider updates grid attributes before calling RemoveProvider
func (k Keeper) DestroyProvider(ctx context.Context, providerId string) (destroyed bool) {
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



func (k Keeper) ProviderGrantGuild(ctx context.Context, providerId string, guildId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.ProviderGuildAccessKey + providerId + "/" + guildId)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, 1)
	store.Set(byteKey, bz)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
     _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventProviderGrantGuild{&types.EventProviderGrantGuildDetail{ProviderId: providerId, GuildId: guildId}})

}

func (k Keeper) ProviderRevokeGuild(ctx context.Context, providerId string, guildId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderKey))
	store.Delete(types.KeyPrefix(types.ProviderGuildAccessKey + providerId + "/" + guildId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventProviderRevokeGuild{&types.EventProviderRevokeGuildDetail{ProviderId: providerId, GuildId: guildId}})
}

func (k Keeper) ProviderGuildAccessAllowed(ctx context.Context, providerId string, guildId string) (bool) {
    	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
    	byteKey := types.KeyPrefix(types.ProviderGuildAccessKey + providerId + "/" + guildId)
    	bz := store.Get(byteKey)

    	// doesn't exist: no element
    	return bz != nil && binary.BigEndian.Uint64(bz) != 0
}