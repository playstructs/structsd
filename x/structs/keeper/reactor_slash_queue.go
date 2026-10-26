package keeper

import (
	"context"
	"errors"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// ReactorSlashReconcileEntry is one reactor still being reconciled, and the
// diagnostic infusion address. Numeric snapshot state controls actual resumption;
// this address is the last processed row, or a pending failure after that pass.
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
	if _, err := k.captureReactorRepairSnapshot(ctx, reactorId); err != nil {
		k.logger.Error("Cannot capture reactor repair snapshot", "reactorId", reactorId, "error", err)
	}
	k.logger.Info("Reactor queued for slash reconciliation", "reactorId", reactorId)
}

// EnqueueReactorRestart schedules a permissionless repair without rewinding
// work already queued by a slash, bonding transition or earlier restart.
func (k Keeper) EnqueueReactorRestart(ctx context.Context, reactorId string) {
	if reactorId == "" || reactorSlashReconcileStore(k, ctx).Has([]byte(reactorId)) {
		return
	}
	k.EnqueueReactorSlashReconcile(ctx, reactorId)
}

// GetReactorSlashReconcileQueue lists every pending reactor for inspection.
// Live processing uses a bounded prefix read instead of materializing this list.
func (k Keeper) GetReactorSlashReconcileQueue(ctx context.Context) (queue []ReactorSlashReconcileEntry) {
	return k.readReactorSlashReconcileQueue(ctx, 0)
}

// A zero limit is reserved for the full diagnostic listing above. The block
// hook passes a positive limit so empty/missing reactors also consume a bounded
// number of reads and deletes, independently of the infusion work budget.
func (k Keeper) readReactorSlashReconcileQueue(ctx context.Context, limit int) (queue []ReactorSlashReconcileEntry) {
	store := reactorSlashReconcileStore(k, ctx)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		if limit > 0 && len(queue) >= limit {
			break
		}
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
	k.reactorRepairSnapshotStore(ctx).Delete([]byte(reactorId))
	k.reactorRepairRetryStore(ctx, "end/").Delete([]byte(reactorId))
}

// Start after the last serviced reactor and wrap once. Both iterator ranges
// share one row limit; a long backlog never becomes a full scan. The position
// is a scheduling hint and remains valid if that reactor has since been removed.
func (k Keeper) readRotatingReactorSlashQueue(ctx context.Context, limit int) []ReactorSlashReconcileEntry {
	if limit <= 0 {
		return nil
	}
	root := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	position := root.Get(types.KeyPrefix(types.ReactorSlashReconcilePosition))
	store := reactorSlashReconcileStore(k, ctx)
	var queue []ReactorSlashReconcileEntry
	readRange := func(start, end []byte) {
		iterator := store.Iterator(start, end)
		defer iterator.Close()
		for ; iterator.Valid() && len(queue) < limit; iterator.Next() {
			queue = append(queue, ReactorSlashReconcileEntry{ReactorId: string(iterator.Key()), Cursor: string(iterator.Value())})
		}
	}
	if len(position) == 0 {
		readRange(nil, nil)
		return queue
	}
	// Appending a zero byte sorts strictly after the exact key, without
	// excluding longer keys that share its prefix (for example 3-1 and 3-10).
	after := append(append([]byte(nil), position...), 0)
	readRange(after, nil)
	if len(queue) < limit {
		readRange(nil, after)
	}
	return queue
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
	queue := k.readRotatingReactorSlashQueue(ctx, types.ReactorSlashReconcileBudget)
	if len(queue) == 0 {
		return
	}

	type completedRepair struct {
		id       string
		revision uint64
	}
	var completed []completedRepair

	budget := types.ReactorSlashReconcileBudget
	lastProcessed := ""

	for _, entry := range queue {
		if budget <= 0 {
			break
		}
		lastProcessed = entry.ReactorId

		reactor, found := k.GetReactor(ctx, entry.ReactorId)
		if !found {
			// Missing reactors can still have outstanding retries. Drain them
			// with the same budget rather than leaving orphaned queue storage.
			retries, err := k.readReactorRepairRetries(ctx, entry.ReactorId, budget)
			if err != nil {
				// Nothing about a missing reactor can be repaired, so an
				// unreadable retry row must not charge every block's whole
				// budget to it for ever. Drop the job; a later slash or
				// restart re-enqueues a reactor that still exists.
				k.logger.Error("Reactor retry queue invalid for missing reactor", "reactorId", entry.ReactorId, "error", err)
				budget--
				k.clearReactorSlashReconcile(ctx, entry.ReactorId)
				continue
			}
			for _, retry := range retries {
				budget--
				k.removeReactorRepairRetry(ctx, entry.ReactorId, retry.address)
			}
			if !k.hasReactorRepairRetries(ctx, entry.ReactorId) {
				k.clearReactorSlashReconcile(ctx, entry.ReactorId)
			}
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
		snapshot, found, snapshotErr := k.GetReactorRepairSnapshot(ctx, entry.ReactorId)
		if snapshotErr != nil {
			// An undecodable snapshot is replaced, not revisited every block.
			k.logger.Error("Reactor repair snapshot invalid", "reactorId", entry.ReactorId, "error", snapshotErr)
			found = false
		}
		if !found {
			// Legacy queue entries predate numeric cursors. Restart their proof
			// against the insertion index established by migration/genesis.
			snapshot, snapshotErr = k.captureReactorRepairSnapshot(ctx, entry.ReactorId)
			if snapshotErr != nil {
				k.logger.Error("Cannot capture reactor repair snapshot", "reactorId", entry.ReactorId, "error", snapshotErr)
				k.clearReactorSlashReconcile(ctx, entry.ReactorId)
				continue
			}
		}
		revision, revisionErr := k.GetReactorCommissionRevision(ctx, entry.ReactorId)
		if revisionErr != nil || snapshot.Revision != revision.Issued {
			k.logger.Error("Reactor repair revision mismatch", "reactorId", entry.ReactorId, "error", revisionErr)
			continue
		}
		// Reserve at least half of a nonempty main pass's remaining budget
		// for forward progress. Read retries before creating any new ones;
		// a failure is never retried repeatedly within this invocation.
		retryLimit := budget
		if snapshot.After < snapshot.Through {
			retryLimit = (budget + 1) / 2
		}
		retries, retryErr := k.readReactorRepairRetries(ctx, entry.ReactorId, retryLimit)
		if retryErr != nil {
			// Charge this job one unit, not the whole block, so one bad row
			// cannot starve every other reactor's repair.
			budget--
			k.logger.Error("Reactor retry queue invalid", "reactorId", entry.ReactorId, "error", retryErr)
			continue
		}
		retried := make(map[string]bool, len(retries))
		for _, retry := range retries {
			budget--
			retried[retry.address] = true
			err := k.repairReactorPosition(ctx, reactor, validatorAddress, snapshot.Revision, retry.address, retry.mode)
			k.removeReactorRepairRetry(ctx, entry.ReactorId, retry.address)
			if err != nil {
				k.enqueueReactorRepairRetry(ctx, entry.ReactorId, retry.address, retry.mode)
				k.logger.Error("Reactor position retry failed", "reactorId", entry.ReactorId, "address", retry.address, "error", err)
			}
		}
		positions, more, readErr := k.ReactorInfusionOrderBatch(ctx, entry.ReactorId, snapshot.After, snapshot.Through, budget)
		if readErr != nil {
			budget--
			k.logger.Error("Reactor repair order invalid", "reactorId", entry.ReactorId, "error", readErr)
			continue
		}
		for _, position := range positions {
			budget--
			if !retried[position.Address] {
				if err := k.repairReactorPosition(ctx, reactor, validatorAddress, snapshot.Revision, position.Address, 0); err != nil {
					k.enqueueReactorRepairRetry(ctx, entry.ReactorId, position.Address, 0)
					k.logger.Error("Reactor position repair failed", "reactorId", entry.ReactorId, "address", position.Address, "error", err)
				} else {
					k.removeReactorRepairRetry(ctx, entry.ReactorId, position.Address)
				}
			}
			snapshot.After = position.Ordinal
		}
		if len(positions) != 0 {
			k.setReactorSlashReconcileCursor(ctx, entry.ReactorId, positions[len(positions)-1].Address)
		}
		if !more {
			snapshot.After = snapshot.Through
		}
		k.setReactorRepairSnapshot(ctx, entry.ReactorId, snapshot)
		if !more && !k.hasReactorRepairRetries(ctx, entry.ReactorId) {
			completed = append(completed, completedRepair{id: entry.ReactorId, revision: snapshot.Revision})
		} else if !more {
			pending, err := k.readReactorRepairRetries(ctx, entry.ReactorId, 1)
			if err == nil && len(pending) != 0 {
				k.setReactorSlashReconcileCursor(ctx, entry.ReactorId, pending[0].address)
			}
		}

	}
	// Publish accounting before eligibility. Completion is tied to the exact
	// slash revision; clearing an older job must never erase a newer transition.
	for _, repair := range completed {
		if k.certifyReactorCommission(ctx, repair.id, repair.revision) {
			k.clearReactorSlashReconcile(ctx, repair.id)
		}
	}
	if lastProcessed != "" {
		root := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
		root.Set(types.KeyPrefix(types.ReactorSlashReconcilePosition), []byte(lastProcessed))
	}
}

// Each attempt has its own SDK and object caches. A failed repair may have
// re-homed an infusion before detecting invalid backing; none of those partial
// changes or events may leak into a successful neighbour's commit.
func (k Keeper) repairReactorPosition(ctx context.Context, reactor types.Reactor, validator sdk.ValAddress, revision uint64, address string, mode byte) error {
	account, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return err
	}
	cached, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	cc := k.NewCurrentContext(cached)
	exists := cc.GetInfusion(reactor.Id, address).CheckInfusion() == nil
	if mode != 0 && !exists {
		if _, err := k.stakingKeeper.GetDelegation(cached, account, validator); err != nil {
			if errors.Is(err, stakingtypes.ErrNoDelegation) {
				return nil
			}
			return err
		}
		if mode == 1 {
			_, vault := k.GetVaultOwner(cached, account)
			if !vault && k.GetPlayerIndexFromAddress(cached, address) == 0 {
				return nil
			}
		}
		if _, _, err := k.reconcileInfusionForDelegation(cached, cc, account, validator); err != nil {
			return err
		}
	} else if revision != 0 {
		// A legitimately destroyed position has no insertion-index row either.
		// A dangling index remains a failure and cannot certify a slash.
		if !exists && !k.reactorInfusionOrderStore(cached, "address/"+reactor.Id+"/").Has([]byte(address)) {
			return nil
		}
		if err := k.reconcileSlashedReactorPosition(cached, cc, reactor, account, validator); err != nil {
			return err
		}
	} else if err := k.reconcileExistingInfusionForDelegation(cached, cc, account, validator); err != nil {
		return err
	}
	cc.CommitAll()
	write()
	return nil
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
