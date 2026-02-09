package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//sdkerrors "cosmossdk.io/errors"

)


// GetNextPlayerId allocate a new substation ID
func (k Keeper) GetNextSubstationId(ctx context.Context) uint64 {

    nextId := k.GetSubstationCount(ctx)

    k.SetSubstationCount(ctx, nextId + 1)

	return nextId
}

// GetSubstationCount get the total number of substation
func (k Keeper) GetSubstationCount(ctx context.Context) uint64 {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.SubstationCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0 {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetSubstationCount set the total number of substation
func (k Keeper) SetSubstationCount(ctx context.Context, count uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.SubstationCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendSubstation appends a substation in the store with a new id and update the count
func (k Keeper) AppendSubstation(
	ctx context.Context,
    allocation types.Allocation,
    player types.Player,
) (substation types.Substation, updatedAllocation types.Allocation, err error) {
	// Set the ID of the appended value
    substation.Id = GetObjectID(types.ObjectType_substation, k.GetNextSubstationId(ctx))

    // Update the allocations new destination
    allocation.DestinationId = substation.Id

    power := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id))
    updatedAllocation, _, err = k.SetAllocation(ctx, allocation, power)
    if (err != nil) {
        return substation, updatedAllocation, err
    }

    // Setup some Substation details
    substation.Owner    = player.Id
    substation.Creator  = player.Creator

    permissionId := GetObjectPermissionIDBytes(substation.Id, player.Id)
    k.PermissionAdd(ctx, permissionId, types.PermissionAll)


    // actually commit to the store
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.SubstationKey))
	appendedValue := k.cdc.MustMarshal(&substation)
	store.Set([]byte(substation.Id), appendedValue)


    // Cache invalidation event
	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventSubstation{Substation: &substation})

	return substation, updatedAllocation, err
}

// SetSubstation set a specific substation in the store
func (k Keeper) SetSubstation(ctx context.Context, substation types.Substation) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.SubstationKey))
	b := k.cdc.MustMarshal(&substation)
	store.Set([]byte(substation.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventSubstation{Substation: &substation})

}


// ClearSubstation removes a substation from the store
func (k Keeper) ClearSubstation(ctx context.Context, substationId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.SubstationKey))

	store.Delete([]byte(substationId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ ObjectId: substationId })
}

// RemoveSubstation removes a substation from the store
func (k Keeper) RemoveSubstation(ctx context.Context, substationId string, migrationSubstationId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.SubstationKey))

	/*
	 * This is going to start out very inefficient. We'll need to tackle
	 * ways to improve these types of graph traversal
	 */
    playerConnections := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, substationId))
    if (playerConnections > 0) {

        connectedPlayers := k.GetAllPlayerBySubstation(ctx, substationId)
        // Need all players connected
        if (migrationSubstationId == "") {
            for _, disconnectPlayer := range connectedPlayers {
                k.SubstationDisconnectPlayer(ctx, disconnectPlayer)
            }
        } else {
            if (migrationSubstationId == substationId) {
                // TODO Move/copy this check to the message verification
                return  // error
            }
            migrationSubstation, migrationSubstationFound := k.GetSubstation(ctx, migrationSubstationId)
            if (!migrationSubstationFound) {
                return // error
            }
            for _, migratePlayer := range connectedPlayers {
                k.SubstationConnectPlayer(ctx, migrationSubstation, migratePlayer)
            }

        }
	}


    /* TODO
     * This isn't all super amazing, it's a lot of scans. Allocations Out being the
     * least of the problem but Allocations In will be super inefficient.
     *
     * Potential solution in the future is to have a Decommissioning state for substations
     * where the object isn't deleted until all other things are moved/disconnected but it
     * monitors allocation connection count until these values are zero.
     *
     * Basically, don't let it be deleted until that's all dealt with manually.
     */

	// Destroy allocations out
    allocationsOut := k.GetAllAllocationBySourceIndex(ctx, substationId)
    k.DestroyAllAllocations(ctx, allocationsOut)

	// Disconnect allocations in
    // TODO Need a more efficient way than scan
     allocationsIn := k.GetAllAllocationByDestinationIndex(ctx, substationId)
     k.DestroyAllAllocations(ctx, allocationsIn)


	// Clear out Grid attributes
	k.ClearGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, substationId))
    k.ClearGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, substationId))

    k.ClearGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, substationId))
    k.ClearGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, substationId))

    k.ClearGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_allocationPointerStart, substationId))
    k.ClearGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_allocationPointerEnd, substationId))

	store.Delete([]byte(substationId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ ObjectId: substationId })
}

// GetSubstation returns a substation from its id
func (k Keeper) GetSubstation(ctx context.Context, substationId string) (val types.Substation, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.SubstationKey))
	b := store.Get([]byte(substationId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// GetAllSubstation returns all substation
func (k Keeper) GetAllSubstation(ctx context.Context) (list []types.Substation) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.SubstationKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Substation
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		list = append(list, val)
	}

	return
}



func (k Keeper) SubstationConnectPlayer(ctx context.Context, substation types.Substation, player types.Player) (types.Player, error) {

    // If the player is already on a substation then disconnect them from it first
    if (player.SubstationId != "") {
        k.SetGridAttributeDecrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, player.SubstationId), 1)

        // Update Connection Capacity for the old Substation
        k.UpdateGridConnectionCapacity(ctx, player.SubstationId)
    }

    // Update the player record
    player.SubstationId = substation.Id
    // Commit the player changes
    k.SetPlayer(ctx, player)


    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, player.SubstationId), 1)
    // Update Connection Capacity
    k.UpdateGridConnectionCapacity(ctx, player.SubstationId)

    return player, nil

}

func (k Keeper) SubstationDisconnectPlayer(ctx context.Context, player types.Player) (types.Player, error) {

    // If the player is already on a substation then disconnect them from it first
    if (player.SubstationId != "") {
        k.SetGridAttributeDecrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, player.SubstationId), 1)
        // Update Connection Capacity for the old Substation
        k.UpdateGridConnectionCapacity(ctx, player.SubstationId)
    }

    // Update the player record
    player.SubstationId = ""
    // Commit the player changes
    k.SetPlayer(ctx, player)

    return player, nil
}
