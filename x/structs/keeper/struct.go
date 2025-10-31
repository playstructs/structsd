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
	"strconv"


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

    // Block Start Build
    k.SetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structure.Id),  uint64(ctxSDK.BlockHeight()))


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

func StructDestructionQueueReadKeyPrefix(blockHeight int64) []byte {
	return []byte(types.StructDestroyedQueueKey + strconv.FormatInt(blockHeight, 10) + "/")
}

func StructDestructionQueueAddKeyPrefix(blockHeight int64) []byte {
    sweepHeight := blockHeight + types.StructSweepDelay
	return []byte(types.StructDestroyedQueueKey + strconv.FormatInt(sweepHeight, 10) + "/")
}


func (k Keeper) AppendStructDestructionQueue(ctx context.Context, structId string) {
    k.logger.Info("Struct Sweep Queue Add", "structId", structId)

    unwrapCtx := sdk.UnwrapSDKContext(ctx)
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), StructDestructionQueueAddKeyPrefix(unwrapCtx.BlockHeight()))

	store.Set([]byte(structId), []byte{})
}


func (k Keeper) StructSweepDestroyed(ctx context.Context) {
    unwrapCtx := sdk.UnwrapSDKContext(ctx)
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), StructDestructionQueueReadKeyPrefix(unwrapCtx.BlockHeight()))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
        k.logger.Info("Struct Sweep", "structId", iterator.Key())

        // Attributes
        // "health":               StructAttributeType_health,
        k.ClearStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_health, string(iterator.Key()) ))
        // "status":               StructAttributeType_status,
        k.ClearStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_status, string(iterator.Key()) ))

        structure, structFound := k.GetStruct(ctx, string(iterator.Key()))
        if structFound {
            // Location Back-Reference
            switch structure.LocationType {
                case types.ObjectType_planet:
                    planet, planetFound := k.GetPlanet(ctx, structure.LocationId)
                    if planetFound {
                        switch structure.OperatingAmbit {
                            case types.Ambit_water:
                                planet.Water[structure.Slot] = ""
                            case types.Ambit_land:
                                planet.Land[structure.Slot]  = ""
                            case types.Ambit_air:
                                planet.Air[structure.Slot]   = ""
                            case types.Ambit_space:
                                planet.Space[structure.Slot] = ""
                        }
                        k.SetPlanet(ctx, planet)
                    }
                case types.ObjectType_fleet:
                    fleet, fleetFound := k.GetFleet(ctx, structure.LocationId)
                    if fleetFound {
                        if structure.Type == types.CommandStructTypeId {
                            fleet.CommandStruct = ""
                        } else {
                            switch structure.OperatingAmbit {
                                case types.Ambit_water:
                                    fleet.Water[structure.Slot] = ""
                                case types.Ambit_land:
                                    fleet.Land[structure.Slot]  = ""
                                case types.Ambit_air:
                                    fleet.Air[structure.Slot]   = ""
                                case types.Ambit_space:
                                    fleet.Space[structure.Slot] = ""
                            }
                        }
                        k.SetFleet(ctx, fleet)
                    }
            }
        }

        // Object
        k.RemoveStruct(ctx, string(iterator.Key()))

        store.Delete(iterator.Key())
	}
}