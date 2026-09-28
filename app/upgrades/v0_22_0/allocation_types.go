package v0_22_0

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/app/upgrades"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// MigrateLegacyAllocationTypes preserves the behavior of undeclared legacy
// modes: fixed power, no resizing hook and no agreement settlement. All declared
// modes are unchanged. Conflicting contract/hook evidence is not safe to guess.
// The phase has its own cache so a late conflict also refuses atomically when
// tested alone; the assembled upgrade supplies the outer transaction boundary.
func MigrateLegacyAllocationTypes(ctx context.Context, keepers *upgrades.Keepers) error {
	k := keepers.StructsKeeper
	var rows []types.Allocation
	ids := map[string]bool{}
	companions := map[string]bool{}
	for _, row := range k.GetAllAllocation(ctx) {
		if _, valid := types.AllocationType_name[int32(row.Type)]; valid {
			continue
		}
		if row.Id != keeperlib.GetObjectID(types.ObjectType_allocation, row.Index) {
			return fmt.Errorf("legacy allocation %s has inconsistent index %d", row.Id, row.Index)
		}
		rows = append(rows, row)
		ids[row.Id] = true
		companions[keeperlib.GetObjectID(types.ObjectType_agreement, row.Index)] = true
	}
	if len(rows) == 0 {
		return nil
	}
	// Audit the complete relation once, including a corrupt non-companion
	// agreement id. This is an upgrade-wide inventory, never a live-path scan.
	for _, agreement := range k.GetAllAgreement(ctx) {
		if ids[agreement.AllocationId] || companions[agreement.Id] {
			return fmt.Errorf("legacy allocation type conflicts with agreement %s", agreement.Id)
		}
	}
	cached, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	for _, row := range rows {
		if hook, found := k.GetAutoResizeAllocationBySource(cached, row.SourceObjectId); found && hook == row.Id {
			return fmt.Errorf("legacy allocation %s has an auto-resize hook", row.Id)
		}
		cc := k.NewCurrentContext(cached)
		allocation, found := cc.GetAllocation(row.Id)
		if !found {
			return fmt.Errorf("legacy allocation %s disappeared during repair", row.Id)
		}
		// This upgrade-only type repair deliberately touches no power, index,
		// permission or ID counter. Do not expose a live mode-changing setter.
		allocation.Allocation.Type = types.AllocationType_static
		allocation.Changed = true
		cc.CommitAll()
	}
	write()
	sdk.UnwrapSDKContext(ctx).Logger().Info("Normalized legacy allocation types", "upgrade", UpgradeName, "repaired", len(rows))
	return nil
}
