package keeper

import (
	"encoding/binary"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
	"structs/x/structs/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strconv"
)

// GetReactorCount get the total number of reactor
func (k Keeper) GetReactorCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.ReactorCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0 {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetReactorCount set the total number of reactor
func (k Keeper) SetReactorCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.ReactorCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// GetReactorBytesFromValidator get the bytes based on validator address
func (k Keeper) GetReactorBytesFromValidator(ctx sdk.Context, validatorAddress []byte) (reactorBytes []byte, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorValidatorKey))

	reactorBytes = store.Get(validatorAddress)
	// Count doesn't exist: no element
	if reactorBytes == nil {
		return reactorBytes, false
	}

	return reactorBytes, true
}

// SetReactorValidatorBytes set the validator address index bytes
func (k Keeper) SetReactorValidatorBytes(ctx sdk.Context, id uint64, validatorAddress []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorValidatorKey))

	store.Set(validatorAddress, GetReactorIDBytes(id))
}

// AppendReactor appends a reactor in the store with a new id and update the count
func (k Keeper) AppendReactor(
	ctx sdk.Context,
	reactor types.Reactor,
) uint64 {
	// Create the reactor
	count := k.GetReactorCount(ctx)

	// Set the ID of the appended value
	reactor.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	appendedValue := k.cdc.MustMarshal(&reactor)
	store.Set(GetReactorIDBytes(reactor.Id), appendedValue)

	// Add a record to the Validator index
	k.SetReactorValidatorBytes(ctx, reactor.Id, reactor.Validator)

	// Update reactor count
	k.SetReactorCount(ctx, count+1)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: reactor.Id, ObjectType: types.ObjectType_reactor})


	return count
}

// SetReactor set a specific reactor in the store
func (k Keeper) SetReactor(ctx sdk.Context, reactor types.Reactor) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	b := k.cdc.MustMarshal(&reactor)
	store.Set(GetReactorIDBytes(reactor.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: reactor.Id, ObjectType: types.ObjectType_reactor})
}

// GetReactor returns a reactor from its id
func (k Keeper) GetReactor(ctx sdk.Context, id uint64, full bool) (val types.Reactor, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	b := store.Get(GetReactorIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	if full {
    	val.Energy = k.ReactorGetEnergy(ctx, val.Id)
		val.Load = k.ReactorGetLoad(ctx, val.Id)
	}

	return val, true
}

// GetReactor returns a reactor from its id
func (k Keeper) GetReactorByBytes(ctx sdk.Context, id []byte, full bool) (val types.Reactor, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	b := store.Get(id)
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	if full {
	    val.Energy = k.ReactorGetEnergy(ctx, val.Id)
		val.Load = k.ReactorGetLoad(ctx, val.Id)
	}

	return val, true
}

// RemoveReactor removes a reactor from the store
func (k Keeper) RemoveReactor(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	store.Delete(GetReactorIDBytes(id))
}

// GetAllReactor returns all reactor
func (k Keeper) GetAllReactor(ctx sdk.Context, full bool) (list []types.Reactor) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Reactor
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		if full {
		    val.Energy = k.ReactorGetEnergy(ctx, val.Id)
			val.Load = k.ReactorGetLoad(ctx, val.Id)
		}

		list = append(list, val)
	}

	return
}

// GetReactorIDBytes returns the byte representation of the ID
func GetReactorIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetReactorIDFromBytes returns ID in uint64 format from a byte array
func GetReactorIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}


func (k Keeper) ReactorCheckIdentity(ctx sdk.Context, reactor types.Reactor, validator staking.Validator) (bool, error) {

    if (reactor.Activated) {
        return false, sdkerrors.Wrapf(types.ErrReactorActivation, "Reactor already activated")
    }

    identity := validator.Description.Identity

    if (identity == "") {
        return false, sdkerrors.Wrapf(types.ErrReactorActivation, "Identity Missing for Reactor Activation")
    }
    // TODO verify that identity is actually an address
        // return error about wrong identity format


    if (reactor.Energy < types.InitialReactorAllocation) {
        energyString := strconv.FormatUint(types.InitialReactorAllocation, 10)
        return false, sdkerrors.Wrapf(types.ErrReactorActivation, "Reactor Activation Requires %s Energy", energyString)
    }

    playerId := k.GetPlayerIdFromAddress(ctx, identity)

    var player types.Player
    if (playerId == 0) {
        // No Player Found, Creating..
        player = types.CreateEmptyPlayer()
        player.SetCreator(identity)

        k.AppendPlayer(ctx, player)

        // TODO Add Related Address
    } else {
       player, _ = k.GetPlayer(ctx, playerId)
    }

    // Apply Ownership Permissions of the Reactor to the Player
    k.ReactorPermissionAdd(ctx, reactor.Id, player.Id, types.ReactorPermissionAll)

    return true, nil

}

// Iterate through the allocations, starting from oldest, and destroy them until power
// consumption matches output
func (k Keeper) CascadeReactorAllocationFailure(ctx sdk.Context, reactor types.Reactor) {
	allocations := k.GetAllReactorAllocations(ctx, reactor.Id)
	for _, allocation := range allocations {
		if k.ReactorGetEnergy(ctx, reactor.Id) > k.ReactorGetLoad(ctx, reactor.Id) {
			break
		}

		k.AllocationDestroy(ctx, allocation)
	}
}

func (k Keeper) ReactorDecrementLoad(ctx sdk.Context, id uint64, amount uint64) (new uint64, err error) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.ReactorLoadKey))

	current := k.ReactorGetLoad(ctx, id)

	if amount > current {
		// this really shouldn't happen. Throw an error I guess but yeesh, this is a problem.
	} else {
		new = current - amount
	}

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, new)
	store.Set(GetReactorIDBytes(id), bz)

	return
}

func (k Keeper) ReactorIncrementLoad(ctx sdk.Context, id uint64, amount uint64) (uint64, error) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.ReactorLoadKey))

	current := k.ReactorGetLoad(ctx, id)

	new := current + amount

	reactorEnergy := k.ReactorGetEnergy(ctx, id)

	if new > reactorEnergy {
		reactorId := strconv.FormatUint(id, 10)
		return 0, sdkerrors.Wrapf(types.ErrReactorAvailableCapacityInsufficient, "source (%s) used for allocation not sufficient", "reactor-"+reactorId)
	}

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, new)
	store.Set(GetReactorIDBytes(id), bz)

	return new, nil
}

// ReactorGetLoad returns the current load of all allocations
// Go to memory first, but then fall back to rebuilding from allocations
func (k Keeper) ReactorGetLoad(ctx sdk.Context, id uint64) (load uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.ReactorLoadKey))

	bz := store.Get(GetReactorIDBytes(id))

	// Reactor Capacity Not in Memory: no element
	if bz == nil {
		load = k.ReactorRebuildLoad(ctx, id)
		k.ReactorSetLoad(ctx, id, load)

	} else {
		load = binary.BigEndian.Uint64(bz)
	}

	return load
}

// ReactorSetLoad - Sets the in-memory representation of the aggregate load of all associated allocations
func (k Keeper) ReactorSetLoad(ctx sdk.Context, id uint64, amount uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.ReactorLoadKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, amount)

	store.Set(GetReactorIDBytes(id), bz)
	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: id, ObjectType: types.ObjectType_reactor})
}

// ReactorRebuildLoad - Rebuilds the current load by iterating through all related allocations
func (k Keeper) ReactorRebuildLoad(ctx sdk.Context, id uint64) (load uint64) {
	allocations := k.GetAllReactorAllocations(ctx, id)

	for _, allocation := range allocations {
		load += allocation.Power
	}

	return
}


// ReactorRebuildLoad - Rebuilds the current load by iterating through all related infusions
func (k Keeper) ReactorRebuildEnergy(ctx sdk.Context, id uint64) (energy uint64) {
	infusions := k.GetAllReactorInfusions(ctx, id)

	for _, infusion := range infusions {
	    // TODO ADD A REAL RATIO CALC
		energy += infusion.Fuel * 10
	}

	return
}


// ReactorGetEnergy returns the current energy production of the reactor
// Go to memory first, but then fall back to rebuilding from storage
func (k Keeper) ReactorGetEnergy(ctx sdk.Context, id uint64) (energy uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.ReactorEnergyKey))

	bz := store.Get(GetReactorIDBytes(id))

	// Reactor Energy Not in Memory: no element
	if bz == nil {
		energy = k.ReactorRebuildEnergy(ctx, id)
		k.ReactorSetEnergy(ctx, id, energy)

	} else {
		energy = binary.BigEndian.Uint64(bz)
	}

	return energy
}

// ReactorSetEnergy- Sets the in-memory representation of the reactors energy production
func (k Keeper) ReactorSetEnergy(ctx sdk.Context, id uint64, amount uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.ReactorEnergyKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, amount)

	store.Set(GetReactorIDBytes(id), bz)
	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: id, ObjectType: types.ObjectType_reactor})
}


