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
	planet.Id = GetObjectID(types.ObjectType_planet, count)
	planet.SetCreator(player.Creator)
	planet.SetOwner(player.Id)


    k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_ore, planet.Id), types.PlanetStartingOre)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetKey))
	appendedValue := k.cdc.MustMarshal(&planet)
	store.Set([]byte(planet.Id), appendedValue)

	// Update planet count
	k.SetPlanetCount(ctx, count+1)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventPlanet{Planet: &planet})

	return planet
}

// SetPlanet set a specific planet in the store
func (k Keeper) SetPlanet(ctx sdk.Context, planet types.Planet) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetKey))
	b := k.cdc.MustMarshal(&planet)
	store.Set([]byte(planet.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventPlanet{Planet: &planet})
}

// GetPlanet returns a planet from its id
func (k Keeper) GetPlanet(ctx sdk.Context, planetId string) (val types.Planet, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetKey))
	b := store.Get([]byte(planetId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

    planetOre := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_ore, val.Id))
    playerOre := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_ore, val.Owner))

    val.OreRemaining = planetOre
    val.OreStored    = playerOre

	return val, true
}

// RemovePlanet removes a planet from the store
func (k Keeper) RemovePlanet(ctx sdk.Context, planetId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetKey))
	store.Delete([]byte(planetId))

    _ = ctx.EventManager().EmitTypedEvent(&types.EventDelete{ ObjectId: planetId })
}

// GetAllPlanet returns all planet
func (k Keeper) GetAllPlanet(ctx sdk.Context) (list []types.Planet) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlanetKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Planet
		k.cdc.MustUnmarshal(iterator.Value(), &val)

        planetOre := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_ore, val.Id))
        playerOre := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_ore, val.Owner))

		val.OreRemaining = planetOre
        val.OreStored    = playerOre

		list = append(list, val)
	}

	return
}


func (k Keeper) PlanetComplete(ctx sdk.Context, planet types.Planet) (bool) {
    if (planet.OreRemaining > 0) {
        return false
    }

    planet.SetStatus(1)
    k.SetPlanet(ctx, planet)
    return true

}

