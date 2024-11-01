package keeper

import (
	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
    storetypes "cosmossdk.io/store/types"

	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//"strconv"
	"cosmossdk.io/math"
	"strings"

)


func InfusionKeyPrefix(destinationId string) []byte {
	return []byte(types.InfusionKey + destinationId + "/")
}

func GetInfusionID(address string) ([]byte) {
    return []byte(address)
}

// AppendInfusion appends a infusion in the store
func (k Keeper) AppendInfusion(
	ctx context.Context,
	infusion types.Infusion,
) error {

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), InfusionKeyPrefix(infusion.DestinationId))
	appendedValue := k.cdc.MustMarshal(&infusion)
	store.Set(GetInfusionID(infusion.Address), appendedValue)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventInfusion{Infusion: &infusion})

	return nil
}

// SetInfusion set a specific infusion in the store
func (k Keeper) SetInfusion(ctx context.Context, infusion types.Infusion) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), InfusionKeyPrefix(infusion.DestinationId))

	b := k.cdc.MustMarshal(&infusion)
	store.Set(GetInfusionID(infusion.Address), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventInfusion{Infusion: &infusion})
}

// GetInfusion returns a infusion from its id
func (k Keeper) GetInfusion(ctx context.Context, destinationId string, address string) (val types.Infusion, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), InfusionKeyPrefix(destinationId))

	b := store.Get(GetInfusionID(address))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// GetInfusion returns a infusion from its id (destinationId-address)
func (k Keeper) GetInfusionByID(ctx context.Context, infusionId string) (val types.Infusion, found bool) {
    infusionIdSplit := strings.Split(infusionId, "-")
	return k.GetInfusion(ctx, infusionIdSplit[0], infusionIdSplit[1])
}


func (k Keeper) UpsertInfusion(ctx context.Context, destinationType types.ObjectType, destinationId string, address string, player types.Player, fuel uint64, commission math.LegacyDec, ratio uint64) (infusion types.Infusion, newInfusionFuel uint64, oldInfusionFuel uint64, newInfusionPower uint64, oldInfusionPower uint64, newCommissionPower uint64, oldCommissionPower uint64, newPlayerPower uint64, oldPlayerPower uint64, err error) {

    infusion, infusionFound := k.GetInfusion(ctx, destinationId, address)
    if (infusionFound) {
         newInfusionFuel, oldInfusionFuel, newInfusionPower, oldInfusionPower, newCommissionPower, oldCommissionPower, newPlayerPower, oldPlayerPower, _, _, err = infusion.SetFuelAndCommission(fuel, commission)
    } else {

        infusion = types.CreateNewInfusion(destinationType, destinationId, address, player.Id, fuel, commission, ratio)

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

     return
}

// RemoveInfusion removes a infusion from the store
func (k Keeper) RemoveInfusion(ctx context.Context, destinationId string, address string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), InfusionKeyPrefix(destinationId))

	store.Delete(GetInfusionID(address))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	infusionId := destinationId + "-" + address
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ ObjectId: infusionId})
}

// GetAllInfusion returns all infusion
func (k Keeper) GetAllInfusion(ctx context.Context) (list []types.Infusion) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.InfusionKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Infusion
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}


// GetAllReactorInfusions returns all infusion relating to a reactor
func (k Keeper) GetAllReactorInfusions(ctx context.Context, reactorId string) (list []types.Infusion) {
	return k.GetAllInfusionsByDestination(ctx, reactorId)
}

// GetAllReactorInfusions returns all infusion relating to a struct
func (k Keeper) GetAllStructInfusions(ctx context.Context, structId string) (list []types.Infusion) {
	return k.GetAllInfusionsByDestination(ctx, structId)
}

// GetAllInfusionsByDestination returns all infusion relating to a struct
func (k Keeper) GetAllInfusionsByDestination(ctx context.Context, objectId string) (list []types.Infusion) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), InfusionKeyPrefix(objectId))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Infusion
		k.cdc.MustUnmarshal(iterator.Value(), &val)
	}

	return
}



func (k Keeper) DestroyInfusion(ctx context.Context, infusion types.Infusion) {

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
func (k Keeper) DestroyAllInfusions(ctx context.Context, infusions []types.Infusion) {
     for _, infusion := range infusions {
        k.DestroyInfusion(ctx, infusion)
     }
}
