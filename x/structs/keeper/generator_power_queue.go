package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

const generatorPowerQueuePrefix = "Generator/powerQueue/"
const generatorPowerPositionKey = "Generator/powerPosition"

type GeneratorPowerSnapshot struct{ After, Through uint64 }

func (k Keeper) generatorPowerStore(ctx context.Context) prefix.Store {
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte(generatorPowerQueuePrefix))
}

func decodeGeneratorPowerSnapshot(id string, value []byte) (GeneratorPowerSnapshot, error) {
	if len(value) != 16 {
		return GeneratorPowerSnapshot{}, fmt.Errorf("invalid generator power snapshot for %s", id)
	}
	snapshot := GeneratorPowerSnapshot{After: binary.BigEndian.Uint64(value[:8]), Through: binary.BigEndian.Uint64(value[8:])}
	if snapshot.After > snapshot.Through {
		return snapshot, fmt.Errorf("invalid generator power interval for %s", id)
	}
	return snapshot, nil
}

func (k Keeper) GetGeneratorPowerSnapshot(ctx context.Context, id string) (GeneratorPowerSnapshot, bool, error) {
	value := k.generatorPowerStore(ctx).Get([]byte(id))
	if value == nil {
		return GeneratorPowerSnapshot{}, false, nil
	}
	snapshot, err := decodeGeneratorPowerSnapshot(id, value)
	return snapshot, true, err
}

func (k Keeper) setGeneratorPowerSnapshot(ctx context.Context, id string, snapshot GeneratorPowerSnapshot) {
	value := make([]byte, 16)
	binary.BigEndian.PutUint64(value[:8], snapshot.After)
	binary.BigEndian.PutUint64(value[8:], snapshot.Through)
	k.generatorPowerStore(ctx).Set([]byte(id), value)
}

// EnqueueGeneratorPower is called after infusion commits, so the cutoff includes
// positions created in the same CurrentContext as the lifecycle transition.
// A real new transition resets the interval; later ordinary deposits do not.
func (k Keeper) EnqueueGeneratorPower(ctx context.Context, id string) {
	if !ObjectIdHasType(id, types.ObjectType_struct) {
		panic("invalid generator power destination")
	}
	k.setGeneratorPowerSnapshot(ctx, id, GeneratorPowerSnapshot{Through: k.ReactorInfusionOrderEnd(ctx, id)})
}

// Full listing is for genesis/export and diagnostics, never ordinary block work.
func (k Keeper) GetGeneratorPowerQueue(ctx context.Context) ([]string, error) {
	iterator := k.generatorPowerStore(ctx).Iterator(nil, nil)
	defer iterator.Close()
	var ids []string
	for ; iterator.Valid(); iterator.Next() {
		id := string(iterator.Key())
		if _, err := decodeGeneratorPowerSnapshot(id, iterator.Value()); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (k Keeper) readRotatingGeneratorPowerQueue(ctx context.Context, limit int) []string {
	root := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	position := root.Get([]byte(generatorPowerPositionKey))
	store := k.generatorPowerStore(ctx)
	var ids []string
	read := func(start, end []byte) {
		iterator := store.Iterator(start, end)
		defer iterator.Close()
		for ; iterator.Valid() && len(ids) < limit; iterator.Next() {
			ids = append(ids, string(iterator.Key()))
		}
	}
	if len(position) == 0 {
		read(nil, nil)
		return ids
	}
	after := append(append([]byte(nil), position...), 0)
	read(after, nil)
	if len(ids) < limit {
		read(nil, after)
	}
	return ids
}

// One block reads at most 128 jobs and processes a shared 128 infusion/allocation
// rows. Empty or malformed jobs cannot cause a full queue scan. Rotation prevents
// repeated transitions or a broken first job from starving another generator.
func (k Keeper) ProcessGeneratorPowerQueue(ctx context.Context) {
	ids := k.readRotatingGeneratorPowerQueue(ctx, types.GeneratorPowerReconcileBudget)
	if len(ids) == 0 {
		return
	}
	cc := k.NewCurrentContext(ctx)
	budget := types.GeneratorPowerReconcileBudget
	type progress struct {
		id       string
		snapshot GeneratorPowerSnapshot
		done     bool
	}
	var changes []progress
	last := ""
	for _, id := range ids {
		if budget <= 0 {
			break
		}
		last = id
		snapshot, found, err := k.GetGeneratorPowerSnapshot(ctx, id)
		if err != nil || !found {
			k.logger.Error("Generator power snapshot invalid", "generator", id, "error", err)
			continue
		}
		structure := cc.GetStruct(id)
		destroyed := structure.CheckStruct() != nil || structure.IsDestroyed()
		enabled := !destroyed && structure.IsBuilt() && structure.IsOnline()
		positions, more, err := k.ReactorInfusionOrderBatch(ctx, id, snapshot.After, snapshot.Through, budget)
		if err != nil {
			budget = 0
			k.logger.Error("Generator power inventory invalid", "generator", id, "error", err)
			continue
		}
		failed := false
		retryAfter := uint64(0)
		for _, position := range positions {
			budget--
			infusion := cc.GetInfusion(id, position.Address)
			err := k.reconcileGeneratorPosition(cc, infusion, destroyed, enabled)
			if err != nil {
				if !failed {
					retryAfter = position.Ordinal - 1
				}
				failed = true
				k.logger.Error("Generator power position failed", "generator", id, "address", position.Address, "error", err)
			}
			snapshot.After = position.Ordinal
		}
		if failed {
			snapshot.After = retryAfter
		}
		// Direct-generator allocations are inadmissible even after recovery.
		// Read raw scoped rows so stale entries are removed under their original
		// source key and cannot indefinitely hide live allocations behind them.
		allocationIDs, moreAllocations := k.GetAllocationIdsBySourceIndexUpTo(ctx, id, budget)
		budget -= len(allocationIDs)
		for _, allocationID := range allocationIDs {
			allocation, found := cc.GetAllocation(allocationID)
			if !found || allocation.GetAllocation().SourceObjectId != id {
				_ = k.RemoveAllocationSourceIndex(ctx, id, allocationID)
				continue
			}
			if err := allocation.Destroy(); err != nil {
				moreAllocations = true
				k.logger.Error("Generator allocation teardown failed", "generator", id, "allocation", allocation.ID(), "error", err)
			}
		}
		done := !failed && !more && !moreAllocations
		if done && destroyed {
			for _, attribute := range []types.GridAttributeType{types.GridAttributeType_load, types.GridAttributeType_capacity, types.GridAttributeType_fuel, types.GridAttributeType_power} {
				cc.ClearGridAttribute(GetGridAttributeIDByObjectId(attribute, id))
			}
		}
		changes = append(changes, progress{id: id, snapshot: snapshot, done: done})
	}
	// Accounting and its public events are committed before advancing a cursor.
	cc.CommitAll()
	for _, change := range changes {
		if change.done {
			k.generatorPowerStore(ctx).Delete([]byte(change.id))
		} else {
			k.setGeneratorPowerSnapshot(ctx, change.id, change.snapshot)
		}
	}
	if last != "" {
		runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)).Set([]byte(generatorPowerPositionKey), []byte(last))
	}
}

func (k Keeper) reconcileGeneratorPosition(cc *CurrentContext, infusion *InfusionCache, destroyed, enabled bool) error {
	if err := infusion.CheckInfusion(); err != nil {
		return err
	}
	row := infusion.GetInfusion()
	if err := row.ValidateGeneratorPower(); err != nil {
		return err
	}
	if row.DestinationId != infusion.DestinationId || row.Address != infusion.Address {
		return fmt.Errorf("generator infusion key mismatch")
	}
	if destroyed {
		infusion.Destroy()
		return nil
	}
	address, err := sdk.AccAddressFromBech32(row.Address)
	if err != nil {
		return err
	}
	owner, ownerErr := cc.GetAssetOwnerByAddress(address)
	if ownerErr != nil {
		return infusion.SetGeneratorPowerEnabled(false)
	}
	if owner.ID() != row.AssetOwnerID() {
		// Re-home while disabled so intermediate ownership never briefly
		// credits power to the new owner on a source which remains offline.
		if err := infusion.SetGeneratorPowerEnabled(false); err != nil {
			return err
		}
		if err := infusion.SetOwnerId(owner.ID()); err != nil {
			return err
		}
	}
	return infusion.SetGeneratorPowerEnabled(enabled && cc.generatorInfusionEligible(infusion.GetInfusion()))
}

// Upgrade/genesis-only repair for older snapshots without generator jobs. It
// also rebuilds their finite cutoffs after the insertion index was rebuilt.
func (k Keeper) RebuildGeneratorPowerQueue(ctx context.Context) {
	cc := k.NewCurrentContext(ctx)
	seen := map[string]bool{}
	for _, row := range k.GetAllInfusion(ctx) {
		if row.DestinationType != types.ObjectType_struct || seen[row.DestinationId] {
			continue
		}
		seen[row.DestinationId] = true
		k.EnqueueGeneratorPower(ctx, row.DestinationId)
	}
	// Generator objects may still have old allocations even without an infusion.
	for _, row := range k.GetAllStruct(ctx) {
		if seen[row.Id] {
			continue
		}
		structure := cc.GetStruct(row.Id)
		if structure.GetStructType().HasPowerGenerationSystem() && (!structure.IsOnline() || structure.IsDestroyed()) {
			seen[row.Id] = true
			k.EnqueueGeneratorPower(ctx, row.Id)
		}
	}
	// A swept generator may have no object or infusion left to identify its
	// old direct allocations. Only upgrade/import performs this full scan.
	for _, row := range k.GetAllAllocation(ctx) {
		if ObjectIdHasType(row.SourceObjectId, types.ObjectType_struct) && !seen[row.SourceObjectId] {
			seen[row.SourceObjectId] = true
			k.EnqueueGeneratorPower(ctx, row.SourceObjectId)
		}
	}
}
