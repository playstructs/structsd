package keeper

import (
	"encoding/binary"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	//"fmt"
)

// GetReactorCount get the total number of reactor
func (k Keeper) GetReactorCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
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
func (k Keeper) SetReactorCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.ReactorCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// GetReactorBytesFromValidator get the bytes based on validator address
func (k Keeper) GetReactorBytesFromValidator(ctx sdk.Context, validatorAddress []byte) (reactorBytes []byte, found bool) {

    if validatorAddress == nil {
        return reactorBytes, false
    }

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorValidatorKey))

	reactorBytes = store.Get(validatorAddress)
	// Count doesn't exist: no element
	if reactorBytes == nil {
		return reactorBytes, false
	}

	return reactorBytes, true
}

// SetReactorValidatorBytes set the validator address index bytes
func (k Keeper) SetReactorValidatorBytes(ctx sdk.Context, reactorId string, validatorAddress []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorValidatorKey))

	store.Set(validatorAddress, []byte(reactorId))
}

// AppendReactor appends a reactor in the store with a new id and update the count
func (k Keeper) AppendReactor(
	ctx sdk.Context,
	reactor types.Reactor,
) types.Reactor {
	// Create the reactor
	count := k.GetReactorCount(ctx)

	// Set the ID of the appended value
	reactor.Id = GetObjectID(types.ObjectType_reactor, count)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	appendedValue := k.cdc.MustMarshal(&reactor)
	store.Set([]byte(reactor.Id), appendedValue)

	// Add a record to the Validator index
	k.SetReactorValidatorBytes(ctx, reactor.Id, reactor.RawAddress)

	// Update reactor count
	k.SetReactorCount(ctx, count+1)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventReactor{Reactor: &reactor})


	return reactor
}

// SetReactor set a specific reactor in the store
func (k Keeper) SetReactor(ctx sdk.Context, reactor types.Reactor) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	b := k.cdc.MustMarshal(&reactor)
	store.Set([]byte(reactor.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventReactor{Reactor: &reactor})
}

// GetReactor returns a reactor from its id
func (k Keeper) GetReactor(ctx sdk.Context, reactorId string, full bool) (val types.Reactor, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	b := store.Get([]byte(reactorId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	if full {
        val.Load        = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, val.Id))
        val.Capacity    = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, val.Id))
        val.Fuel        = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, val.Id))
	}

	return val, true
}

// GetReactor returns a reactor from its id
func (k Keeper) GetReactorByBytes(ctx sdk.Context, id []byte, full bool) (val types.Reactor, found bool) {
    if id == nil {
        return val, false
    }

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	b := store.Get(id)
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	if full {
        val.Load        = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, val.Id))
        val.Capacity    = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, val.Id))
        val.Fuel        = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, val.Id))
	}

	return val, true
}

// RemoveReactor removes a reactor from the store
func (k Keeper) RemoveReactor(ctx sdk.Context, reactorId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	store.Delete([]byte(reactorId))

	_ = ctx.EventManager().EmitTypedEvent(&types.EventReactorDelete{ReactorId: reactorId})
}

// GetAllReactor returns all reactor
func (k Keeper) GetAllReactor(ctx sdk.Context, full bool) (list []types.Reactor) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Reactor
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		if full {
            val.Load        = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, val.Id))
            val.Capacity    = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, val.Id))
            val.Fuel        = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, val.Id))
		}

		list = append(list, val)
	}

	return
}


