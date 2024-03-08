package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//"strconv"
	//"cosmossdk.io/math"

)


func GetInfusionID(destinationId string, address string) (id string) {
    id = destinationId + "-" + address
    return
}

// AppendInfusion appends a infusion in the store
func (k Keeper) AppendInfusion(
	ctx sdk.Context,
	infusion types.Infusion,
) string {

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	appendedValue := k.cdc.MustMarshal(&infusion)
	infusionId := GetInfusionID(infusion.DestinationId, infusion.Address)
	store.Set([]byte(infusionId), appendedValue)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventInfusion{Infusion: &infusion})

	return infusionId
}

// SetInfusion set a specific infusion in the store
func (k Keeper) SetInfusion(ctx sdk.Context, infusion types.Infusion) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	b := k.cdc.MustMarshal(&infusion)
	infusionId := GetInfusionID(infusion.DestinationId, infusion.Address)
	store.Set([]byte(infusionId), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventInfusion{Infusion: &infusion})
}

// GetInfusion returns a infusion from its id
func (k Keeper) GetInfusion(ctx sdk.Context, destinationId string, address string) (val types.Infusion, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	infusionId := GetInfusionID(destinationId, address)
	b := store.Get([]byte(infusionId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) UpsertInfusion(ctx sdk.Context, destinationType types.ObjectType, destinationId string, address string, player types.Player, fuel uint64, commission sdk.Dec) (infusion types.Infusion, newInfusionFuel uint64, oldInfusionFuel uint64, newInfusionPower uint64, oldInfusionPower uint64, newCommissionPower uint64, oldCommissionPower uint64, newPlayerPower uint64, oldPlayerPower uint64, err error) {

    infusion, infusionFound := k.GetInfusion(ctx, destinationId, address)
    if (infusionFound) {
         newInfusionFuel, oldInfusionFuel, newInfusionPower, oldInfusionPower, newCommissionPower, oldCommissionPower, newPlayerPower, oldPlayerPower, err = infusion.SetFuelAndCommission(fuel, commission)
    } else {

        infusion = types.CreateNewInfusion(destinationType, destinationId, address, player.Id, fuel, commission)

        // Should already be the value, but let's be safe
        oldInfusionFuel = 0
        oldPlayerPower = 0
        oldCommissionPower = 0
        oldInfusionPower = 0

        newInfusionFuel = fuel
        newInfusionPower, newCommissionPower, newPlayerPower = infusion.GetPowerDistribution()
    }

    k.SetInfusion(ctx, infusion)

    // Update the Fuel record on the Destination
    if (oldInfusionFuel != newInfusionFuel) {
        k.SetGridAttributeDelta(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, destinationId), oldInfusionFuel, newInfusionFuel)
    }

    // Update the Commissioned Power on the Destination
    if (oldCommissionPower != newCommissionPower) {
        k.SetGridAttributeDelta(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, destinationId), oldCommissionPower, newCommissionPower)

        // Check for an automated allocation
        destinationAllocationId, destinationAutoResizeAllocationFound := k.GetAutoResizeAllocationBySource(ctx, destinationId)
        if (destinationAutoResizeAllocationFound) {
            k.AutoResizeAllocation(ctx, destinationAllocationId, destinationId, oldCommissionPower, newCommissionPower)
        } else {
            if (oldCommissionPower > newCommissionPower) {
                k.AppendGridCascadeQueue(ctx, destinationId)
            }
        }
    }

    // Update the Player's Power Capacity
    if (oldPlayerPower != newPlayerPower) {
        k.SetGridAttributeDelta(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id), oldPlayerPower, newPlayerPower)

        // Check for an automated allocation
        playerAllocationId, playerAutoResizeAllocationFound := k.GetAutoResizeAllocationBySource(ctx, player.Id)
        if (playerAutoResizeAllocationFound) {
            k.AutoResizeAllocation(ctx, playerAllocationId, player.Id, oldPlayerPower, newPlayerPower)
        } else {
            // This might be able to be an else from the above statement, but I need more coffee before committing
            if (oldPlayerPower > newPlayerPower) {
                k.AppendGridCascadeQueue(ctx, player.Id)
            }
        }
    }

    // need to write some events

     return
}

// RemoveInfusion removes a infusion from the store
func (k Keeper) RemoveInfusion(ctx sdk.Context, destinationId string, address string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	infusionId := GetInfusionID(destinationId, address)
	store.Delete([]byte(infusionId))

	_ = ctx.EventManager().EmitTypedEvent(&types.EventDelete{ ObjectId: infusionId})
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
func (k Keeper) GetAllReactorInfusions(ctx sdk.Context, reactorId string) (list []types.Infusion) {
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
func (k Keeper) GetAllStructInfusions(ctx sdk.Context, structId string) (list []types.Infusion) {
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

// GetAllInfusionsByDestination returns all infusion relating to a struct
func (k Keeper) GetAllInfusionsByDestination(ctx sdk.Context, objectId string) (list []types.Infusion) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.InfusionKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Infusion
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		if val.DestinationId == objectId {
			list = append(list, val)
		}
	}

	return
}



func (k Keeper) DestroyInfusion(ctx sdk.Context, infusion types.Infusion) {

    infusionPower, commissionPower, playerPower := infusion.GetPowerDistribution()

    // Quiet the go lords
    _ = infusionPower


    // update destination fuel
    k.SetGridAttributeDecrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, infusion.DestinationId), infusion.Fuel)

    // Update destination commission capacity
    k.SetGridAttributeDecrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, infusion.DestinationId), commissionPower)

    // Check for an automated allocation on the destination
    destinationAllocationId, destinationAutoResizeAllocationFound := k.GetAutoResizeAllocationBySource(ctx, infusion.DestinationId)
    if (destinationAutoResizeAllocationFound) {
        k.AutoResizeAllocation(ctx, destinationAllocationId, infusion.DestinationId, commissionPower, 0)
    } else {
        k.AppendGridCascadeQueue(ctx, infusion.DestinationId)
    }


    // update player capacity
    k.SetGridAttributeDecrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, infusion.PlayerId), playerPower)

    // Check for an automated allocation on the player
    playerAllocationId, playerAutoResizeAllocationFound := k.GetAutoResizeAllocationBySource(ctx, infusion.PlayerId)
    if (playerAutoResizeAllocationFound) {
        k.AutoResizeAllocation(ctx, playerAllocationId, infusion.PlayerId, playerPower, 0)
    } else {
        k.AppendGridCascadeQueue(ctx, infusion.PlayerId)
    }

    // Remove the Infusion record from the store
	k.RemoveInfusion(ctx, infusion.DestinationId, infusion.Address)

}

// TODO could likely be done far more efficiently
// Currently makes separate writes for each update
func (k Keeper) DestroyAllInfusions(ctx sdk.Context, infusions []types.Infusion) {
     for _, infusion := range infusions {
        k.DestroyInfusion(ctx, infusion)
     }
}
