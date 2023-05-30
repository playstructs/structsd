package keeper

import (
	"encoding/binary"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

)



/* Setup Reactor (when a validator is created)
 *
 * Triggered during Staking Hooks:
 *   AfterValidatorCreated
 */
func (k Keeper) ReactorInitialize(ctx sdk.Context, validatorAddress sdk.ValAddress) (reactor types.Reactor) {

    validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)

    /* Does this Reactor exist? */
    reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress)
    if (reactorBytesFound) {
         reactor, _  = k.GetReactorByBytes(ctx, reactorBytes)
    } else {
        /* Build the initial Reactor object */
        reactor = types.CreateEmptyReactor()
        reactor.SetValidator(validator)
    }

    /* Bring the Reactor ONLINE
     *
     * It's possible we'll want this to start in a different
     * but it should be fine for now.
    */
	_ = reactor.SetStatusOnline()

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
	_ = reactor.SetEnergy(validator)



	k.SetReactor(ctx, reactor)

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
    reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress)
    if (reactorBytesFound) {
         reactor, _  = k.GetReactorByBytes(ctx, reactorBytes)
    } else {
        /* Build the initial Reactor object */
        reactor = types.CreateEmptyReactor()
        reactor.SetValidator(validator)
    }

    /* Sync Energy Levels
     *
     * Set the initial power level of the Reactor based on the
     * tokens staked to the validator
     */
	_ = reactor.SetEnergy(validator)


    /* Check on the Status of the Reactor
    */
    if (reactor.Load > reactor.Power) {
        _ = reactor.SetStatusOverload()
    } else {
        _ = reactor.SetStatusOnline()
    }

    /* TODO: Permissions
     *
     * Create permissions based on the details field.
     *  Link to Faction
     *  Link to Player
     */


	k.SetReactor(ctx, reactor)

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
    reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress)
    if (reactorBytesFound) {
         reactor, _  = k.GetReactorByBytes(ctx, reactorBytes)
    } else {
        /* Build the initial Reactor object */
        reactor = types.CreateEmptyReactor()
        reactor.SetValidator(validator)
    }

    /* Sync Energy Levels
     *
     * May as well update power levels while we are here
     */
	_ = reactor.SetEnergy(validator)


    /* Check on the Status of the Reactor
    */
    if (reactor.Load > reactor.Power) {
        _ = reactor.SetStatusOverload()
    } else {
        _ = reactor.SetStatusOnline()
    }


	k.SetReactor(ctx, reactor)

	return reactor
}



// GetReactorCount get the total number of reactor
func (k Keeper) GetReactorCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.ReactorCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
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
func (k Keeper) GetReactorBytesFromValidator(ctx sdk.Context, validatorAddress sdk.ValAddress) (reactorBytes []byte, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorValidatorKey))

    reactorBytes =  store.Get(validatorAddress)
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

    // Add a record to the Power index
    k.SetReactorPower(ctx, reactor.Id, reactor.Power)

    // Add a record to the Load index
    k.SetReactorLoad(ctx, reactor.Id, reactor.Load)

	// Update reactor count
	k.SetReactorCount(ctx, count+1)

	return count
}

// SetReactor set a specific reactor in the store
func (k Keeper) SetReactor(ctx sdk.Context, reactor types.Reactor) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	b := k.cdc.MustMarshal(&reactor)
	store.Set(GetReactorIDBytes(reactor.Id), b)
}

// GetReactor returns a reactor from its id
func (k Keeper) GetReactor(ctx sdk.Context, id uint64) (val types.Reactor, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	b := store.Get(GetReactorIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// GetReactor returns a reactor from its id
func (k Keeper) GetReactorByBytes(ctx sdk.Context, id []byte) (val types.Reactor, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	b := store.Get(id)
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}


// RemoveReactor removes a reactor from the store
func (k Keeper) RemoveReactor(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	store.Delete(GetReactorIDBytes(id))
}

// GetAllReactor returns all reactor
func (k Keeper) GetAllReactor(ctx sdk.Context) (list []types.Reactor) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Reactor
		k.cdc.MustUnmarshal(iterator.Value(), &val)
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


// GetReactorPower returns the current power being generated by a Reactor
func (k Keeper) GetReactorPower(ctx sdk.Context, id uint64) uint64 {
    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorPowerKey))
	power := store.Get(GetReactorIDBytes(id))

	// Count doesn't exist: no element
	if power == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(power)
}

// SetReactorPower updates the cached state of available power in a reactor
func (k Keeper) SetReactorPower(ctx sdk.Context, id uint64, power uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorPowerKey))
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, power)
	store.Set(GetReactorIDBytes(id), bz)
}


// GetReactorLoad returns the current power being allocated by a Reactor
func (k Keeper) GetReactorLoad(ctx sdk.Context, id uint64) uint64 {
    store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorLoadKey))
	load := store.Get(GetReactorIDBytes(id))

	// Count doesn't exist: no element
	if load == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(load)
}

// SetReactorLoad updates the cached state of allocated power in a reactor
func (k Keeper) SetReactorLoad(ctx sdk.Context, id uint64, load uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorLoadKey))
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, load)
	store.Set(GetReactorIDBytes(id), bz)
}

//Add Reactor Substation
func (k Keeper) AddReactorSubstation(ctx sdk.Context, reactorId uint64, substationId uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorSubstationsKey))

    var reactorSubstations types.ReactorSubstations
    reactorSubstationBinary := store.Get(GetReactorIDBytes(reactorId))
    if reactorSubstationBinary == nil {

        reactorSubstations = types.ReactorSubstations{SubstationId:[]uint64{substationId}}
    } else {
        k.cdc.MustUnmarshal(reactorSubstationBinary, &reactorSubstations)

        var found bool = false;
        for _, iteratedSubstationId := range reactorSubstations.SubstationId {
            if (iteratedSubstationId == substationId) {
                found = true;
            }
        }
        if (!found){
            reactorSubstations.SubstationId = append(reactorSubstations.SubstationId, substationId)
            b := k.cdc.MustMarshal(&reactorSubstations)
            store.Set(GetReactorIDBytes(reactorId), b)
        }
    }
}


//Remove Reactor Substation
func (k Keeper) RemoveReactorSubstation(ctx sdk.Context, reactorId uint64, substationId uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorSubstationsKey))

    var reactorSubstations types.ReactorSubstations
    reactorSubstationBinary := store.Get(GetReactorIDBytes(reactorId))
    if reactorSubstationBinary == nil {
        return
    } else {
        k.cdc.MustUnmarshal(reactorSubstationBinary, &reactorSubstations)

        var newList []uint64;
        var found bool = false;
        for _, iteratedSubstationId := range reactorSubstations.SubstationId {
            if (iteratedSubstationId == substationId) {
                found = true;
            } else {
                newList = append(newList, iteratedSubstationId)
            }
        }
        if (found){
            reactorSubstations.SubstationId = newList
            b := k.cdc.MustMarshal(&reactorSubstations)
            store.Set(GetReactorIDBytes(reactorId), b)
        }
    }
}

// GetReactorSubstations
func (k Keeper) GetReactorSubstations(ctx sdk.Context, reactorId uint64) (reactorSubstations types.ReactorSubstations) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReactorSubstationsKey))
    reactorSubstationBinary := store.Get(GetReactorIDBytes(reactorId))
    if reactorSubstationBinary == nil {
        return types.ReactorSubstations{ SubstationId: []uint64{} }
    } else {
        k.cdc.MustUnmarshal(reactorSubstationBinary, &reactorSubstations)
        return reactorSubstations
    }
}


// Call all the relevant Substations to update their status
func (k Keeper) CascadeReactorStatus(ctx sdk.Context, id uint64, reactor types.Reactor) {
    if (reactor.PreviousStatus != reactor.Status){
        reactorSubstations := k.GetReactorSubstations(ctx, reactor.Id)
        for _, substationId := range reactorSubstations.SubstationId {
            k.UpdateSubstationByReactorCascade(ctx, substationId, reactor)
        }
    }
}

