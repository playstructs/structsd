package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	"strconv"
	"cosmossdk.io/math"
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

func (k Keeper) UpsertInfusion(ctx sdk.Context, destinationType types.ObjectType, destinationId uint64, address string, fuel uint64, automatedAllocation bool, delegateTaxOnAllocations sdk.Dec) (infusion types.Infusion, sourceAllocation types.Allocation, withDynamicSourceAllocation bool, playerAllocation types.Allocation, withDynamicPlayerAllocation bool){

    infusion, infusionFound := k.GetInfusion(ctx, destinationType, destinationId, address)
    if (infusionFound) {
        infusion.SetFuel(fuel)
    } else {
        infusion = types.CreateNewInfusion(destinationType, destinationId, address, fuel)
    }


    if (automatedAllocation) {

        energyToPortion := infusion.Energy
        if (delegateTaxOnAllocations.GT(math.LegacyZeroDec())) {
           withDynamicSourceAllocation = true

           sourceAllocation = types.CreateEmptyAllocation(destinationType)

            if (infusion.LinkedSourceAllocationId > 0) {
                sourceAllocation.SetId(infusion.LinkedSourceAllocationId)
            } else {
                sourceAllocation.SetId(0)
            }

            sourceAllocation.SetCreator(address)
            sourceAllocation.SetSource(destinationId)

            sourcePortion := delegateTaxOnAllocations.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(energyToPortion)))
            sourceAllocation.SetPower(sourcePortion.RoundInt().Uint64())
            energyToPortion = energyToPortion - sourceAllocation.Power

            sourceAllocation.SetLinkedInfusion(address)

            withDynamicPlayerAllocation = true
            sourceAllocation = k.UpsertAllocation(ctx, sourceAllocation)

            infusion.SetLinkedSourceAllocation(sourceAllocation.Id)
        } else {
            withDynamicSourceAllocation = false
        }

        playerAllocation = types.CreateEmptyAllocation(destinationType)

        if (infusion.LinkedPlayerAllocationId > 0) {
            playerAllocation.SetId(infusion.LinkedPlayerAllocationId)
        } else {
            playerAllocation.SetId(0)
        }

        playerAllocation.SetCreator(address)
        playerAllocation.SetController(address)
        playerAllocation.SetSource(destinationId)

        // TODO actual fuel to energy ratio
        // apply tax
        playerAllocation.SetPower(energyToPortion)

        playerAllocation.SetLinkedInfusion(address)

        withDynamicPlayerAllocation = true
        playerAllocation = k.UpsertAllocation(ctx, playerAllocation)
        infusion.SetLinkedPlayerAllocation(playerAllocation.Id)
     } else {
         withDynamicSourceAllocation = false
         withDynamicPlayerAllocation = false
     }

     k.SetInfusion(ctx, infusion)

    switch infusion.DestinationType {
        case types.ObjectType_reactor:
            newFuel, newEnergy := k.ReactorRebuildInfusions(ctx, destinationId)
            k.ReactorSetFuel(ctx, destinationId, newFuel)
            k.ReactorSetEnergy(ctx, destinationId, newEnergy)


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

    if (infusion.LinkedPlayerAllocationId > 0) {
        playerAllocation, _ := k.GetAllocation(ctx, infusion.LinkedPlayerAllocationId)
        k.AllocationDestroy(ctx, playerAllocation)
    }

    if (infusion.LinkedSourceAllocationId > 0) {
        sourceAllocation, _ := k.GetAllocation(ctx, infusion.LinkedSourceAllocationId)
        k.AllocationDestroy(ctx, sourceAllocation)
    }

	k.RemoveInfusion(ctx, infusion.DestinationType, infusion.DestinationId, infusion.Address)

	// Figure out what the infusion source is and update the energy load
	switch infusion.DestinationType {
	    case types.ObjectType_reactor:
            newFuel, newEnergy := k.ReactorRebuildInfusions(ctx, infusion.DestinationId)
            k.ReactorSetFuel(ctx, infusion.DestinationId, newFuel)
            k.ReactorSetEnergy(ctx, infusion.DestinationId, newEnergy)
	}

}
