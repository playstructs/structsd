package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
)

const reactorRepairSnapshotPrefix = "Reactor/repairSnapshot/"

// ReactorRepairSnapshot selects a finite (After, Through] insertion interval.
// Revision identifies the slash accounting this interval must repair.
type ReactorRepairSnapshot struct {
	After    uint64
	Through  uint64
	Revision uint64
}

func (k Keeper) reactorRepairSnapshotStore(ctx context.Context) prefix.Store {
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte(reactorRepairSnapshotPrefix))
}

func (k Keeper) GetReactorRepairSnapshot(ctx context.Context, id string) (ReactorRepairSnapshot, bool, error) {
	value := k.reactorRepairSnapshotStore(ctx).Get([]byte(id))
	if value == nil {
		return ReactorRepairSnapshot{}, false, nil
	}
	if len(value) != 24 {
		return ReactorRepairSnapshot{}, true, fmt.Errorf("invalid reactor repair snapshot for %s", id)
	}
	snapshot := ReactorRepairSnapshot{After: binary.BigEndian.Uint64(value[:8]), Through: binary.BigEndian.Uint64(value[8:16]), Revision: binary.BigEndian.Uint64(value[16:])}
	if snapshot.After > snapshot.Through {
		return snapshot, true, fmt.Errorf("invalid reactor repair interval for %s", id)
	}
	return snapshot, true, nil
}

func (k Keeper) setReactorRepairSnapshot(ctx context.Context, id string, snapshot ReactorRepairSnapshot) {
	value := make([]byte, 24)
	binary.BigEndian.PutUint64(value[:8], snapshot.After)
	binary.BigEndian.PutUint64(value[8:16], snapshot.Through)
	binary.BigEndian.PutUint64(value[16:], snapshot.Revision)
	k.reactorRepairSnapshotStore(ctx).Set([]byte(id), value)
}

func (k Keeper) captureReactorRepairSnapshot(ctx context.Context, id string) (ReactorRepairSnapshot, error) {
	revision, err := k.GetReactorCommissionRevision(ctx, id)
	if err != nil {
		return ReactorRepairSnapshot{}, err
	}
	snapshot := ReactorRepairSnapshot{Through: k.ReactorInfusionOrderEnd(ctx, id), Revision: revision.Issued}
	k.setReactorRepairSnapshot(ctx, id, snapshot)
	return snapshot, nil
}

// ResetReactorRepairSnapshots is upgrade-only. Rebuilding insertion order makes
// older numeric cursors meaningless, so pending jobs restart against the rebuilt
// inventory. Ordinary queue processing never scans all jobs this way.
func (k Keeper) ResetReactorRepairSnapshots(ctx context.Context) {
	for _, entry := range k.GetReactorSlashReconcileQueue(ctx) {
		if _, err := k.captureReactorRepairSnapshot(ctx, entry.ReactorId); err != nil {
			panic(err) // The enclosing upgrade cache rolls back an invalid rebuild.
		}
		k.setReactorSlashReconcileCursor(ctx, entry.ReactorId, "")
	}
}
