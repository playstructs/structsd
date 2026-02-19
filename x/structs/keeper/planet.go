package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

)

// GetPlanetCount get the total number of planet
func (k Keeper) GetPlanetCount(ctx context.Context) uint64 {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.PlanetCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0  {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetPlanetCount set the total number of planet
func (k Keeper) SetPlanetCount(ctx context.Context, count uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.PlanetCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}


// SetPlanet set a specific planet in the store
func (k Keeper) SetPlanet(ctx context.Context, planet types.Planet) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlanetKey))
	b := k.cdc.MustMarshal(&planet)
	store.Set([]byte(planet.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPlanet{Planet: &planet})
}

// GetPlanet returns a planet from its id
func (k Keeper) GetPlanet(ctx context.Context, planetId string) (val types.Planet, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlanetKey))
	b := store.Get([]byte(planetId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// RemovePlanet removes a planet from the store
func (k Keeper) RemovePlanet(ctx context.Context, planetId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlanetKey))
	store.Delete([]byte(planetId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ ObjectId: planetId })
}

// GetAllPlanet returns all planet
func (k Keeper) GetAllPlanet(ctx context.Context) (list []types.Planet) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlanetKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Planet
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		list = append(list, val)
	}

	return
}

