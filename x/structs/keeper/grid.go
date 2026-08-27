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
	"strconv"
	"strings"
)

// GetObjectID returns the string representation of the ID, based on ObjectType
// This is the unified objectId model across the system
func GetObjectID(objectType types.ObjectType, objectId uint64) string {
	id := fmt.Sprintf("%d-%d", objectType, objectId)
	return id
}

/* ObjectIdHasType reports whether an object id names the given type.
 *
 * Every grid attribute id is derived from the object id string and nothing else,
 * so two different kinds of object that share an id string share their capacity
 * and load counters. Ids are namespaced by type precisely to stop that, and a
 * handler that accepts a message-supplied id without checking the namespace
 * throws the protection away: a player id used where a substation id belongs
 * resolves to a cache that reads and writes the *player's* grid attributes.
 */
func ObjectIdHasType(objectId string, objectType types.ObjectType) bool {
	parts := strings.SplitN(objectId, "-", 2)
	if len(parts) != 2 {
		return false
	}

	typeNum, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return false
	}

	return types.ObjectType(typeNum) == objectType
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


// GridCascadeQueueEntry is one pending cascade. The sequence travels with the
// object id because the processor has to delete the row it read, and only the
// sequence identifies it.
type GridCascadeQueueEntry struct {
	Sequence uint64
	ObjectId string
}

func gridCascadeQueueSequenceKey(sequence uint64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, sequence)
	return key
}

// nextGridCascadeQueueSequence hands out the next sequence and advances the
// counter. Big-endian so the store's byte order is the numeric order.
func (k Keeper) nextGridCascadeQueueSequence(ctx context.Context) uint64 {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})

	var next uint64
	if bz := store.Get(types.KeyPrefix(types.GridCascadeQueueSequenceKey)); bz != nil {
		next = binary.BigEndian.Uint64(bz)
	}

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, next+1)
	store.Set(types.KeyPrefix(types.GridCascadeQueueSequenceKey), bz)

	return next
}

/* GetGridCascadeQueueBatch reads up to limit pending entries in sequence order.
 *
 * It does not clear them. The processor deletes each entry as it finishes it,
 * so an entry the budget did not reach is still queued in the next block - which
 * is the whole point of the budget. See GridCascadeBlockBudget.
 */
func (k Keeper) GetGridCascadeQueueBatch(ctx context.Context, limit int) (batch []GridCascadeQueueEntry) {
	if limit <= 0 {
		return
	}

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridCascadeQueue))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		if len(batch) >= limit {
			return
		}

		key := iterator.Key()
		if len(key) != 8 {
			// Not a sequence row. Nothing writes one, but a malformed key would
			// otherwise panic the EndBlocker rather than be skipped.
			continue
		}

		batch = append(batch, GridCascadeQueueEntry{
			Sequence: binary.BigEndian.Uint64(key),
			ObjectId: string(iterator.Value()),
		})
	}

	return
}

// GetGridCascadeQueueLength reports how many entries are pending. Used for
// logging the backlog the budget left behind, so a cap is never silent.
func (k Keeper) GetGridCascadeQueueLength(ctx context.Context) (count int) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridCascadeQueue))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		count++
	}

	return
}

// RemoveGridCascadeQueueEntry clears a processed entry. Both rows go, keyed by
// the same values they were written under.
func (k Keeper) RemoveGridCascadeQueueEntry(ctx context.Context, entry GridCascadeQueueEntry) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridCascadeQueue))
	store.Delete(gridCascadeQueueSequenceKey(entry.Sequence))

	if entry.ObjectId != "" {
		indexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridCascadeQueueIndex))
		indexStore.Delete([]byte(entry.ObjectId))
	}
}

// GetGridCascadeQueue returns the pending object ids in sequence order,
// optionally clearing the queue. Genesis, tests and diagnostics use it; the
// EndBlocker does not, because draining to exhaustion is what this issue was.
func (k Keeper) GetGridCascadeQueue(ctx context.Context, clear bool) (queue []string) {
	batch := k.GetGridCascadeQueueBatch(ctx, int(^uint(0)>>1))

	for _, entry := range batch {
		queue = append(queue, entry.ObjectId)

		if clear {
			k.RemoveGridCascadeQueueEntry(ctx, entry)
		}
	}

	return
}

/* AppendGridCascadeQueue queues an object for the next cascade pass.
 *
 * An object already pending is left where it is rather than re-queued. Moving
 * it to the back would let a stream of appends push an entry backwards forever,
 * which is the starvation the sequence exists to prevent, and a second row for
 * the same object is redundant work: the cascade reads live load and capacity,
 * so one visit accounts for every append that preceded it.
 */
func (k Keeper) AppendGridCascadeQueue(ctx context.Context, queueId string) (err error) {

	// Skip if queueId is empty or nil to prevent "key is nil or empty" panic
	if queueId == "" {
		return types.NewObjectNotFoundError("grid_queue", queueId)
	}

	indexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridCascadeQueueIndex))
	if indexStore.Has([]byte(queueId)) {
		return nil
	}

	sequence := k.nextGridCascadeQueueSequence(ctx)

	gridCascadeQueueStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridCascadeQueue))
	gridCascadeQueueStore.Set(gridCascadeQueueSequenceKey(sequence), []byte(queueId))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, sequence)
	indexStore.Set([]byte(queueId), bz)

	k.logger.Info("Grid Queue (Add)", "queueId", queueId, "sequence", sequence)

	return err
}


/* MigrateGridCascadeQueueToSequence re-keys a pre-v0.22.0 cascade queue.
 *
 * The old rows were keyed by object id with a placeholder value; the new ones
 * are keyed by sequence and hold the object id, with a reverse index for
 * dedup. Every row under the prefix is therefore read as a legacy object-id key,
 * which is only safe because nothing has written the new shape yet - object ids
 * are not fixed width, so an id eight bytes long is indistinguishable from a
 * sequence once the two shapes coexist. Run this before anything in the upgrade
 * that can enqueue.
 *
 * Re-appended in sorted order, which is the order the old cascade processed
 * them in, so the queue a restarted chain drains is the queue it would have
 * drained.
 */
func (k Keeper) MigrateGridCascadeQueueToSequence(ctx context.Context) (migrated int, err error) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.GridCascadeQueue))

	var legacy []string
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	for ; iterator.Valid(); iterator.Next() {
		legacy = append(legacy, string(iterator.Key()))
	}
	iterator.Close()

	if len(legacy) == 0 {
		return 0, nil
	}

	for _, key := range legacy {
		store.Delete([]byte(key))
	}

	slices.Sort(legacy)

	for _, objectId := range legacy {
		if appendErr := k.AppendGridCascadeQueue(ctx, objectId); appendErr != nil {
			// An empty id could not have been written by AppendGridCascadeQueue,
			// which refuses one. Drop it rather than fail the upgrade.
			k.logger.Warn("Grid Queue (migration skipped entry)", "objectId", objectId, "error", appendErr)
			continue
		}
		migrated++
	}

	return migrated, nil
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

// GetGridCascadeQueueExport returns the pending object ids in sequence order.
// Genesis import re-appends them in that order, which reproduces the queue the
// export was taken from.
func (k Keeper) GetGridCascadeQueueExport(ctx context.Context) (queue []string) {
	return k.GetGridCascadeQueue(ctx, false)
}
