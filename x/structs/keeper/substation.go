package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

)


// GetNextPlayerId allocate a new substation ID
func (k Keeper) GetNextSubstationId(ctx sdk.Context) uint64 {

    nextId := k.GetSubstationCount(ctx)

    k.SetSubstationCount(ctx, nextId + 1)

	return nextId
}

// GetSubstationCount get the total number of substation
func (k Keeper) GetSubstationCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
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
func (k Keeper) SetSubstationCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.SubstationCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendSubstation appends a substation in the store with a new id and update the count
func (k Keeper) AppendSubstation(
	ctx sdk.Context,
    allocation types.Allocation,
    player types.Player,
) (substation types.Substation, updatedAllocation types.Allocation, err error) {
	// Set the ID of the appended value
    substation.Id = GetObjectID(types.ObjectType_substation, k.GetNextSubstationId(ctx))

    // Update the allocations new destination
    allocation.DestinationId = substation.Id
    updatedAllocation, err = k.SetAllocation(ctx, allocation)
    if (err != nil) {
        return substation, updatedAllocation, err
    }

    // Setup some Substation details
    substation.Owner    = player.Id
    substation.Creator  = player.Creator

    permissionId := GetObjectPermissionIDBytes(substation.Id, player.Id)
    k.PermissionAdd(ctx, permissionId, types.Permission(types.SubstationPermissionAll))


    // actually commit to the store
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	appendedValue := k.cdc.MustMarshal(&substation)
	store.Set([]byte(substation.Id), appendedValue)


    // Cache invalidation event
    _ = ctx.EventManager().EmitTypedEvent(&types.EventSubstation{Substation: &substation})

	return substation, updatedAllocation, err
}

// SetSubstation set a specific substation in the store
func (k Keeper) SetSubstation(ctx sdk.Context, substation types.Substation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	b := k.cdc.MustMarshal(&substation)
	store.Set([]byte(substation.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventSubstation{Substation: &substation})

}

// RemoveSubstation removes a substation from the store
func (k Keeper) RemoveSubstation(ctx sdk.Context, substationId string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))

	// TODO
	// Destroy allocations out
	// Disconnect allocations in
	// Disconnect Players
	// Clear out Grid attributes

	store.Delete([]byte(substationId))

	_ = ctx.EventManager().EmitTypedEvent(&types.EventSubstationDelete{SubstationId: substationId})
}

// GetSubstation returns a substation from its id
func (k Keeper) GetSubstation(ctx sdk.Context, substationId string, full bool) (val types.Substation, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	b := store.Get([]byte(substationId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	if full {
        val.Load                = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, val.Id))
        val.Capacity            = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, val.Id))
        val.ConnectionCount      = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, val.Id))
        val.ConnectionCapacity   = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, val.Id))
	}

	return val, true
}

// GetAllSubstation returns all substation
func (k Keeper) GetAllSubstation(ctx sdk.Context, full bool) (list []types.Substation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Substation
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		if full {
			val.Load                = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, val.Id))
			val.Capacity            = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, val.Id))
			val.ConnectionCount      = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, val.Id))
			val.ConnectionCapacity   = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, val.Id))
		}

		list = append(list, val)
	}

	return
}



func (k Keeper) SubstationConnectPlayer(ctx sdk.Context, substation types.Substation, player types.Player) (error) {

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


    return nil

}

func (k Keeper) SubstationDisconnectPlayer(ctx sdk.Context, player types.Player) (error) {

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

    return nil
}
