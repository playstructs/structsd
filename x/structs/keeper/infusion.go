package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"context"

	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"encoding/binary"
	//"strconv"
	"slices"
	"strings"
)

func InfusionKeyPrefix(destinationId string) []byte {
	return []byte(types.InfusionKey + destinationId + "/")
}

func GetInfusionID(address string) []byte {
	return []byte(address)
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
	if len(infusionIdSplit) != 3 {
		return types.Infusion{}, false
	}
	return k.GetInfusion(ctx, infusionIdSplit[0] + "-" + infusionIdSplit[1], infusionIdSplit[2])
}

// RemoveInfusion removes a infusion from the store
func (k Keeper) RemoveInfusion(ctx context.Context, destinationId string, address string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), InfusionKeyPrefix(destinationId))

	store.Delete(GetInfusionID(address))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	infusionId := destinationId + "-" + address
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ObjectId: infusionId})
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
		list = append(list, val)
	}

	return
}


// GetAllInfusionsByDestination returns all infusion relating to a struct
func (k Keeper) GetAllInfusionIdsByDestination(ctx context.Context, objectId string) (list []string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), InfusionKeyPrefix(objectId))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Infusion
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		allocationId := objectId + "-" + val.Address
		list = append(list, allocationId)
	}

	return
}

func (k Keeper) GetInfusionDestructionQueue(ctx context.Context, clear bool) (queue []string) {
	infusionDestructionQueueStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.InfusionDestructionQueue))
	iterator := storetypes.KVStorePrefixIterator(infusionDestructionQueueStore, []byte{})

	for ; iterator.Valid(); iterator.Next() {
		queue = append(queue, string(iterator.Key()))
	}
	iterator.Close()

	slices.Sort(queue)

	if clear {
		for _, key := range queue {
			infusionDestructionQueueStore.Delete([]byte(key))
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

func (k Keeper) GetInfusionDestructionQueueExport(ctx context.Context) (queue []string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.InfusionDestructionQueue))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		queue = append(queue, string(iterator.Key()))
	}
	return
}
