package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

const reactorCommissionReplayKey = "Reactor/commissionReplayPending"

// ScheduleReactorCommissionRevisionReplay is genesis-only. InitChain cannot
// deliver events to block consumers, and imported chains may start above 1.
// Persist the obligation until their first successful block publishes it.
func (k Keeper) ScheduleReactorCommissionRevisionReplay(ctx context.Context) {
	runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)).Set([]byte(reactorCommissionReplayKey), []byte{1})
}

func (k Keeper) processReactorCommissionRevisionReplay(ctx context.Context) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	if !store.Has([]byte(reactorCommissionReplayKey)) {
		return
	}
	// EventAllGenesis has already published this snapshot at height 1.
	if sdk.UnwrapSDKContext(ctx).BlockHeight() > 1 {
		if err := k.EmitReactorCommissionRevisionSnapshot(ctx); err != nil {
			panic(err)
		}
		if err := k.EmitOwnerReactorDiversitySnapshot(ctx); err != nil { panic(err) }
	}
	store.Delete([]byte(reactorCommissionReplayKey))
}

func (k Keeper) emitReactorCommissionRevision(ctx context.Context, id string, revision ReactorCommissionRevision) error {
	return sdk.UnwrapSDKContext(ctx).EventManager().EmitTypedEvent(&types.EventReactorCommissionRevision{
		ReactorId: id, SlashRevision: revision.Issued, ReconciledSlashRevision: revision.Certified,
	})
}

// EmitReactorCommissionRevisionSnapshot is restricted to genesis replay and
// upgrades. Validate the full snapshot before emitting its reset marker. Each
// subsequent event contains one fixed-size revision pair; there is no single
// unbounded repeated-field event. Missing rows mean the initial 0/0 state.
func (k Keeper) EmitReactorCommissionRevisionSnapshot(ctx context.Context) error {
	var records []types.EventReactorCommissionRevision
	if err := func() error {
		iterator := k.reactorCommissionRevisionStore(ctx).Iterator(nil, nil)
		defer iterator.Close()
		for ; iterator.Valid(); iterator.Next() {
			id := string(iterator.Key())
			revision, err := decodeReactorCommissionRevision(id, iterator.Value())
			if err != nil {
				return err
			}
			records = append(records, types.EventReactorCommissionRevision{
				ReactorId: id, SlashRevision: revision.Issued, ReconciledSlashRevision: revision.Certified,
			})
		}
		return nil
	}(); err != nil {
		return err
	}
	events := sdk.UnwrapSDKContext(ctx).EventManager()
	if err := events.EmitTypedEvent(&types.EventReactorCommissionRevisionReset{}); err != nil {
		return err
	}
	for i := range records {
		if err := events.EmitTypedEvent(&records[i]); err != nil {
			return err
		}
	}
	return nil
}
