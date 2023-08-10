package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	"strconv"
)


// AppendInfusion appends a infusion in the store
func (k Keeper) AppendInfusion(
	ctx sdk.Context,
	infusion types.Infusion,
) string {

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	appendedValue := k.cdc.MustMarshal(&infusion)
	infusionId := GetInfusionId(infusion.DestinationType, infusion.DestinationId, infusion.Address)
	store.Set(GetInfusionIDBytes(infusionId), appendedValue)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectName: infusionId, ObjectType: types.ObjectType_infusion})

	return infusionId
}

// SetInfusion set a specific infusion in the store
func (k Keeper) SetInfusion(ctx sdk.Context, infusion types.Infusion) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	b := k.cdc.MustMarshal(&infusion)
	infusionId := GetInfusionId(infusion.DestinationType, infusion.DestinationId, infusion.Address)
	store.Set(GetInfusionIDBytes(infusionId), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventCacheInvalidation{ObjectName: infusionId, ObjectType: types.ObjectType_infusion})
}

// GetInfusion returns a infusion from its id
func (k Keeper) GetInfusion(ctx sdk.Context, destinationType types.ObjectType, destinationId uint64, address string) (val types.Infusion, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	infusionId := GetInfusionId(destinationType, destinationId, address)
	b := store.Get(GetInfusionIDBytes(infusionId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) UpsertInfusion(ctx sdk.Context, destinationType types.ObjectType, destinationId uint64, address string, fuel uint64, automatedAllocation bool, delegateTaxOnAllocations sdk.Dec) (infusion types.Infusion, allocation types.Allocation, withDynamicAllocation bool){

    infusion, infusionFound := k.GetInfusion(ctx, destinationType, destinationId, address)
    if (infusionFound) {
        infusion.SetFuel(fuel)
    } else {
        infusion = types.CreateNewInfusion(destinationType, destinationId, address, fuel)
    }


    if (automatedAllocation) {
        allocation = types.CreateEmptyAllocation(destinationType)

        if (infusion.LinkedAllocation > 0) {
            allocation.SetId(infusion.LinkedAllocation)
        } else {
            allocation.SetId(0)
        }

        allocation.SetCreator(address)
        allocation.SetController(address)
        allocation.SetSource(destinationId)

        // TODO actual fuel to energy ratio
        // apply tax
        allocation.SetPower(fuel * 10)

        allocation.SetLinkedInfusion(address)

        withDynamicAllocation = true
        appendedAllocation := k.UpsertAllocation(ctx, allocation)
        infusion.SetLinkedAllocation(appendedAllocation.Id)
     } else {
         withDynamicAllocation = false
     }

     k.SetInfusion(ctx, infusion)

    switch infusion.DestinationType {
        case types.ObjectType_reactor:
            k.ReactorSetEnergy(ctx, destinationId, k.ReactorRebuildEnergy(ctx, destinationId))
    }


     return
}

// RemoveInfusion removes a infusion from the store
func (k Keeper) RemoveInfusion(ctx sdk.Context, destinationType types.ObjectType, destinationId uint64, address string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	infusionId := GetInfusionId(destinationType, destinationId, address)
	store.Delete(GetInfusionIDBytes(infusionId))
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

func GetInfusionId(destinationType types.ObjectType, destinationId uint64, address string) (id string) {
    destinationIdString := strconv.FormatUint(destinationId , 10)
    id = destinationType.String() + "-" + destinationIdString + "-" + address

    return
}

// GetInfusionIDBytes returns the byte representation of the ID
func GetInfusionIDBytes(id string) []byte {
	return []byte(id)
}

// GetInfusionIDFromBytes returns ID in uint64 format from a byte array
func GetInfusionIDFromBytes(bz []byte) string {
	return string(bz)
}

func (k Keeper) InfusionDestroy(ctx sdk.Context, infusion types.Infusion) {

	// Figure out what the infusion source is and update the energy load
	switch infusion.DestinationType {
	case types.ObjectType_reactor:
		// Decrease Reactor Energy
		//k.ReactorDecrementEnergy(ctx, infusion.DestinationId, infusion.Fuel)
	}

	k.RemoveInfusion(ctx, infusion.DestinationType, infusion.DestinationId, infusion.Address)

}
