package keeper

import (
	"encoding/binary"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
	"structs/x/structs/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"strconv"
	"cosmossdk.io/math"
)

/* Setup Reactor (when a validator is created)
 *
 * Triggered during Staking Hooks:
 *   AfterValidatorCreated
 */
func (k Keeper) ReactorInitialize(ctx sdk.Context, validatorAddress sdk.ValAddress) (reactor types.Reactor) {

	validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.String())
	if reactorBytesFound {
		reactor, _ = k.GetReactorByBytes(ctx, reactorBytes, false)
	} else {
		/* Build the initial Reactor object */
		reactor = types.CreateEmptyReactor()
		reactor.SetValidator(validatorAddress.String())
		k.SetReactorValidatorBytes(ctx, reactor.Id, validatorAddress.String())

		/* TODO: Permissions
		 *
		 * Create permissions based on the details field.
		 *  Link to Faction
		 *  Link to Player
		 */

		/* Sync Energy Levels
		 *
		 * Set the initial power level of the Reactor based on the
		 * tokens staked to the validator
		 */
		reactor.SetEnergy(validator)

		/*
		 * Commit Reactor to the Keeper
		 */
		reactorId := k.AppendReactor(ctx, reactor)
        reactor.SetId(reactorId)

        activated, _ := k.ReactorActivate(ctx, reactor, validator)
        if (activated) {
            reactor.SetActivated(true)
            k.SetReactor(ctx, reactor)
        }



	}
	return reactor
}

/* Change Reactor Energy (Anytime Delegations Change)
 *
 * Triggered during Staking Hooks:
 *   AfterValidatorBeginUnbonding
 *   AfterDelegationRemoved (Doesn't actually exist yet)
 *   AfterDelegationModified
 *   AfterValidatorBonded
 *
 */
func (k Keeper) ReactorUpdateEnergy(ctx sdk.Context, validatorAddress sdk.ValAddress) (reactor types.Reactor) {

	validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.String())
	if reactorBytesFound {
		reactor, _ = k.GetReactorByBytes(ctx, reactorBytes, false)
	}

	/* Sync Energy Levels
	 *
	 * Set the initial power level of the Reactor based on the
	 * tokens staked to the validator
	 */
	reactor.SetEnergy(validator)

	/* TODO: Permissions
	 *
	 * Create permissions based on the details field.
	 *  Link to Faction
	 *  Link to Player
	 */

	k.SetReactor(ctx, reactor)

    activated, _ := k.ReactorActivate(ctx, reactor, validator)
    if (activated) {
        reactor.SetActivated(true)
        k.SetReactor(ctx, reactor)
    }

	// Update the connected Substations with the new details
	k.CascadeReactorAllocationFailure(ctx, reactor)


	return reactor
}



/* Change Reactor Allocations for Player Delegations
 *
 * Triggered during Staking Hooks:
 *   AfterDelegationModified
 *
 */
func (k Keeper) ReactorUpdatePlayerAllocation(ctx sdk.Context, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) (reactor types.Reactor) {

	validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.String())
	if !reactorBytesFound {
        return
	}
    reactor, _ = k.GetReactorByBytes(ctx, reactorBytes, false)

    if !reactor.Activated {
        return
    }


    delegation, _ := k.stakingKeeper.GetDelegation(ctx, playerAddress, validatorAddress)

    if validator.GetDelegatorShares().IsZero() {
        //
        return
    }

    delegationShare := ((delegation.Shares.Quo(validator.DelegatorShares)).Mul(math.LegacyNewDecFromInt(validator.Tokens))).RoundInt()


    // TODO this is all very wrong.
    allocation := types.CreateEmptyAllocation(types.ObjectType_reactor)
    allocation.SetPower(delegationShare.Uint64())
    k.AppendAllocation(ctx, allocation)

    // need to define a guild or reactor parameter for allowed allocation of contributions
    // need to define a guild or reactor parameter for minimum contributions before allocations are allowed


	// Update the connected Substations with the new details
	k.CascadeReactorAllocationFailure(ctx, reactor)


	return reactor
}


/* Update Reactor Details (Primarily In-Game Permissions/Ownership)
 *
 * Triggered during Staking Hooks:
 *   BeforeValidatorModified (Ugh, why isn't this AfterValidatorModified)
 *
 */
func (k Keeper) ReactorUpdateFromValidator(ctx sdk.Context, validatorAddress sdk.ValAddress) (reactor types.Reactor) {

	validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.String())
	if reactorBytesFound {
		reactor, _ = k.GetReactorByBytes(ctx, reactorBytes, false)
	}



	/* Sync Energy Levels
	 *
	 * May as well update power levels while we are here
	 */
	reactor.SetEnergy(validator)

	k.SetReactor(ctx, reactor)

    activated, _ := k.ReactorActivate(ctx, reactor, validator)
    if (activated) {
        reactor.SetActivated(true)
        k.SetReactor(ctx, reactor)
    }


	// Update the connected Substations with the new details
	k.CascadeReactorAllocationFailure(ctx, reactor)

	return reactor
}

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
func (k Keeper) GetReactorBytesFromValidator(ctx sdk.Context, validatorAddress string) (reactorBytes []byte, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorValidatorKey))

	reactorBytes = store.Get([]byte(validatorAddress))
	// Count doesn't exist: no element
	if reactorBytes == nil {
		return reactorBytes, false
	}

	return reactorBytes, true
}

// SetReactorValidatorBytes set the validator address index bytes
func (k Keeper) SetReactorValidatorBytes(ctx sdk.Context, id uint64, validatorAddress string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorValidatorKey))

	store.Set([]byte(validatorAddress), GetReactorIDBytes(id))
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


func (k Keeper) ReactorActivate(ctx sdk.Context, reactor types.Reactor, validator staking.Validator) (bool, error) {

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
        player.SetId(k.GetNextPlayerId(ctx))
        k.SetPlayer(ctx, player)
        k.SetPlayerIdForAddress(ctx, identity, player.Id)

        // TODO Add Related Address
    } else {
       player, _ = k.GetPlayer(ctx, playerId)
    }

    // Apply Ownership Permissions of the Reactor to the Player
    k.ReactorPermissionAdd(ctx, reactor.Id, player.Id, types.ReactorPermissionAll)

    // Now let's get the player some power
    if (player.SubstationId == 0) {
        var substation types.Substation
        substation = types.CreateEmptySubstation()
        substation.SetId(k.GetNextSubstationId(ctx))
        substation.SetOwner(player.Id)
        substation.SetCreator(identity)
        substation.SetPlayerConnectionAllocation(types.InitialReactorOwnerEnergy)
        k.SetSubstation(ctx, substation)

        k.SubstationPermissionAdd(ctx, substation.Id, player.Id, types.SubstationPermissionAll)


        var allocation types.Allocation
        allocation = types.CreateEmptyAllocation(types.ObjectType_reactor)
        allocation.SetController(identity)
        allocation.SetCreator(identity)
        allocation.SetPower(types.InitialReactorAllocation)
        allocation.SetSource(reactor.Id)

        // Connect Allocation to Substation
        allocation.Connect(substation.Id)
        _ = k.SubstationIncrementEnergy(ctx, substation.Id, allocation.Power)

        _ = k.AppendAllocation(ctx, allocation)

        // Connect Player to Substation
        k.SubstationIncrementConnectedPlayerLoad(ctx, substation.Id, 1)
        player.SetSubstation(substation.Id)
        k.SetPlayer(ctx, player)

    }


    return true, nil

}

// Iterate through the allocations, starting from oldest, and destroy them until power
// consumption matches output
func (k Keeper) CascadeReactorAllocationFailure(ctx sdk.Context, reactor types.Reactor) {
	allocations := k.GetAllReactorAllocations(ctx, reactor.Id)
	for _, allocation := range allocations {
		if reactor.Energy > k.ReactorGetLoad(ctx, reactor.Id) {
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

// ReactorGetEnergy returns the current energy production of the reactor
// Go to memory first, but then fall back to rebuilding from storage
func (k Keeper) ReactorGetEnergy(ctx sdk.Context, id uint64) (load uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.ReactorEnergyKey))

	bz := store.Get(GetReactorIDBytes(id))

	// Reactor Energy Not in Memory: no element
	if bz == nil {
		reactor, _ := k.GetReactor(ctx, id, false)
		load = reactor.Energy
		k.ReactorSetEnergy(ctx, id, load)

	} else {
		load = binary.BigEndian.Uint64(bz)
	}

	return load
}

// ReactorSetEnergy- Sets the in-memory representation of the reactors energy production
func (k Keeper) ReactorSetEnergy(ctx sdk.Context, id uint64, amount uint64) {
	store := prefix.NewStore(ctx.KVStore(k.memKey), types.KeyPrefix(types.ReactorEnergyKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, amount)

	store.Set(GetReactorIDBytes(id), bz)
	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: id, ObjectType: types.ObjectType_reactor})
}




// GetReactorPermissionIDBytes returns the byte representation of the reactor and player id pair
func GetReactorPermissionIDBytes(reactorId uint64, playerId uint64) []byte {
	reactorIdString  := strconv.FormatUint(reactorId, 10)
	playerIdString := strconv.FormatUint(playerId, 10)

	return []byte(reactorIdString + "-" + playerIdString)
}


func (k Keeper) ReactorGetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte) (types.ReactorPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorPermissionKey))

	bz := store.Get(permissionRecord)

	// Substation Capacity Not in Memory: no element
	if bz == nil {
		return types.ReactorPermissionless
	}

	load := types.ReactorPermission(binary.BigEndian.Uint16(bz))

	return load
}

func (k Keeper) ReactorSetPlayerPermissionsByBytes(ctx sdk.Context, permissionRecord []byte, permissions types.ReactorPermission) (types.ReactorPermission) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorPermissionKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint16(bz, uint16(permissions))

	store.Set(permissionRecord, bz)

	return permissions
}

func (k Keeper) ReactorPermissionClearAll(ctx sdk.Context, reactorId uint64, playerId uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorPermissionKey))
	store.Delete(GetReactorPermissionIDBytes(reactorId, playerId))
}

func (k Keeper) ReactorPermissionAdd(ctx sdk.Context, reactorId uint64, playerId uint64, flag types.ReactorPermission) types.ReactorPermission {
    permissionRecord := GetReactorPermissionIDBytes(reactorId, playerId)

    currentPermission := k.ReactorGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.ReactorSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission | flag)
	return newPermissions
}

func (k Keeper) ReactorPermissionRemove(ctx sdk.Context, reactorId uint64, playerId uint64, flag types.ReactorPermission) types.ReactorPermission {
    permissionRecord := GetReactorPermissionIDBytes(reactorId, playerId)

    currentPermission := k.ReactorGetPlayerPermissionsByBytes(ctx, permissionRecord)
    newPermissions := k.ReactorSetPlayerPermissionsByBytes(ctx, permissionRecord, currentPermission &^ flag)
	return newPermissions
}

func (k Keeper) ReactorPermissionHasAll(ctx sdk.Context, reactorId uint64, playerId uint64, flag types.ReactorPermission) bool {
    permissionRecord := GetReactorPermissionIDBytes(reactorId, playerId)

    currentPermission := k.ReactorGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag == flag
}

func (k Keeper) ReactorPermissionHasOneOf(ctx sdk.Context, reactorId uint64, playerId uint64, flag types.ReactorPermission) bool {
    permissionRecord := GetReactorPermissionIDBytes(reactorId, playerId)

    currentPermission := k.ReactorGetPlayerPermissionsByBytes(ctx, permissionRecord)

	return currentPermission&flag != 0
}
