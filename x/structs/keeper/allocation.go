package keeper

import (
	//"encoding/binary"
    "context"
    "strconv"

    "github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k Keeper) SetAllocationOnly(ctx context.Context, allocation types.Allocation) (types.Allocation, error){

    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationKey))
    b := k.cdc.MustMarshal(&allocation)
    store.Set([]byte(allocation.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})

    return allocation,  nil

}


// ImportAllocation set a specific allocation in the store
// Assumes Grid updates happen elsewhere
func (k Keeper) ImportAllocation(ctx context.Context, allocation types.Allocation){
    k.SetAllocationSourceIndex(ctx, allocation.SourceObjectId, allocation.Id)
    k.SetAllocationDestinationIndex(ctx, allocation.DestinationId, allocation.Id)

    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationKey))
    b := k.cdc.MustMarshal(&allocation)
    store.Set([]byte(allocation.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})
}



// RemoveAllocation removes a allocation from the store
func (k Keeper) RemoveAllocation(ctx context.Context, allocationId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationKey))
	store.Delete([]byte(allocationId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ObjectId: allocationId})
}


// GetAllocation returns a allocation from its id
func (k Keeper) GetAllocation(ctx context.Context, allocationId string) (val types.Allocation, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationKey))
	b := store.Get([]byte(allocationId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// GetAllAllocation returns all allocation
func (k Keeper) GetAllAllocation(ctx context.Context) (list []types.Allocation) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AllocationKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Allocation
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		list = append(list, val)
	}

	return
}


