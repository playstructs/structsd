package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

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
	//substation types.Substation,
	playerConnectionAllocation uint64,
    player types.Player,
) (types.Substation) {
	substation := types.CreateEmptySubstation()

	// Set the ID of the appended value
    substation.SetId(k.GetNextSubstationId(ctx))

    // Setup some Substation details
    substation.SetOwner(player.Id)
    substation.SetCreator(player.Creator)
    substation.SetPlayerConnectionAllocation(playerConnectionAllocation)
    k.SubstationPermissionAdd(ctx, substation.Id, player.Id, types.SubstationPermissionAll)


    // actually commit to the store
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	appendedValue := k.cdc.MustMarshal(&substation)
	store.Set(GetSubstationIDBytes(substation.Id), appendedValue)


    // Cache invalidation event
    _ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: substation.Id, ObjectType: types.ObjectType_substation})

	return substation
}

// SetSubstation set a specific substation in the store
func (k Keeper) SetSubstation(ctx sdk.Context, substation types.Substation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	b := k.cdc.MustMarshal(&substation)
	store.Set(GetSubstationIDBytes(substation.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: substation.Id, ObjectType: types.ObjectType_substation})

}

// GetSubstation returns a substation from its id
func (k Keeper) GetSubstation(ctx sdk.Context, id uint64, full bool) (val types.Substation, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	b := store.Get(GetSubstationIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	if full {
		val.Load = k.SubstationGetLoad(ctx, val.Id)
		val.Energy = k.SubstationGetEnergy(ctx, val.Id)
		val.ConnectedPlayerCount = k.SubstationGetConnectedPlayerCount(ctx, val.Id)
	}

	return val, true
}

// RemoveSubstation removes a substation from the store
func (k Keeper) RemoveSubstation(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	store.Delete(GetSubstationIDBytes(id))
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
			val.Load = k.SubstationGetLoad(ctx, val.Id)
			val.Energy = k.SubstationGetEnergy(ctx, val.Id)
			val.ConnectedPlayerCount = k.SubstationGetConnectedPlayerCount(ctx, val.Id)
		}

		list = append(list, val)
	}

	return
}

// GetSubstationIDBytes returns the byte representation of the ID
func GetSubstationIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetSubstationIDFromBytes returns ID in uint64 format from a byte array
func GetSubstationIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

// UpdateSubstationStatus(ctx sdk.Context)
// Run at the EndBlock to update all Substations for the next block
func (k Keeper) UpdateSubstationStatus(ctx sdk.Context) {

}

// Iterate through the allocations, starting from oldest, and destroy them until power
// consumption matches output
//
// This function can end up operating in
// a recursive loop as Allocations effect
// other Substations
func (k Keeper) CascadeSubstationAllocationFailure(ctx sdk.Context, substationId uint64) {

	// do a check first before spending the computation resources to load the allocation list
	if k.SubstationGetEnergy(ctx, substationId) > k.SubstationGetLoad(ctx, substationId) {
		return
	}

	allocations := k.GetAllSubstationAllocationIn(ctx, substationId)
	for _, allocation := range allocations {
		if k.SubstationGetEnergy(ctx, substationId) > k.SubstationGetLoad(ctx, substationId) {
			break
		}

		k.AllocationDestroy(ctx, allocation)
	}

	// If we get here and Energy is still less than load, then we need to deactivate the structs
    if k.SubstationGetEnergy(ctx, substationId) < k.SubstationGetLoad(ctx, substationId) {
        players := k.GetAllPlayerBySubstation(ctx, substationId)
        for _, player := range players {
            planet, planetFound := k.GetPlanet(ctx, player.PlanetId)
            // TODO Lets hardcore some laziness but fix once we have a better idea
            if (planetFound) {
                for i := uint64(0); i < planet.LandSlots; i++ {
                    if (planet.Land[i] != 0) {
                        k.StructDeactivate(ctx, planet.Land[i])
                    }
                }
            }
        }
    }
}

// For use when an allocation is being destroyed and the power is no longer applied to the load
// of a substation.
//
// Updates the Allocation Load and Total Load memory values
func (k Keeper) SubstationDecrementAllocationLoad(ctx sdk.Context, id uint64, amount uint64) (newAllocationLoad uint64, err error) {
	currentAllocationLoad := k.SubstationGetAllocationLoad(ctx, id)

	if amount > currentAllocationLoad {
		// this really shouldn't happen. Throw an error I guess but yeesh, this is a problem.
	} else {
		newAllocationLoad = currentAllocationLoad - amount
	}

	k.SubstationSetAllocationLoad(ctx, id, newAllocationLoad)

	currentTotalLoad := k.SubstationGetLoad(ctx, id)
	newTotalLoad := currentTotalLoad - amount
	k.SubstationSetLoad(ctx, id, newTotalLoad)

	return newAllocationLoad, err
}

// For use when an allocation is being created and the power of the allocation is applied to the load
// of a substation.
//
// Substation must have room available in the load/energy ratio for the new allocation or the process is aborted.
//
// Updates the Allocation Load and Total Load memory values
func (k Keeper) SubstationIncrementAllocationLoad(ctx sdk.Context, id uint64, amount uint64) (uint64, error) {

	currentTotalLoad := k.SubstationGetLoad(ctx, id)
	currentAllocationLoad := k.SubstationGetAllocationLoad(ctx, id)

	newTotalLoad := currentTotalLoad + amount
	newAllocationLoad := currentAllocationLoad + amount

	substationEnergy := k.SubstationGetEnergy(ctx, id)

	if newTotalLoad > substationEnergy {
		return 0, sdkerrors.Wrapf(types.ErrSubstationAvailableCapacityInsufficient, "source (substation-%d) used for allocation insufficient", id)
	}

	k.SubstationSetAllocationLoad(ctx, id, newAllocationLoad)
	k.SubstationSetLoad(ctx, id, newTotalLoad)

	return newTotalLoad, nil
}

// For use when a player account is being disconnected and the power is no longer applied to the load
// of a substation.
//
// Updates the Connected Player Load and Total Load memory values
func (k Keeper) SubstationDecrementConnectedPlayerLoad(ctx sdk.Context, id uint64, amount uint64) (uint64, error) {

	_ = k.SubstationDecrementConnectedPlayerCount(ctx, id, amount)

	newConnectedPlayerLoad := k.SubstationRebuildConnectedPlayerLoad(ctx, id)

	k.SubstationSetConnectedPlayerLoad(ctx, id, newConnectedPlayerLoad)


	return newConnectedPlayerLoad, nil
}

// For use when a new player is connected and the power of the connection slot is applied to the load
// of a substation.
//
// Substation must have room available in the load/energy ratio for the new player connection or the process is aborted.
//
// Updates the Connected Player Load and Total Load memory values
func (k Keeper) SubstationIncrementConnectedPlayerLoad(ctx sdk.Context, id uint64, amount uint64) (uint64, error) {


	_ = k.SubstationIncrementConnectedPlayerCount(ctx, id, amount)

	newConnectedPlayerLoad := k.SubstationRebuildConnectedPlayerLoad(ctx, id)
	//currentAllocationLoad := k.SubstationGetAllocationLoad(ctx, id)

	//newTotalLoad := newConnectedPlayerLoad + currentAllocationLoad

	//substationEnergy := k.SubstationGetEnergy(ctx, id)

    /*
	if newTotalLoad > substationEnergy {
		k.SubstationDecrementConnectedPlayerCount(ctx, id, amount)
		substationId := strconv.FormatUint(id, 10)
		return 0, sdkerrors.Wrapf(types.ErrSubstationAvailableCapacityInsufficient, "source (%s) used for allocation insufficient", "substation-"+substationId)
	}
	*/

	k.SubstationSetConnectedPlayerLoad(ctx, id, newConnectedPlayerLoad)

	return newConnectedPlayerLoad, nil
}

// SubstationGetLoad returns the current total load of the substation
// Go to memory first, but then fall back to rebuilding from storage
func (k Keeper) SubstationGetLoad(ctx sdk.Context, id uint64) (load uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.SubstationLoadKey))

	bz := store.Get(GetSubstationIDBytes(id))

	// Substation Capacity Not in Memory: no element
	if bz == nil {
		allocationLoad := k.SubstationGetAllocationLoad(ctx, id)

		connectedPlayerLoad := k.SubstationGetConnectedPlayerLoad(ctx, id)

		load = allocationLoad + connectedPlayerLoad
		k.SubstationSetLoad(ctx, id, load)

	} else {
		load = binary.BigEndian.Uint64(bz)
	}

	return load
}

// SubstationGetAllocationLoad returns the current load of all allocations
// Go to memory first, but then fall back to rebuilding from allocations
func (k Keeper) SubstationGetAllocationLoad(ctx sdk.Context, id uint64) (load uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.SubstationAllocationLoadKey))

	bz := store.Get(GetSubstationIDBytes(id))

	// Substation Capacity Not in Memory: no element
	if bz == nil {
		load = k.SubstationRebuildAllocationLoad(ctx, id) + 0
		k.SubstationSetAllocationLoad(ctx, id, load)
	} else {
		load = binary.BigEndian.Uint64(bz)
	}

	return load
}

// SubstationGetConnectedPlayerLoad returns the current load of all allocations
// Go to memory first, but then fall back to rebuilding from allocations
func (k Keeper) SubstationGetConnectedPlayerLoad(ctx sdk.Context, id uint64) (load uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.SubstationConnectedPlayerLoadKey))

	bz := store.Get(GetSubstationIDBytes(id))

	// Substation Capacity Not in Memory: no element
	if bz == nil {
		connectedPlayerLoad := k.SubstationRebuildConnectedPlayerLoad(ctx, id)
		k.SubstationSetConnectedPlayerLoad(ctx, id, connectedPlayerLoad)
	} else {
		load = binary.BigEndian.Uint64(bz)
	}

	return load
}

// SubstationSetLoad - Sets the in-memory representation of the aggregate load of all associated allocations
func (k Keeper) SubstationSetLoad(ctx sdk.Context, id uint64, amount uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.SubstationLoadKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, amount)

	store.Set(GetSubstationIDBytes(id), bz)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: id, ObjectType: types.ObjectType_substation})
}

// SubstationSetAllocationLoad - Sets the in-memory representation of the aggregate load of all associated allocations
func (k Keeper) SubstationSetAllocationLoad(ctx sdk.Context, id uint64, amount uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.SubstationAllocationLoadKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, amount)

	store.Set(GetSubstationIDBytes(id), bz)

	connectedPlayerLoad := k.SubstationGetConnectedPlayerLoad(ctx, id)
	k.SubstationSetLoad(ctx, id, amount + connectedPlayerLoad)
}

// Sets the in-memory representation of the aggregate load of all connected players
func (k Keeper) SubstationSetConnectedPlayerLoad(ctx sdk.Context, id uint64, amount uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.SubstationConnectedPlayerLoadKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, amount)

	store.Set(GetSubstationIDBytes(id), bz)

    allocationLoad := k.SubstationGetAllocationLoad(ctx, id)


    newAmount := amount + allocationLoad
    k.SubstationSetLoad(ctx, id, newAmount)

}

// Rebuilds the current load by iterating through all related allocations
func (k Keeper) SubstationRebuildAllocationLoad(ctx sdk.Context, id uint64) (load uint64) {
	allocations := k.GetAllSubstationAllocationOut(ctx, id)

    for _, allocation := range allocations {
        load += allocation.Power
    }

	return
}

// Rebuilds the current player connection load
func (k Keeper) SubstationRebuildConnectedPlayerLoad(ctx sdk.Context, id uint64) (load uint64) {
	connectedPlayerCount := k.SubstationGetConnectedPlayerCount(ctx, id)

	substation, _ := k.GetSubstation(ctx, id, false)
	load = connectedPlayerCount * substation.PlayerConnectionAllocation

	return
}

// SubstationGetEnergy returns the current aggregate energy supply across all allocations
// Go to memory first, but then fall back to rebuilding from storage
func (k Keeper) SubstationGetEnergy(ctx sdk.Context, id uint64) (load uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.SubstationEnergyKey))

	bz := store.Get(GetSubstationIDBytes(id))

	// Reactor Energy Not in Memory: no element
	if bz == nil {
		load = k.SubstationRebuildAllocationEnergy(ctx, id)
		k.SubstationSetEnergy(ctx, id, load)

	} else {
		load = binary.BigEndian.Uint64(bz)
	}

	return load
}

// Rebuilds the current available energy by iterating through all related allocations
func (k Keeper) SubstationRebuildAllocationEnergy(ctx sdk.Context, id uint64) (load uint64) {
	allocations := k.GetAllSubstationAllocationIn(ctx, id)

	for _, allocation := range allocations {
		load += allocation.Power
	}

	return
}

// Sets the in-memory representation of the substations available energy
func (k Keeper) SubstationSetEnergy(ctx sdk.Context, id uint64, amount uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.SubstationEnergyKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, amount)

	store.Set(GetReactorIDBytes(id), bz)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: id, ObjectType: types.ObjectType_substation})

}

// Used to decrement the Substation Energy memory value
func (k Keeper) SubstationDecrementEnergy(ctx sdk.Context, id uint64, amount uint64) (new uint64) {

	current := k.SubstationGetEnergy(ctx, id)

	if amount > current {
		// this really shouldn't happen. Throw an error I guess but yeesh, this is a problem.
	} else {
		new = current - amount
	}

	k.SubstationSetEnergy(ctx, id, new)

	return
}

// Used to increment the Substation Energy memory value
func (k Keeper) SubstationIncrementEnergy(ctx sdk.Context, id uint64, amount uint64) (new uint64) {
	current := k.SubstationGetEnergy(ctx, id)

	new = current + amount

	k.SubstationSetEnergy(ctx, id, new)

	return new
}

// return the number of players currently sourcing energy from a substation
func (k Keeper) SubstationGetConnectedPlayerCount(ctx sdk.Context, id uint64) (count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationConnectedPlayerCount))

	bz := store.Get(GetSubstationIDBytes(id))

	// No connected player count set for Substation yet: no element
	if bz == nil {
		count = 0
	} else {
		count = binary.BigEndian.Uint64(bz)
	}

	return
}

// Increment the number of players currently sourcing energy from a substation
//
// This function does not also update the Substation Load memory values, which must be done separately!
func (k Keeper) SubstationIncrementConnectedPlayerCount(ctx sdk.Context, id uint64, amount uint64) (count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationConnectedPlayerCount))

	count = k.SubstationGetConnectedPlayerCount(ctx, id)
	count = count + amount

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)

	store.Set(GetSubstationIDBytes(id), bz)

	return
}

// Decrement the number of players currently sourcing energy from a substation
//
// This function does not also update the Substation Load memory values, which must be done separately!
func (k Keeper) SubstationDecrementConnectedPlayerCount(ctx sdk.Context, id uint64, amount uint64) (count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationConnectedPlayerCount))

	count = k.SubstationGetConnectedPlayerCount(ctx, id)
	count = count - amount

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)

	store.Set(GetSubstationIDBytes(id), bz)

	return
}



func (k Keeper) SubstationConnectPlayer(ctx sdk.Context, substation types.Substation, player types.Player) (error) {

    // If the player is already on a substation then disconnect them from it first
    if (player.SubstationId > 0) {
        k.SubstationDecrementConnectedPlayerLoad(ctx, player.SubstationId, 1)
    }

    // Connect Player to Substation
    k.SubstationIncrementConnectedPlayerLoad(ctx, substation.Id, 1)

    player.SetSubstation(substation.Id)
    k.SetPlayer(ctx, player)

    return nil

}

func (k Keeper) SubstationConnectAllocation(ctx sdk.Context, substation types.Substation, allocation types.Allocation)  (error) {

    // Check to see if already connected
    if (allocation.DestinationId == substation.Id) {
        // TODO add real error
        return nil
    }

	// Check to see if there is already a destination Substation using this.
	// Disconnect it if so
	if (allocation.DestinationId > 0) {
		_ = k.SubstationDecrementEnergy(ctx, allocation.DestinationId, allocation.Power)
		k.CascadeSubstationAllocationFailure(ctx, allocation.DestinationId)
	}

	allocation.SetDestinationId(substation.Id)
	k.SetAllocation(ctx, allocation)

    return nil
}

func (k Keeper) SubstationIsOnline(ctx sdk.Context, substationId uint64) (bool) {
    return (k.SubstationGetEnergy(ctx, substationId) >= k.SubstationGetLoad(ctx, substationId))
}