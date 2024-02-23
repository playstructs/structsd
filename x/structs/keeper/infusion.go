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
                        newInfusionPower uint64,
                        oldInfusionPower uint64,
                        newCommissionPower uint64,
                        oldCommissionPower uint64,
                        newPlayerPower uint64,
                        oldPlayerPower uint64,
                        err error
                    ){

    infusion, infusionFound := k.GetInfusion(ctx, destinationType, destinationId, player.Address)
    if (infusionFound) {
         newInfusionFuel, oldInfusionFuel, newInfusionPower, oldInfusionPower, newCommissionPower, oldCommissionPower, newPlayerPower, oldPlayerPower, err = infusion.SetFuelAndCommission(fuel, commission)
    } else {
        infusion = types.CreateNewInfusion(destinationType, destinationId, player.Address, fuel, commission)

        // Should already be the value, but let's be safe
        oldInfusionFuel = 0
        oldPlayerPower = 0
        oldCommissionPower = 0
        oldInfusionPower = 0

        newInfusionFuel = fuel
        newInfusionPower, newCommissionPower, newPlayerPower = infusion.getPowerDistribution()
    }

    k.SetInfusion(ctx, infusion)

    destinationIdBytes  := GetObjectIDBytes(destinationType, destinationId)
    playerIdBytes       := GetObjectIDBytes(types.ObjectType_player, player.Id)

    // Update the Fuel record on the Destination
    if (oldInfusionFuel != newInfusionFuel) {
        k.SetGridAttributeDelta(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_fuel, destinationIdBytes), oldInfusionFuel, newInfusionFuel)
    }

    // Update the Commissioned Power on the Destination
    if (oldCommissionPower != newCommissionPower) {
        k.SetGridAttributeDelta(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_capacity, destinationIdBytes), oldCommissionPower, newCommissionPower)

        // Check for an automated allocation
        destinationAllocationId, destinationAutoResizeAllocationFound := k.GetAutoResizeAllocationBySource(destinationIdBytes)
        if (destinationAutoResizeAllocationFound) {
            k.AutoResizeAllocation(ctx, destinationAllocationId, destinationIdBytes, oldCommissionPower, newCommissionPower)
        } else {
            if (oldCommissionPower > newCommissionPower) {
                k.AppendGridCascadeQueue(ctx, destinationIdBytes)
            }
        }
    }

    // Update the Player's Power Capacity
    if (oldPlayerPower != newPlayerPower) {
        k.SetGridAttributeDelta(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_capacity, playerIdBytes), oldPlayerPower, newPlayerPower)

        // Check for an automated allocation
        playerAllocationId, playerAutoResizeAllocationFound := k.GetAutoResizeAllocationBySource(playerIdBytes)
        if (playerAutoResizeAllocationFound) {
            k.AutoResizeAllocation(ctx, playerAllocationId, playerIdBytes, oldPlayerPower, newPlayerPower)
        } else {
            // This might be able to be an else from the above statement, but I need more coffee before committing
            if (oldPlayerPower > newPlayerPower) {
                k.AppendGridCascadeQueue(ctx, playerIdBytes)
            }
        }
    }

    // need to write some events

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

func (k Keeper) DestroyInfusion(ctx sdk.Context, infusion types.Infusion) {

    infusionPower, commissionPower, playerPower := infusion.getPowerDistribution()

    // Quiet the go lords
    _ = infusionPower

    destinationIdBytes  := GetObjectIDBytes(infusion.DestinationType, infusion.DestinationId)
    playerIdBytes       := GetObjectIDBytes(types.ObjectType_player, infusion.PlayerId)

    // update destination fuel
    k.SetGridAttributeDecrement(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_fuel, destinationIdBytes), infusion.Fuel)

    // Update destination commission capacity
    k.SetGridAttributeDecrement(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_capacity, destinationIdBytes), commissionPower)

    // Check for an automated allocation on the destination
    destinationAllocationId, destinationAutoResizeAllocationFound := k.GetAutoResizeAllocationBySource(destinationIdBytes)
    if (destinationAutoResizeAllocationFound) {
        k.AutoResizeAllocation(ctx, destinationAllocationId, destinationIdBytes, commissionPower, 0)
    } else {
        k.AppendGridCascadeQueue(ctx, destinationIdBytes)
    }


    // update player capacity
    k.SetGridAttributeDecrement(ctx, GetGridAttributeIDBytesByObjectId(types.GridAttributeType_capacity, playerIdBytes), playerPower)

    // Check for an automated allocation on the player
    playerAllocationId, playerAutoResizeAllocationFound := k.GetAutoResizeAllocationBySource(playerIdBytes)
    if (playerAutoResizeAllocationFound) {
        k.AutoResizeAllocation(ctx, playerAllocationId, playerIdBytes, playerPower, 0)
    } else {
        k.AppendGridCascadeQueue(ctx, playerIdBytes)
    }

    // Remove the Infusion record from the store
	k.RemoveInfusion(ctx, infusion.DestinationType, infusion.DestinationId, infusion.Address)

}
