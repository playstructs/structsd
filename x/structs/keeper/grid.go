package keeper

import (
	"context"
	"encoding/binary"
	"structs/x/structs/types"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	sdkerrors "cosmossdk.io/errors"
	"fmt"

	"slices"
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

func (k Keeper) GetGridAttribute(ctx context.Context, gridAttributeId string) (amount uint64) {
	gridAttributeStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridAttributeKey))

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

func (k Keeper) ClearGridAttribute(ctx context.Context, gridAttributeId string) {
	gridAttributeStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridAttributeKey))
	gridAttributeStore.Delete([]byte(gridAttributeId))
}

func (k Keeper) SetGridAttribute(ctx context.Context, gridAttributeId string, amount uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridAttributeKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, amount)

	store.Set([]byte(gridAttributeId), bz)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGrid{&types.GridRecord{AttributeId: gridAttributeId, Value: amount}})
	k.logger.Info("Grid Change (Set)", "gridAttributeId", gridAttributeId, "amount", amount)
}

func (k Keeper) SetGridAttributeDelta(ctx context.Context, gridAttributeId string, oldAmount uint64, newAmount uint64) (amount uint64, err error) {
	currentAmount := k.GetGridAttribute(ctx, gridAttributeId)

	var resetAmount uint64
	if oldAmount < currentAmount {
		resetAmount = currentAmount - oldAmount
	}

	amount = resetAmount + newAmount

	k.logger.Info("Grid Change (Delta)", "gridAttributeId", gridAttributeId, "oldAmount", oldAmount, "newAmount", newAmount)
	k.SetGridAttribute(ctx, gridAttributeId, amount)

	return
}

func (k Keeper) SetGridAttributeDecrement(ctx context.Context, gridAttributeId string, decrementAmount uint64) (amount uint64, err error) {
	currentAmount := k.GetGridAttribute(ctx, gridAttributeId)

	if decrementAmount < currentAmount {
		amount = currentAmount - decrementAmount
	}

	k.logger.Info("Grid Change (Decrement)", "gridAttributeId", gridAttributeId, "decrementAmount", decrementAmount)
	k.SetGridAttribute(ctx, gridAttributeId, amount)

	return
}

func (k Keeper) SetGridAttributeIncrement(ctx context.Context, gridAttributeId string, incrementAmount uint64) (amount uint64) {
	currentAmount := k.GetGridAttribute(ctx, gridAttributeId)

	amount = currentAmount + incrementAmount

	k.logger.Info("Grid Change (Increment)", "gridAttributeId", gridAttributeId, "incrementAmount", incrementAmount)
	k.SetGridAttribute(ctx, gridAttributeId, amount)

	return
}

func (k Keeper) GetGridCascadeQueue(ctx context.Context, clear bool) (queue []string) {
	gridCascadeQueueStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridCascadeQueue))
	iterator := storetypes.KVStorePrefixIterator(gridCascadeQueueStore, []byte{})

	for ; iterator.Valid(); iterator.Next() {
		queue = append(queue, string(iterator.Key()))
	}
    iterator.Close()

    slices.Sort(queue)

    if clear {
        for _, key := range queue {
            gridCascadeQueueStore.Delete([]byte(key))
        }
    }
	return
}

func (k Keeper) AppendGridCascadeQueue(ctx context.Context, queueId string) (err error) {

	// Skip if queueId is empty or nil to prevent "key is nil or empty" panic
	if queueId == "" {
		return sdkerrors.Wrapf(types.ErrObjectNotFound, "queueId (%s) empty", queueId)
	}

	gridCascadeQueueStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridCascadeQueue))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, 1)

	gridCascadeQueueStore.Set([]byte(queueId), bz)

	k.logger.Info("Grid Queue (Add)", "queueId", queueId)

	return err
}

func (k Keeper) GridCascade(ctx context.Context) {

	// This needs to be able to iterate until the queue is empty
	// If there are no bugs, there should always be an end
	// If there are bugs, Cisphyx will find it
	for {
		// Get Queue (and clear it in the process)
		gridQueue := k.GetGridCascadeQueue(ctx, true)

		if len(gridQueue) == 0 {
			break
		}

		// For each Queue Item
		for _, objectId := range gridQueue {

			allocationList := k.GetAllAllocationIdBySourceIndex(ctx, objectId)
			allocationPointer := 0

			for k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId)) > k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId)) {
				k.logger.Info("Grid Queue (Brownout)", "objectId", objectId, "load", k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId)), "capacity", k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId)))

				k.DestroyAllocation(ctx, allocationList[allocationPointer])
				k.logger.Info("Grid Queue (Allocation Destroyed)", "allocationId", allocationList[allocationPointer])

				allocationPointer++
			}
		}
	}
}

func (k Keeper) UpdateGridConnectionCapacity(ctx context.Context, objectId string) {
	capacity := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId))
	load := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId))

	if capacity > load {
		availableCapacity := capacity - load

		connectionCount := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, objectId))
		if connectionCount == 0 {
			connectionCount = 1
		}

		k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, objectId), availableCapacity/connectionCount)
	} else {
		k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, objectId), 0)
	}
}

// GetAllGridExport returns all grid attributes
func (k Keeper) GetAllGridExport(ctx context.Context) (list []*types.GridRecord) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridAttributeKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		list = append(list, &types.GridRecord{AttributeId: string(iterator.Key()), Value: binary.BigEndian.Uint64(iterator.Value())})
	}

	return
}

func (k Keeper) GetGridAttributesByObject(ctx context.Context, objectId string) types.GridAttributes {
	return types.GridAttributes{
		Ore:                k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_ore, objectId)),
		Fuel:               k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, objectId)),
		Capacity:           k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId)),
		Load:               k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId)),
		StructsLoad:        k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, objectId)),
		Power:              k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_power, objectId)),
		ConnectionCapacity: k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, objectId)),
		ConnectionCount:    k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, objectId)),
		ProxyNonce:         k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_proxyNonce, objectId)),
		LastAction:         k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, objectId)),
		Nonce:              k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_nonce, objectId)),
		Ready:              k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_ready, objectId)),
		CheckpointBlock:    k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_checkpointBlock, objectId)),
	}
}

func (k Keeper) GetGridCascadeQueueExport(ctx context.Context) (queue []string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridCascadeQueue))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		queue = append(queue, string(iterator.Key()))
	}
	return
}
