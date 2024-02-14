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

	_ = ctx.EventManager().EmitTypedEvent(&types.EventInfusion{Infusion: &infusion})

	return infusionId
}

// SetInfusion set a specific infusion in the store
func (k Keeper) SetInfusion(ctx sdk.Context, infusion types.Infusion) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	b := k.cdc.MustMarshal(&infusion)
	infusionId := GetInfusionId(infusion.DestinationType, infusion.DestinationId, infusion.Address)
	store.Set(GetInfusionIDBytes(infusionId), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventInfusion{Infusion: &infusion})
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

func (k Keeper) UpsertInfusion(ctx sdk.Context, destinationType types.ObjectType, destinationId uint64, player types.Player, fuel uint64, commission sdk.Dec)
                    (
                        infusion types.Infusion,
                        newInfusionFuel uint64,
                        oldInfusionFuel uint64,
                        newInfusionEnergy uint64,
                        oldInfusionEnergy uint64,
                        newCommissionEnergy uint64,
                        oldCommissionEnergy uint64,
                        newPlayerEnergy uint64,
                        oldPlayerEnergy uint64,
                        err error
                    ){

    infusion, infusionFound := k.GetInfusion(ctx, destinationType, destinationId, player.Address)
    if (infusionFound) {
         newInfusionFuel, oldInfusionFuel, newInfusionEnergy, oldInfusionEnergy, newCommissionEnergy, oldCommissionEnergy, newPlayerEnergy, oldPlayerEnergy, err = infusion.SetFuelAndCommission(fuel, commission)
    } else {
        infusion = types.CreateNewInfusion(destinationType, destinationId, player.Address, fuel, commission)

        // Should already be the value, but let's be safe
        oldInfusionFuel = 0
        oldPlayerEnergy = 0
        oldCommissionEnergy = 0
        oldInfusionEnergy = 0

        newInfusionFuel = fuel
        newInfusionEnergy, newCommissionEnergy, newPlayerEnergy = infusion.getEnergyDistribution()
    }


    // TODO HERE
    BREAKING COMMMENT


    // TODO move into case
    // Need to Set Destination (case below)
    if (oldCommissionEnergy != newCommissionEnergy) {
        currentReactorEnergy    := k.ReactorGetEnergy(ctx, destinationId)
        newReactorEnergy        := currentReactorEnergy - oldCommissionEnergy
        newReactorEnergy         = newReactorEnergy     + newCommissionEnergy



        if (oldCommissionEnergy > newCommissionEnergy) {
            // TODO will currently fail, no reactor
            k.CascadeReactorAllocationFailure(ctx, reactor)
        }
    }

    // Need to Set player power

    // need to update automated allocations on destination
    // need to update automated allocations on player

    // need to cascade failures to both possibly

    // need to write some events

     k.SetInfusion(ctx, infusion)

    switch infusion.DestinationType {
        case types.ObjectType_reactor:
            k.ReactorAlterEnergy(ctx, destinationId, newEnergy)
            k.ReactorAlterEnergy(ctx, destinationId, newEnergy)
        case types.ObjectType_struct:
            k.StructAlterFuel(ctx, destinationId, newFuel)
            k.StructAlterEnergy(ctx, destinationId, newEnergy)



    }


     return
}

// RemoveInfusion removes a infusion from the store
func (k Keeper) RemoveInfusion(ctx sdk.Context, destinationType types.ObjectType, destinationId uint64, address string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	infusionId := GetInfusionId(destinationType, destinationId, address)
	store.Delete(GetInfusionIDBytes(infusionId))

	_ = ctx.EventManager().EmitTypedEvent(&types.EventInfusionDelete{DestinationType: destinationType, DestinationId: destinationId, Address: address})
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

// GetAllReactorInfusions returns all infusion relating to a struct
func (k Keeper) GetAllStructInfusions(ctx sdk.Context, structId uint64) (list []types.Infusion) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Infusion
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		if val.DestinationType == types.ObjectType_struct && val.DestinationId == structId {
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
        case types.ObjectType_struct:
            newFuel, newEnergy := k.StructRebuildInfusions(ctx, infusion.DestinationId)
            k.StructSetFuel(ctx, infusion.DestinationId, newFuel)
            k.StructSetEnergy(ctx, infusion.DestinationId, newEnergy)
	}

}
