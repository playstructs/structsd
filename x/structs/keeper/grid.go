package keeper

import (
	"context"
	"encoding/binary"
	"structs/x/structs/types"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"fmt"

	"slices"
	"strings"
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

// IsValidGridAttributeID returns true if id has the shape "<prefix>-<objectId>"
// with both the prefix and the objectId non-empty. The prefix is the
// GridAttributeType bucket and the objectId is whatever was passed to
// GetGridAttributeIDByObjectId (typically itself "<objectType>-<id>", but in
// some test paths it is a single token like "source1", so we deliberately do
// not require three segments here).
//
// The case we *do* want to catch is the historical bug where callers passed
// an empty objectId, producing keys like "2-". See
// docs/incident-2026-05-grid-orphan.md for the originating incident: a
// pre-daac34c AutoResizeAllocation against an automated allocation with no
// destination wrote `GetGridAttributeIDByObjectId(capacity, "")` = "2-" into
// the grid store.
//
// Used as a defensive backstop in Keeper.SetGridAttribute and as the
// detection rule in Keeper.PruneMalformedGridAttributes.
func IsValidGridAttributeID(id string) bool {
	idx := strings.Index(id, "-")
	if idx <= 0 {
		// No "-", or id starts with "-" (no prefix segment).
		return false
	}
	if idx == len(id)-1 {
		// Ends with "-": the objectId portion is empty (the "2-" shape).
		return false
	}
	return true
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

    ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGrid{&types.GridRecord{AttributeId: gridAttributeId, Value: 0}})
    k.logger.Info("Grid Change (Clear)", "gridAttributeId", gridAttributeId)
}

// SetGridAttribute writes a grid attribute value for the given id.
//
// Defensive backstop: any caller that builds an id via
// GetGridAttributeIDByObjectId(t, "") or otherwise produces a key with an
// empty objectId portion (see IsValidGridAttributeID) is silently dropped
// here, with a loud Error log so the regression surfaces in operator output
// rather than as an orphan KV row. The known historical regression
// (pre-daac34c AutoResizeAllocation writing "2-" when DestinationId == "")
// has been fixed at every callsite, so this backstop should never fire in
// practice; it exists purely to make the next "I forgot a guard" leak
// harmless.
func (k Keeper) SetGridAttribute(ctx context.Context, gridAttributeId string, amount uint64) {
	if !IsValidGridAttributeID(gridAttributeId) {
		k.logger.Error(
			"refusing to write malformed grid attribute id; ignoring write",
			"gridAttributeId", gridAttributeId,
			"amount", amount,
		)
		return
	}

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridAttributeKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, amount)

	store.Set([]byte(gridAttributeId), bz)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGrid{&types.GridRecord{AttributeId: gridAttributeId, Value: amount}})
	k.logger.Info("Grid Change (Set)", "gridAttributeId", gridAttributeId, "amount", amount)
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
		return types.NewObjectNotFoundError("grid_queue", queueId)
	}

	gridCascadeQueueStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridCascadeQueue))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, 1)

	gridCascadeQueueStore.Set([]byte(queueId), bz)

	k.logger.Info("Grid Queue (Add)", "queueId", queueId)

	return err
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

// PruneMalformedGridAttributes walks the GridAttribute store and deletes any
// row whose key does not satisfy IsValidGridAttributeID — i.e. anything that
// is not "<prefix>-<non-empty objectId>". Returns the deleted keys for
// logging.
//
// This is used by the v0.17.0 upgrade handler to recover the testnet "2-"
// orphan documented in docs/incident-2026-05-grid-orphan.md. Idempotent: a
// second invocation finds no malformed rows and returns nil.
func (k Keeper) PruneMalformedGridAttributes(ctx context.Context) []string {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridAttributeKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	var malformed []string
	for ; iterator.Valid(); iterator.Next() {
		key := string(iterator.Key())
		if !IsValidGridAttributeID(key) {
			malformed = append(malformed, key)
		}
	}
	iterator.Close()

	if len(malformed) == 0 {
		return nil
	}

	ctxSDK := sdk.UnwrapSDKContext(ctx)
	for _, key := range malformed {
		store.Delete([]byte(key))
		_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGrid{&types.GridRecord{AttributeId: key, Value: 0}})
		k.logger.Info("Grid Change (Pruned malformed)", "gridAttributeId", key)
	}
	return malformed
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
