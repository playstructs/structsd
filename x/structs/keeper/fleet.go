package keeper

import (
	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

)


// AppendFleet appends a fleet in the store with a new id and update the count
func (k Keeper) AppendFleet(
	ctx context.Context,
	player types.Player,
) (fleet types.Fleet) {
    fleet = types.CreateEmptyFleet()

	// Set the ID of the appended value
	fleet.Id = GetObjectID(types.ObjectType_fleet, player.Index)
	fleet.Owner = player.Id


	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.FleetKey))
	appendedValue := k.cdc.MustMarshal(&fleet)
	store.Set([]byte(fleet.Id), appendedValue)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventFleet{Fleet: &fleet})

	return fleet
}

// SetFleet set a specific fleet in the store
func (k Keeper) SetFleet(ctx context.Context, fleet types.Fleet) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.FleetKey))
	b := k.cdc.MustMarshal(&fleet)
	store.Set([]byte(fleet.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventFleet{Fleet: &fleet})
}

// GetFleet returns a fleet from its id
func (k Keeper) GetFleet(ctx context.Context, fleetId string) (val types.Fleet, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.FleetKey))
	b := store.Get([]byte(fleetId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// RemoveFleet removes a fleet from the store
func (k Keeper) RemoveFleet(ctx context.Context, fleetId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.FleetKey))
	store.Delete([]byte(fleetId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ ObjectId: fleetId })
}

// GetAllFleet returns all fleet
func (k Keeper) GetAllFleet(ctx context.Context) (list []types.Fleet) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.FleetKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Fleet
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		list = append(list, val)
	}

	return
}


