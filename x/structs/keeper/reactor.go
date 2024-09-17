package keeper

import (
	"encoding/binary"
	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//sdkerrors "cosmossdk.io/errors"

	//"fmt"
)

// GetReactorCount get the total number of reactor
func (k Keeper) GetReactorCount(ctx context.Context) uint64 {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.ReactorCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0 {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetReactorCount set the total number of reactor
func (k Keeper) SetReactorCount(ctx context.Context, count uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.ReactorCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// GetReactorBytesFromValidator get the bytes based on validator address
func (k Keeper) GetReactorBytesFromValidator(ctx context.Context, validatorAddress []byte) (reactorBytes []byte, found bool) {

    if validatorAddress == nil {
        return reactorBytes, false
    }

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ReactorValidatorKey))

	reactorBytes = store.Get(validatorAddress)
	// Count doesn't exist: no element
	if reactorBytes == nil {
		return reactorBytes, false
	}

	return reactorBytes, true
}

// SetReactorValidatorBytes set the validator address index bytes
func (k Keeper) SetReactorValidatorBytes(ctx context.Context, reactorId string, validatorAddress []byte) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ReactorValidatorKey))

	store.Set(validatorAddress, []byte(reactorId))
}

// AppendReactor appends a reactor in the store with a new id and update the count
func (k Keeper) AppendReactor(
	ctx context.Context,
	reactor types.Reactor,
) types.Reactor {
	// Create the reactor
	count := k.GetReactorCount(ctx)

	// Set the ID of the appended value
	reactor.Id = GetObjectID(types.ObjectType_reactor, count)

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ReactorKey))
	appendedValue := k.cdc.MustMarshal(&reactor)
	store.Set([]byte(reactor.Id), appendedValue)

	// Add a record to the Validator index
	k.SetReactorValidatorBytes(ctx, reactor.Id, reactor.RawAddress)

	// Update reactor count
	k.SetReactorCount(ctx, count+1)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventReactor{Reactor: &reactor})

	return reactor
}

// SetReactor set a specific reactor in the store
func (k Keeper) SetReactor(ctx context.Context, reactor types.Reactor) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ReactorKey))
	b := k.cdc.MustMarshal(&reactor)
	store.Set([]byte(reactor.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventReactor{Reactor: &reactor})
}

// GetReactor returns a reactor from its id
func (k Keeper) GetReactor(ctx context.Context, reactorId string) (val types.Reactor, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ReactorKey))
	b := store.Get([]byte(reactorId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// GetReactor returns a reactor from its id
func (k Keeper) GetReactorByBytes(ctx context.Context, id []byte) (val types.Reactor, found bool) {
    if id == nil {
        return val, false
    }

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ReactorKey))
	b := store.Get(id)
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// RemoveReactor removes a reactor from the store
func (k Keeper) RemoveReactor(ctx context.Context, reactorId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ReactorKey))
	store.Delete([]byte(reactorId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ ObjectId: reactorId})
}

// GetAllReactor returns all reactor
func (k Keeper) GetAllReactor(ctx context.Context) (list []types.Reactor) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ReactorKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Reactor
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		list = append(list, val)
	}

	return
}


