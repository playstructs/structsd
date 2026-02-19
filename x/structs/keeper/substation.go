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

)


// GetNextPlayerId allocate a new substation ID
func (k Keeper) GetNextSubstationId(ctx context.Context) uint64 {
    nextId := k.GetSubstationCount(ctx)
    k.SetSubstationCount(ctx, nextId + 1)
	return nextId
}

// GetSubstationCount get the total number of substation
func (k Keeper) GetSubstationCount(ctx context.Context) uint64 {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.SubstationCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0 {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetSubstationCount set the total number of substation
func (k Keeper) SetSubstationCount(ctx context.Context, count uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.SubstationCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}


// SetSubstation set a specific substation in the store
func (k Keeper) SetSubstation(ctx context.Context, substation types.Substation) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.SubstationKey))
	b := k.cdc.MustMarshal(&substation)
	store.Set([]byte(substation.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventSubstation{Substation: &substation})
}


// ClearSubstation removes a substation from the store
func (k Keeper) ClearSubstation(ctx context.Context, substationId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.SubstationKey))

	store.Delete([]byte(substationId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ ObjectId: substationId })
}

// GetSubstation returns a substation from its id
func (k Keeper) GetSubstation(ctx context.Context, substationId string) (val types.Substation, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.SubstationKey))
	b := store.Get([]byte(substationId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// GetAllSubstation returns all substation
func (k Keeper) GetAllSubstation(ctx context.Context) (list []types.Substation) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.SubstationKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Substation
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		list = append(list, val)
	}

	return
}


