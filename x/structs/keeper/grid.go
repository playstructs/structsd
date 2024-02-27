package keeper

import (
	"encoding/binary"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"fmt"
)


// GetObjectID returns the string representation of the ID, based on ObjectType
// This is the unified objectId model across the system
func GetObjectID(objectType types.ObjectType, objectId uint64) string {
    id := fmt.Sprintf("%d-%d", objectType, objectId)
	return id
}


// GetGridAttributeID returns the string representation of the ID
func GetGridAttributeID(gridAttributeType types.GridAttributeType, objectType types.ObjectType, objectId uint64) string {
    id := fmt.Sprintf("%d-%d-%d", gridAttributeType, objectType, objectId)
	return id
}

// GetGridAttributeIDByObjectId returns the string representation of the ID
func GetGridAttributeIDByObjectId(gridAttributeType types.GridAttributeType, objectId string) string {
    id := fmt.Sprintf("%d-%s", gridAttributeType, objectId)
	return id
}


func (k Keeper) GetGridAttribute(ctx sdk.Context, gridAttributeId string) (amount uint64, err error) {
	gridAttributeStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GridAttributeKey))

	bz := gridAttributeStore.Get([]byte(gridAttributeId))

	if bz == nil {
        // return error?
        // err =
		amount = 0
	} else {
		amount = binary.BigEndian.Uint64(bz)
	}

	return
}


func (k Keeper) SetGridAttribute(ctx sdk.Context, gridAttributeId string, amount uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GridAttributeKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, amount)

	store.Set([]byte(gridAttributeId), bz)
    _ = ctx.EventManager().EmitTypedEvent(&types.EventGridUpdate{Body: &types.EventBodyKeyPair{Key: gridAttributeId, Value: amount}})
}

func (k Keeper) SetGridAttributeDelta(ctx sdk.Context, gridAttributeId string, oldAmount uint64, newAmount uint64) (amount uint64, err error) {
    currentAmount := k.GetGridAttribute(ctx, gridAttributeId)

    if (oldAmount > currentAmount) {
        // An error that should never happen
    }

    resetAmount = currentAmount - oldAmount
    amount = resetAmount + newAmount

    k.SetGridAttribute(ctx, gridAttributeId, amount)

    return
}

func (k Keeper) SetGridAttributeDecrement(ctx sdk.Context, gridAttributeId string, decrementAmount uint64) (amount uint64, err error) {
    currentAmount := k.GetGridAttribute(ctx, gridAttributeId)

    if (decrementAmount > currentAmount) {
        // An error that should never happen
    }

    amount = currentAmount - decrementAmount

    k.SetGridAttribute(ctx, gridAttributeId, amount)

    return
}

func (k Keeper) SetGridAttributeIncrement(ctx sdk.Context, gridAttributeId string, incrementAmount uint64) (amount uint64) {
    currentAmount := k.GetGridAttribute(ctx, gridAttributeId)

    amount = currentAmount - incrementAmount

    k.SetGridAttribute(ctx, gridAttributeId, amount)

    return
}

func (k Keeper) GetGridCascadeQueue(ctx sdk.Context, clear bool) (queue []string) {
	gridCascadeQueueStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GridCascadeQueue))
	iterator := sdk.KVStorePrefixIterator(gridCascadeQueueStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		queue = append(queue, string(iterator.Key()))
		if clear {
		    gridCascadeQueueStore.Delete(iterator.Key())
		}
	}

    return
}


func (k Keeper) AppendGridCascadeQueue(ctx sdk.Context, queueId string) (err error) {
    gridCascadeQueueStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GridCascadeQueue))

	bz := make([]byte, 1)
	binary.BigEndian.PutBool(bz, bool(true))

	gridCascadeQueueStore.Set([]byte(queueId), bz)
}

func (k Keeper) GridCascade(ctx sdk.Context) {

    // Initiate the attributes
    var allocationPointer       uint64
    var allocationPointerEnd    uint64

    var allocationDestroyed     bool

    // This needs to be able to iterate until the queue is empty
    // If there are no bugs, there should always be an end
    // If there are bugs, Cisphyx will find it
    for {
        // Get Queue (and clear it in the process)
        gridQueue := k.GetGridCascadeQueue(ctx, true)

        if (len(gridQueue) == 0) {
            break
        }

        // For each Queue Item
        for _, objectId := range gridQueue {
            allocationPointer    = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_allocationPointerStart, objectId))
            allocationPointerEnd = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_allocationPointerEnd, objectId))

            for (k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, queueId)) > k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId))) {

                // Iterate through the allocationPointer until we successfully delete an allocation
                for {
                    allocationDestroyed = k.DestroyAllocation(ctx, GetAllocationIDByObjectId(objectId, allocationPointer))
                    allocationPointer   = allocationPointer + 1

                    if ((allocationDestroyed) || (allocationPointer > allocationPointerEnd)) {
                        break
                    }
                }

                if (allocationPointer > allocationPointerEnd) {
                    // This is bad. Better than infinite loop
                    // We've gotten here because we've blown away all the allocations but somehow the capacity is still lower than load
                    // TODO Write function that forcefully resolves the situation
                        // This would be a bandaid on a bug though.
                    break
                }
            }
        }
    }
}

func (k Keeper) UpdateGridConnectionCapacity(ctx sdk.Context, objectId string) {
    capacity := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId))
    load     := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId))

    if (capacity > load) {
        availableCapacity = capacity - load

        connectionCount = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectCount, objectId))
        if (connectionCount == 0) { connectionCount = 1 }

        k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, objectId), availableCapacity / connectionCount)
    } else {
        k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, objectId), 0)
    }
}