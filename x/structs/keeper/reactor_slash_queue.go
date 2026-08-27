package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// ReactorSlashReconcileEntry is one reactor still being reconciled, and the
// infusion address its next batch resumes from.
type ReactorSlashReconcileEntry struct {
	ReactorId string
	Cursor    string
}

func reactorSlashReconcileStore(k Keeper, ctx context.Context) prefix.Store {
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.ReactorSlashReconcileQueue))
}

/* EnqueueReactorSlashReconcile marks a reactor as needing reconciliation.
 *
 * Re-enqueuing a reactor already in the queue resets it to the start rather than
 * leaving the old cursor: a second slash lands on infusions the first pass may
 * already have walked past, and reconciliation reads live staking state, so
 * starting over is correct and re-reading an entry costs nothing but the read.
 */
func (k Keeper) EnqueueReactorSlashReconcile(ctx context.Context, reactorId string) {
	if reactorId == "" {
		return
	}

	reactorSlashReconcileStore(k, ctx).Set([]byte(reactorId), []byte(""))
	k.logger.Info("Reactor queued for slash reconciliation", "reactorId", reactorId)
}

// GetReactorSlashReconcileQueue lists the reactors still to reconcile. The
// queue holds one row per slashed reactor, so it is short by construction.
func (k Keeper) GetReactorSlashReconcileQueue(ctx context.Context) (queue []ReactorSlashReconcileEntry) {
	store := reactorSlashReconcileStore(k, ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		queue = append(queue, ReactorSlashReconcileEntry{
			ReactorId: string(iterator.Key()),
			Cursor:    string(iterator.Value()),
		})
	}

	return
}

func (k Keeper) setReactorSlashReconcileCursor(ctx context.Context, reactorId string, cursor string) {
	reactorSlashReconcileStore(k, ctx).Set([]byte(reactorId), []byte(cursor))
}

func (k Keeper) clearReactorSlashReconcile(ctx context.Context, reactorId string) {
	reactorSlashReconcileStore(k, ctx).Delete([]byte(reactorId))
}

/* ProcessReactorSlashReconcileQueue reconciles a bounded number of infusions
 * against live staking state, resuming where the last block stopped.
 *
 * Deferred to the EndBlocker on purpose, and not only for the budget: the
 * slashing hook fires *before* staking applies the slash, which is why it had to
 * derive the post-slash token figure by hand. By the time this runs the slash is
 * on disk, so each infusion can simply be refreshed from what staking now says -
 * no captured figure to keep in step with.
 */
func (k Keeper) ProcessReactorSlashReconcileQueue(ctx context.Context) {
	queue := k.GetReactorSlashReconcileQueue(ctx)
	if len(queue) == 0 {
		return
	}

	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	budget := types.ReactorSlashReconcileBudget

	for _, entry := range queue {
		if budget <= 0 {
			break
		}

		reactor, found := k.GetReactor(ctx, entry.ReactorId)
		if !found {
			k.clearReactorSlashReconcile(ctx, entry.ReactorId)
			continue
		}

		validatorAddress, err := sdk.ValAddressFromBech32(reactor.Validator)
		if err != nil {
			k.logger.Error("Reactor slash reconciliation: unparseable validator", "reactorId", entry.ReactorId, "validator", reactor.Validator, "error", err)
			k.clearReactorSlashReconcile(ctx, entry.ReactorId)
			continue
		}

		// Read the batch and close the iterator before reconciling anything, so
		// nothing walks the infusion prefix while the reconciliation is writing
		// into it.
		addresses, next := k.readInfusionAddressBatch(ctx, entry.ReactorId, entry.Cursor, budget)

		for _, address := range addresses {
			budget--

			accountAddress, addressErr := sdk.AccAddressFromBech32(address)
			if addressErr != nil {
				k.logger.Error("Reactor slash reconciliation: unparseable infusion address", "reactorId", entry.ReactorId, "address", address, "error", addressErr)
				continue
			}

			// Deliberately the "existing" variant: this walks infusions that are
			// already on file, so there is no player to create and an address
			// that no longer resolves to one must not get one now.
			k.reconcileExistingInfusionForDelegation(ctx, cc, accountAddress, validatorAddress)
		}

		if next == "" {
			k.clearReactorSlashReconcile(ctx, entry.ReactorId)
			k.logger.Info("Reactor slash reconciliation complete", "reactorId", entry.ReactorId)
			continue
		}

		k.setReactorSlashReconcileCursor(ctx, entry.ReactorId, next)
		k.logger.Info("Reactor slash reconciliation deferred", "reactorId", entry.ReactorId, "resumeFrom", next)
	}
}

// readInfusionAddressBatch returns up to limit infusion addresses for a
// destination starting at cursor, and the address the next batch should resume
// from ("" when the destination is exhausted).
func (k Keeper) readInfusionAddressBatch(ctx context.Context, destinationId string, cursor string, limit int) (addresses []string, next string) {
	if limit <= 0 {
		return nil, cursor
	}

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), InfusionKeyPrefix(destinationId))
	iterator := store.Iterator([]byte(cursor), nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		if len(addresses) >= limit {
			// The cursor is the next key to read, not the last one read, so a
			// resume neither repeats nor skips an entry.
			return addresses, string(iterator.Key())
		}
		addresses = append(addresses, string(iterator.Key()))
	}

	return addresses, ""
}

// ReadInfusionAddressBatchForTest exposes the batch walk so the cursor
// semantics can be tested directly. The property it guards - that a batched
// walk reproduces a single full scan exactly - is not observable from the
// outside, and getting it wrong strands an infusion on pre-slash fuel forever.
func (k Keeper) ReadInfusionAddressBatchForTest(ctx context.Context, destinationId string, cursor string, limit int) ([]string, string) {
	return k.readInfusionAddressBatch(ctx, destinationId, cursor, limit)
}
