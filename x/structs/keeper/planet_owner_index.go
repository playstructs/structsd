package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"structs/x/structs/types"
)

const planetOwnerIndexPrefix = "Planet/owner/"

func planetOwnerStore(k Keeper, ctx context.Context, owner string) prefix.Store {
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte(planetOwnerIndexPrefix+owner+"/"))
}

func (k Keeper) setPlanetOwnerIndex(ctx context.Context, planet types.Planet) {
	if planet.Owner != "" {
		planetOwnerStore(k, ctx, planet.Owner).Set([]byte(planet.Id), []byte{1})
	}
}

func (k Keeper) removePlanetOwnerIndex(ctx context.Context, planet types.Planet) {
	if planet.Owner != "" {
		planetOwnerStore(k, ctx, planet.Owner).Delete([]byte(planet.Id))
	}
}

// RebuildPlanetOwnerIndex is for upgrades only. Genesis rebuilds this derived
// index through SetPlanet; it does not need a separate wire representation.
func (k Keeper) RebuildPlanetOwnerIndex(ctx context.Context) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte(planetOwnerIndexPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, nil)
	var keys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, append([]byte(nil), iterator.Key()...))
	}
	iterator.Close()
	for _, key := range keys {
		store.Delete(key)
	}
	for _, planet := range k.GetAllPlanet(ctx) {
		k.setPlanetOwnerIndex(ctx, planet)
	}
}
