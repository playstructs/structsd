package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"context"
	"time"

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

// =====================================================================
// InfusionMaturitySweepQueue
//
// Time-indexed queue of (CompletionTime, infusionKey) rows that the structs
// EndBlocker drains as their maturity time arrives. Modeled on staking's
// UBDQueue. Cosmos SDK v0.53 does not expose an unbonding-completion hook,
// so the structs module records each new UBD entry's CompletionTime when
// AfterUnbondingInitiated fires, then walks the queue each block to
// reconcile Defusing for entries whose maturity has passed.
//
// Store key layout: prefix + sdk.FormatTimeBytes(completionTime) + "/" + infusionKey
// FormatTimeBytes is fixed-width and lexicographically sortable.
// =====================================================================

// maturitySweepKey builds the (time-bytes + "/" + infusionKey) suffix used inside the prefix store.
func maturitySweepKey(completionTime time.Time, infusionKey string) []byte {
	timeBz := sdk.FormatTimeBytes(completionTime)
	key := make([]byte, 0, len(timeBz)+1+len(infusionKey))
	key = append(key, timeBz...)
	key = append(key, '/')
	key = append(key, []byte(infusionKey)...)
	return key
}

// EnqueueInfusionMaturitySweep records that the given infusion has a UBD entry
// maturing at completionTime. Idempotent on the (time, infusionKey) pair.
func (k Keeper) EnqueueInfusionMaturitySweep(ctx context.Context, completionTime time.Time, infusionKey string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.InfusionMaturitySweepQueue))
	store.Set(maturitySweepKey(completionTime, infusionKey), []byte{0x01})
}

// DequeueMatureInfusionSweeps returns the infusion keys whose recorded
// CompletionTime is <= blockTime, deleting their queue rows. Caller is
// expected to invoke ReconcileInfusionForDelegation for each returned key.
//
// Duplicate infusion keys are de-duplicated before return so a single sweep
// pass reconciles each (delegator, validator) at most once even if multiple
// UBD entries on that pair matured in the same block.
func (k Keeper) DequeueMatureInfusionSweeps(ctx context.Context, blockTime time.Time) []string {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.InfusionMaturitySweepQueue))

	// Upper bound (exclusive) for the iterator: smallest key whose time
	// component is strictly greater than blockTime. PrefixEndBytes("<time>/")
	// returns the byte slice immediately after every key prefixed with
	// "<time>/", which captures every entry whose time matches blockTime
	// exactly plus everything earlier.
	timeBz := sdk.FormatTimeBytes(blockTime)
	timeBz = append(timeBz, '/')
	end := storetypes.PrefixEndBytes(timeBz)

	iter := store.Iterator(nil, end)

	var (
		keysToDelete [][]byte
		seen         = map[string]struct{}{}
		out          []string
	)

	for ; iter.Valid(); iter.Next() {
		fullKey := iter.Key()
		dup := make([]byte, len(fullKey))
		copy(dup, fullKey)
		keysToDelete = append(keysToDelete, dup)

		// Strip the leading "<sortable-time-bytes>/" prefix to recover the
		// infusion key. FormatTimeBytes uses a fixed width, but we don't need
		// to hard-code it: the first '/' is the separator we wrote.
		idx := -1
		for i, b := range dup {
			if b == '/' {
				idx = i
				break
			}
		}
		if idx < 0 || idx+1 >= len(dup) {
			continue
		}
		infusionKey := string(dup[idx+1:])
		if _, ok := seen[infusionKey]; ok {
			continue
		}
		seen[infusionKey] = struct{}{}
		out = append(out, infusionKey)
	}
	iter.Close()

	for _, kk := range keysToDelete {
		store.Delete(kk)
	}

	return out
}

// GetInfusionMaturitySweepQueueExport returns the full queue contents as
// raw composite-key strings (sdk.FormatTimeBytes(completionTime) + "/" + infusionKey)
// for genesis export and tests. The raw form is preserved so an export+import
// cycle reconstructs the queue with byte-identical store keys.
func (k Keeper) GetInfusionMaturitySweepQueueExport(ctx context.Context) (rows []string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.InfusionMaturitySweepQueue))
	iter := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		rows = append(rows, string(iter.Key()))
	}
	return
}

// ImportInfusionMaturitySweepRow writes a single raw composite-key row produced
// by GetInfusionMaturitySweepQueueExport back into the store. Used by
// InitGenesis to round-trip the queue without re-deriving the time prefix.
// Empty rows are ignored so that an export from a chain that never wrote to
// the queue (pre-v0.17.0) is a safe no-op.
func (k Keeper) ImportInfusionMaturitySweepRow(ctx context.Context, rawKey string) {
	if rawKey == "" {
		return
	}
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.InfusionMaturitySweepQueue))
	store.Set([]byte(rawKey), []byte{0x01})
}
