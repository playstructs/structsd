package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
    //sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"

)

// GetStructCount get the total number of struct
func (k Keeper) GetStructCount(ctx context.Context) uint64 {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
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
func (k Keeper) SetStructCount(ctx context.Context, count uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.StructCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

/*
  string id         = 1;
  uint64 index      = 2;
  uint64 type       = 3;

  // Who is it
  string creator  = 4;
  string owner    = 5;

  // Where it is
  string  locationId      = 6;
  ambit   operatingAmbit  = 7;
  uint64  slot            = 8;
  */

// AppendStruct appends a struct in the store with a new id and update the count
func (k Keeper) AppendStruct(
	ctx context.Context,
	structure types.Struct
) (types.Struct) {
 	ctxSDK := sdk.UnwrapSDKContext(ctx)

	// Create the struct
	count := k.GetStructCount(ctx)

	// Set the ID of the appended value
	structure.Id = GetObjectID(types.ObjectType_struct, count)
	structure.Index = count

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.StructKey))
	appendedValue := k.cdc.MustMarshal(&structure)
	store.Set([]byte(structure.Id), appendedValue)

	// Update struct count
	k.SetStructCount(ctx, count+1)

    /*
        health                      = 0;
        status                      = 1;

        blockStartBuild             = 2;
        blockStartOreMine           = 3;
        blockStartOreRefine         = 4;

        protectedStructIndex        = 5;
    */
    // SetStructAttribute
	//structure.SetBuildStartBlock(uint64(ctxSDK.BlockHeight()))

    permissionId := GetObjectPermissionIDBytes(structure.Id, structure.Owner)
    k.PermissionAdd(ctx, permissionId, types.PermissionAll)

    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventStruct{Structure: &structure})

	return structure
}

// SetStruct set a specific struct in the store
func (k Keeper) SetStruct(ctx context.Context, structure types.Struct) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.StructKey))
	b := k.cdc.MustMarshal(&structure)
	store.Set([]byte(structure.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventStruct{Structure: &structure})
}

// GetStruct returns a struct from its id
func (k Keeper) GetStruct(ctx context.Context, structId string) (val types.Struct, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.StructKey))
	b := store.Get([]byte(structId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

    if (val.PowerSystem == 1) {
        val.PowerSystemFuel = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, val.Id))
        val.PowerSystemCapacity = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, val.Id))
        val.PowerSystemLoad = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, val.Id))
    }

	return val, true
}

// RemoveStruct removes a struct from the store
func (k Keeper) RemoveStruct(ctx context.Context, structId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.StructKey))
	store.Delete([]byte(structId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ ObjectId: structId })
}

// GetAllStruct returns all struct
func (k Keeper) GetAllStruct(ctx context.Context) (list []types.Struct) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.StructKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Struct
		k.cdc.MustUnmarshal(iterator.Value(), &val)

        if (val.PowerSystem == 1) {
            val.PowerSystemFuel = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, val.Id))
            val.PowerSystemCapacity = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, val.Id))
            val.PowerSystemLoad = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, val.Id))

        }

		list = append(list, val)
	}

	return
}





func (k Keeper) StructDeactivate(ctx context.Context, structId string) {
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
                allocations := k.GetAllocationsFromSource(ctx, structure.Id, false)
                k.DestroyAllAllocations(ctx, allocations)
            }
        }
    }
}




func (k Keeper) StructDestroy(ctx context.Context, structure types.Struct) {

    planet, planetFound := k.GetPlanet(ctx, structure.PlanetId)
    if (planetFound) {
        switch structure.Ambit {
            case "LAND":
                planet.Land[structure.Slot] = ""
        }

        k.SetPlanet(ctx, planet)
    }

    allocations := k.GetAllocationsFromSource(ctx, structure.Id, false)
    k.DestroyAllAllocations(ctx, allocations)

    infusions := k.GetAllInfusionsByDestination(ctx, structure.Id)
    k.DestroyAllInfusions(ctx, infusions)


    // Clear Load
    k.ClearGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, structure.Id ))

    // Clear Capacity
    k.ClearGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, structure.Id ))

    // Clear Fuel
    k.ClearGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, structure.Id ))

    // Clear Permissions
    permissionId := GetObjectPermissionIDBytes(structure.Id, structure.Owner)
    k.PermissionClearAll(ctx, permissionId)

    structure.SetStatus("DESTROYED")
	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventStruct{Structure: &structure})

    k.RemoveStruct(ctx, structure.Id)

}