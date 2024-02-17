package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
    sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"

)

// GetStructCount get the total number of struct
func (k Keeper) GetStructCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.StructCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0  {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetStructCount set the total number of struct
func (k Keeper) SetStructCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.StructCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendStruct appends a struct in the store with a new id and update the count
func (k Keeper) AppendStruct(
	ctx sdk.Context,
	//struct types.Struct,
	player types.Player,
	structType string,
	planet types.Planet,
	slot uint64,
) (structure types.Struct) {
    structure = types.CreateBaseStruct(structType )

	// Create the struct
	count := k.GetStructCount(ctx)

	// Set the ID of the appended value
	structure.Id = count
	structure.SetCreator(player.Creator)
	structure.SetOwner(player.Id)
	structure.SetPlanetId(planet.Id)
	structure.SetSlot(slot)
	structure.SetBuildStartBlock(uint64(ctx.BlockHeight()))

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StructKey))
	appendedValue := k.cdc.MustMarshal(&structure)
	store.Set(GetObjectIDBytes(types.ObjectType_struct, structure.Id), appendedValue)

	// Update struct count
	k.SetStructCount(ctx, count+1)


	_ = ctx.EventManager().EmitTypedEvent(&types.EventStruct{Structure: &structure})

	return structure
}

// SetStruct set a specific struct in the store
func (k Keeper) SetStruct(ctx sdk.Context, structure types.Struct) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StructKey))
	b := k.cdc.MustMarshal(&structure)
	store.Set(GetObjectIDBytes(types.ObjectType_struct, structure.Id), b)

    _ = ctx.EventManager().EmitTypedEvent(&types.EventStruct{Structure: &structure})
}

// GetStruct returns a struct from its id
func (k Keeper) GetStruct(ctx sdk.Context, id uint64) (val types.Struct, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StructKey))
	b := store.Get(GetStructIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

    if (val.PowerSystem == 1) {
        val.PowerSystemFuel = k.StructGetFuel(ctx, val.Id)
        val.PowerSystemEnergy = k.StructGetEnergy(ctx, val.Id)
        val.PowerSystemLoad = k.StructGetLoad(ctx, val.Id)
    }

	return val, true
}

// RemoveStruct removes a struct from the store
func (k Keeper) RemoveStruct(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StructKey))
	store.Delete(GetObjectIDBytes(types.ObjectType_struct, id))

	_ = ctx.EventManager().EmitTypedEvent(&types.EventStructDelete{StructId: id})
}

// GetAllStruct returns all struct
func (k Keeper) GetAllStruct(ctx sdk.Context) (list []types.Struct) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StructKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Struct
		k.cdc.MustUnmarshal(iterator.Value(), &val)

        if (val.PowerSystem == 1) {
            val.PowerSystemFuel = k.StructGetFuel(ctx, val.Id)
            val.PowerSystemEnergy = k.StructGetEnergy(ctx, val.Id)
            val.PowerSystemLoad = k.StructGetLoad(ctx, val.Id)
        }

		list = append(list, val)
	}

	return
}





func (k Keeper) StructDeactivate(ctx sdk.Context, structId uint64) {
    structure, structureFound := k.GetStruct(ctx, structId)
    if (structureFound) {

        if (structure.Status == "ACTIVE") {

            structure.SetStatus("INACTIVE")

            if (structure.MiningSystem == 1) {
                structure.SetMiningSystemStatus("INACTIVE")
                structure.SetMiningSystemActivationBlock(0)
            }

            if (structure.RefiningSystem == 1) {
                structure.SetRefiningSystemStatus("INACTIVE")
                structure.SetRefiningSystemActivationBlock(0)
            }

            k.SetStruct(ctx, structure)

            if (structure.PowerSystem == 1) {
                k.StructDestroyAllocations(ctx, structure.Id)
            }
        }
    }
}



func (k Keeper) StructDestroyInfusions(ctx sdk.Context, structId uint64) {
    infusions := k.GetAllStructInfusions(ctx, structId)
    for _, infusion := range infusions {
        k.InfusionDestroy(ctx, infusion)
    }
}

func (k Keeper) StructDestroy(ctx sdk.Context, structure types.Struct) {

    planet, planetFound := k.GetPlanet(ctx, structure.PlanetId)
    if (planetFound) {
        switch structure.Ambit {
            case "LAND":
                planet.Land[structure.Slot] = 0
        }

        k.SetPlanet(ctx, planet)
    }

    k.StructDestroyAllocations(ctx, structure.Id)

    k.StructDestroyInfusions(ctx, structure.Id)

    storeLoad := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StructLoadKey))
    storeLoad.Delete(GetStructIDBytes(structure.Id))

    storeEnergy := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StructEnergyKey))
    storeEnergy.Delete(GetStructIDBytes(structure.Id))

    storeFuel := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StructFuelKey))
    storeFuel.Delete(GetStructIDBytes(structure.Id))

    k.RemoveStruct(ctx, structure.Id)

    structure.SetStatus("DESTROYED")

    _ = ctx.EventManager().EmitTypedEvent(&types.EventStruct{Structure: &structure})

}