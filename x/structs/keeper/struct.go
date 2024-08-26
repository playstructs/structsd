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
	structure types.Struct,
	structureType types.StructType,
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

	// Emit the creation of the Struct object
	// Do this first, since the next commands will also emit related events.
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventStruct{Structure: &structure})

    // Set the Permissions
    permissionId := GetObjectPermissionIDBytes(structure.Id, structure.Owner)
    k.PermissionAdd(ctx, permissionId, types.PermissionAll)

    // Set the main Struct dynamic attributes
    // Current Health
    k.SetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_health, structure.Id),           structureType.MaxHealth)
    // Base Status (zero)
    k.SetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_status, structure.Id),           uint64(types.StructStateless))
    // Block Start Build
    k.SetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structure.Id),  uint64(ctxSDK.BlockHeight()))

    // Set the grid details
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, structure.Owner),structureType.BuildDraw)

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
		list = append(list, val)
	}

	return
}





func (k Keeper) StructDeactivate(ctx context.Context, structId string) {
   /*
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
    */
}




func (k Keeper) StructDestroy(ctx context.Context, structure types.Struct) {
    /*
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
    */

}




type StructCache struct {
    StructId string
    K *Keeper
    Ctx context.Context

    StructureLoaded  bool
    StructureChanged bool
    Structure  types.Struct

    StructTypeLoaded  bool
    StructType types.StructType

    HealthAttributeId string
    HealthLoaded  bool
    HealthChanged bool
    Health  uint64

    StatusAttributeId string
    StatusLoaded  bool
    StatusChanged bool
    Status types.StructState

    BlockStartBuildAttributeId string
    BlockStartBuildLoaded bool
    BlockStartBuildChanged bool
    BlockStartBuild  uint64

    BlockStartOreMineAttributeId string
    BlockStartOreMineLoaded bool
    BlockStartOreMineChanged bool
    BlockStarOreMine uint64

    BlockStartOreRefineAttributeId string
    BlockStartOreRefineLoaded bool
    BlockStartOreRefineChanged bool
    BlockStartOreRefine   uint64

    ProtectedStructIndexAttributeId string
    ProtectedStructIndexLoaded bool
    ProtectedStructIndexChanged bool
    ProtectedStructIndex   uint64
}

func (k *Keeper) GetStructCacheFromId(ctx context.Context, structId string) (StructCache) {
    return StructCache{
        StructId: structId,
        K: k,
        Ctx: ctx,

        HealthAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_health, structId),
        StatusAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_status, structId),

        BlockStartBuildAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structId),
        BlockStartOreMineAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, structId),
        BlockStartOreRefineAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, structId),

        ProtectedStructIndexAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, structId),
    }
}

func (cache *StructCache) Commit() () {

    if (cache.StructureChanged) { cache.K.SetStruct(cache.Ctx, cache.Structure) }

    if (cache.HealthChanged) { cache.K.SetStructAttribute(cache.Ctx, cache.HealthAttributeId, cache.Health) }
    if (cache.StatusChanged) { cache.K.SetStructAttribute(cache.Ctx, cache.StatusAttributeId, uint64(cache.Status)) }

    if (cache.BlockStartBuildChanged) { cache.K.SetStructAttribute(cache.Ctx, cache.BlockStartBuildAttributeId, cache.BlockStartBuild) }
    if (cache.BlockStartOreMineChanged) { cache.K.SetStructAttribute(cache.Ctx, cache.BlockStartOreMineAttributeId, cache.BlockStarOreMine) }
    if (cache.BlockStartOreRefineChanged) { cache.K.SetStructAttribute(cache.Ctx, cache.BlockStartOreRefineAttributeId, cache.BlockStartOreRefine) }

    if (cache.ProtectedStructIndexChanged) { cache.K.SetStructAttribute(cache.Ctx, cache.ProtectedStructIndexAttributeId, cache.ProtectedStructIndex) }
}


func (cache *StructCache) LoadStatus() {
    cache.Status = types.StructState(cache.K.GetStructAttribute(cache.Ctx, cache.StatusAttributeId))
    cache.StatusLoaded = true
}

func (cache *StructCache) LoadStruct() (bool) {
    cache.Structure, cache.StructureLoaded = cache.K.GetStruct(cache.Ctx, cache.StructId)
    return cache.StructureLoaded
}

func (cache *StructCache) LoadType() (bool) {
    cache.StructType, cache.StructTypeLoaded = cache.K.GetStructType(cache.Ctx, cache.GetTypeId())
    return cache.StructTypeLoaded
}



func (cache *StructCache) GetOwner() (string) {
    if (!cache.StructureLoaded) { cache.LoadStruct() }
    return cache.Structure.Owner
}

func (cache *StructCache) GetStatus() (types.StructState) {
    if (!cache.StatusLoaded) { cache.LoadStatus() }
    return cache.Status
}

func (cache *StructCache) GetStruct() (types.Struct) {
    if (!cache.StructureLoaded) { cache.LoadStruct() }
    return cache.Structure
}

func (cache *StructCache) GetTypeId() (uint64) {
    if (!cache.StructureLoaded) { cache.LoadStruct() }
    return cache.Structure.Type
}





func (cache *StructCache) IsBuilt() bool {
   return cache.GetStatus()&types.StructStateBuilt != 0
}

func (cache *StructCache) IsOnline() bool {
   return cache.GetStatus()&types.StructStateOnline != 0
}

func (cache *StructCache) IsOffline() bool {
    return !cache.IsOnline()
}