package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

)

// GetPlanetCount get the total number of planet
func (k Keeper) GetPlanetCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
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
func (k Keeper) SetPlanetCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.PlanetCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendPlanet appends a planet in the store with a new id and update the count
func (k Keeper) AppendPlanet(
	ctx sdk.Context,
	//planet types.Planet,
	player types.Player,
) (planet types.Planet) {
    planet = types.CreateEmptyPlanet()

	// Create the planet
	count := k.GetPlanetCount(ctx)

	// Set the ID of the appended value
	planet.Id = count
	planet.SetCreator(player.Creator)
	planet.SetOwner(player.Id)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetKey))
	appendedValue := k.cdc.MustMarshal(&planet)
	store.Set(GetPlanetIDBytes(planet.Id), appendedValue)

	// Update planet count
	k.SetPlanetCount(ctx, count+1)


	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: planet.Id, ObjectType: types.ObjectType_planet})

	return planet
}

// SetPlanet set a specific planet in the store
func (k Keeper) SetPlanet(ctx sdk.Context, planet types.Planet) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetKey))
	b := k.cdc.MustMarshal(&planet)
	store.Set(GetPlanetIDBytes(planet.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: planet.Id, ObjectType: types.ObjectType_planet})
}

// GetPlanet returns a planet from its id
func (k Keeper) GetPlanet(ctx sdk.Context, id uint64) (val types.Planet, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetKey))
	b := store.Get(GetPlanetIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	val.OreRemaining = val.MaxOre - k.GetPlanetRefinementCount(ctx, val.Id)
    val.OreStored    = k.GetPlanetOreCount(ctx, val.Id)

	return val, true
}

// RemovePlanet removes a planet from the store
func (k Keeper) RemovePlanet(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetKey))
	store.Delete(GetPlanetIDBytes(id))
}

// GetAllPlanet returns all planet
func (k Keeper) GetAllPlanet(ctx sdk.Context) (list []types.Planet) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Planet
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		val.OreRemaining = val.MaxOre - k.GetPlanetRefinementCount(ctx, val.Id)
        val.OreStored    = k.GetPlanetOreCount(ctx, val.Id)

		list = append(list, val)
	}

	return
}

// GetPlanetIDBytes returns the byte representation of the ID
func GetPlanetIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetPlanetIDFromBytes returns ID in uint64 format from a byte array
func GetPlanetIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) PlanetComplete(ctx sdk.Context, planet types.Planet) (bool) {
    if (k.GetPlanetRefinementCount(ctx, planet.Id) < planet.MaxOre) {
        return false
    }

    planet.SetStatus(1)
    k.SetPlanet(ctx, planet)
    return true

}

func (k Keeper) GetPlanetRefinementCount(ctx sdk.Context, planetId uint64) (count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetRefinementCountKey))

	bz := store.Get(GetPlanetIDBytes(planetId))

	if bz == nil {
		count = 0
	} else {
		count = binary.BigEndian.Uint64(bz)
	}

	return
}


func (k Keeper) SetPlanetRefinementCount(ctx sdk.Context, planetId uint64, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetRefinementCountKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)

	store.Set(GetPlanetIDBytes(planetId), bz)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: planetId, ObjectType: types.ObjectType_planet})
}


func (k Keeper) IncreasePlanetRefinementCount(ctx sdk.Context, planetId uint64) (uint64) {
    current := k.GetPlanetRefinementCount(ctx, planetId)
    current = current + 1

    k.SetPlanetRefinementCount(ctx, planetId, current)
    return current
}




func (k Keeper) GetPlanetOreCount(ctx sdk.Context, planetId uint64) (count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetOreCountKey))

	bz := store.Get(GetPlanetIDBytes(planetId))

	if bz == nil {
		count = 0
	} else {
		count = binary.BigEndian.Uint64(bz)
	}

	return
}


func (k Keeper) SetPlanetOreCount(ctx sdk.Context, planetId uint64, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetOreCountKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)

	store.Set(GetPlanetIDBytes(planetId), bz)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: planetId, ObjectType: types.ObjectType_planet})
}


func (k Keeper) IncreasePlanetOreCount(ctx sdk.Context, planetId uint64) (uint64) {
    current := k.GetPlanetOreCount(ctx, planetId)
    current = current + 1

    k.SetPlanetOreCount(ctx, planetId, current)
    return current
}

// TODO convert this into a function that errors out if already 0
func (k Keeper) DecreasePlanetOreCount(ctx sdk.Context, planetId uint64) (uint64) {
    current := k.GetPlanetOreCount(ctx, planetId)
    current = current - 1

    k.SetPlanetOreCount(ctx, planetId, current)
    return current
}

