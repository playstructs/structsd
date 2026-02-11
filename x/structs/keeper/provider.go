package keeper

import (
	"context"
	"encoding/binary"
	"strings"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
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

func (k Keeper) ProviderGuildAccessAllowed(ctx context.Context, providerId string, guildId string) bool {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.ProviderGuildAccessKey + providerId + "/" + guildId)
	bz := store.Get(byteKey)

	// doesn't exist: no element
	return bz != nil && binary.BigEndian.Uint64(bz) != 0
}

func (k Keeper) GetAllProviderGuildAccessExport(ctx context.Context) (list []*types.ProviderGuildAccessRecord) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ProviderGuildAccessKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := string(iterator.Key()) // "providerId/guildId"
		parts := strings.SplitN(key, "/", 2)
		if len(parts) != 2 {
			continue
		}
		list = append(list, &types.ProviderGuildAccessRecord{
			ProviderId: parts[0],
			GuildId:    parts[1],
		})
	}
	return
}