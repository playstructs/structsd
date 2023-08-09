package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// GetInfusionCount get the total number of infusion
func (k Keeper) GetInfusionCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.InfusionCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0 {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetInfusionCount set the total number of infusion
func (k Keeper) SetInfusionCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.InfusionCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendInfusion appends a infusion in the store with a new id and update the count
func (k Keeper) AppendInfusion(
	ctx sdk.Context,
	infusion types.Infusion,
) uint64 {
	// Create the infusion
	count := k.GetInfusionCount(ctx)

	// Set the ID of the appended value
	infusion.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	appendedValue := k.cdc.MustMarshal(&infusion)
	store.Set(GetInfusionIDBytes(infusion.Id), appendedValue)

	// Update infusion count
	k.SetInfusionCount(ctx, count+1)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: infusion.Id, ObjectType: types.ObjectType_infusion})

	return count
}

// SetInfusion set a specific infusion in the store
func (k Keeper) SetInfusion(ctx sdk.Context, infusion types.Infusion) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	b := k.cdc.MustMarshal(&infusion)
	store.Set(GetInfusionIDBytes(infusion.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectId: infusion.Id, ObjectType: types.ObjectType_infusion})
}

// GetInfusion returns a infusion from its id
func (k Keeper) GetInfusion(ctx sdk.Context, id uint64) (val types.Infusion, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	b := store.Get(GetInfusionIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) UpsertInfusion(ctx sdk.Context, destinationType types.ObjectType, destinationId uint64, address string, fuel uint64, automatedAllocation bool, delegateTaxOnAllocations sdk.Dec) (infusion types.Infusion){

    id := k.GetInfusionId(destinationType, destinationId, playerAddress)
    infusion, found := k.GetInfusion(ctx, id)
    if (found) {
        if (automatedAllocations) {
            if (infusion.Fuel > fuel) {
                // TODO decrement allocation

                k.ReactorDecrementEnergy()

            } else {
                // TODO increase allocation

            }
            infusion.SetFuel(fuel)
        }
    } else {
        infusion = types.CreateNewInfusion(destinationType, destinationId, address, fuel)

        if (automatedAllocations) {

            allocation := types.CreateEmptyAllocation(destinationType)
            allocation.SetCreator(address)
            allocation.SetController(address)

            // TODO actual fuel to energy ratio
            // apply tax
            allocation.SetPower(fuel)

            allocation.SetLinkedInfusion(address)
            allocationId := k.AppendAllocation(ctx, allocation)




        }


    }


}

// RemoveInfusion removes a infusion from the store
func (k Keeper) RemoveInfusion(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	store.Delete(GetInfusionIDBytes(id))
}

// GetAllInfusion returns all infusion
func (k Keeper) GetAllInfusion(ctx sdk.Context) (list []types.Infusion) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Infusion
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}


// GetAllReactorInfusions returns all infusion relating to a reactor
func (k Keeper) GetAllReactorInfusions(ctx sdk.Context, reactorId uint64) (list []types.Infusion) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Infusion
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		if val.DestinationType == types.ObjectType_reactor && val.DestinationId == reactorId {
			list = append(list, val)
		}
	}

	return
}

// GetInfusionIDBytes returns the byte representation of the ID
func GetInfusionIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetInfusionIDFromBytes returns ID in uint64 format from a byte array
func GetInfusionIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) InfusionDestroy(ctx sdk.Context, infusion types.Infusion) {

	// Figure out what the infusion source is and update the energy load
	switch infusion.DestinationType {
	case types.ObjectType_reactor:
		// Decrease Reactor Energy
		//k.ReactorDecrementEnergy(ctx, infusion.DestinationId, infusion.Fuel)
	}

	k.RemoveInfusion(ctx, infusion.Id)

}
