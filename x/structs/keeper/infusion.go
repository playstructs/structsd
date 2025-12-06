package keeper

import (
	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
    storetypes "cosmossdk.io/store/types"

	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

    "encoding/binary"
	//"strconv"
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
	if len(infusionIdSplit) != 2 {
	    return types.Infusion{}, false
	}
	return k.GetInfusion(ctx, infusionIdSplit[0], infusionIdSplit[1])
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


func (k Keeper) GetInfusionDestructionQueue(ctx context.Context, clear bool) (queue []string) {
	infusionDestructionQueueStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.InfusionDestructionQueue))
	iterator := storetypes.KVStorePrefixIterator(infusionDestructionQueueStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		queue = append(queue, string(iterator.Key()))
		if clear {
		    infusionDestructionQueueStore.Delete(iterator.Key())
		}
	}

    return
}


func (k Keeper) AppendInfusionDestructionQueue(ctx context.Context, infusionId string) (err error) {
    infusionDestructionQueueStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.InfusionDestructionQueue))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, 1)

	infusionDestructionQueueStore.Set([]byte(infusionId), bz)

    k.logger.Info("Infusion Destruction Queue (Add)", "queueId", infusionId)

	return err
}


func (k Keeper) ProcessInfusionDestructionQueue(ctx context.Context) {

    for {
        // Get Queue (and clear it in the process)
        infusionDestructionQueue := k.GetInfusionDestructionQueue(ctx, true)

        if (len(infusionDestructionQueue) == 0) {
            break
        }

        // For each Queue Item
        for _, objectId := range infusionDestructionQueue {
            infusion, infusionFound := k.GetInfusionByID(ctx, objectId)
            if infusionFound {
                if infusion.Power == 0 && infusion.Defusing == 0 {
                    k.DestroyInfusion(ctx, infusion)
                }
            }
        }
    }
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
