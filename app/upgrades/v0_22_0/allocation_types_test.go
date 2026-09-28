package v0_22_0_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMigrateLegacyAllocationTypesPreservesEverythingElse(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}
	modes := []types.AllocationType{0, 1, 2, 3, -2147483648, -1, 4, 500, 2147483647}
	var want []types.Allocation
	var changed []types.Allocation
	for i, mode := range modes {
		index := uint64(i + 1)
		row := types.Allocation{Id: fmt.Sprintf("6-%d", index), Index: index, Type: mode, Creator: "creator", Controller: "1-1", SourceObjectId: "4-1", DestinationId: "4-2"}
		k.ImportAllocation(ctx, row)
		k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, row.Id), index*10)
		if mode == types.AllocationType_automated {
			k.SetAutoResizeAllocationSource(ctx, row.Id, row.SourceObjectId)
		}
		if _, valid := types.AllocationType_name[int32(mode)]; !valid {
			row.Type = types.AllocationType_static
			changed = append(changed, row)
		}
		want = append(want, row)
	}
	k.SetAgreement(ctx, types.Agreement{Id: "11-4", AllocationId: "6-4"})
	k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes("6-5", "1-1"), types.PermAdmin|types.PermDelete)
	k.SetAllocationCount(ctx, 99)
	k.SetGridAttribute(ctx, "3-4-1", 450)
	k.SetGridAttribute(ctx, "2-4-2", 450)
	before := keepertest.StructsStoreRows(t, ctx)
	ctx = ctx.WithEventManager(sdk.NewEventManager()).WithGasMeter(storetypes.NewInfiniteGasMeter())
	require.NoError(t, v22.MigrateLegacyAllocationTypes(ctx, keepers))
	t.Logf("9 allocations, 5 normalized: %d gas", ctx.GasMeter().GasConsumed())
	for _, row := range want {
		actual, found := k.GetAllocation(ctx, row.Id)
		require.True(t, found)
		require.Equal(t, row, actual)
	}
	var emitted []types.Allocation
	for _, event := range ctx.EventManager().ABCIEvents() {
		require.Equal(t, "structs.structs.EventAllocation", event.Type)
		parsed, err := sdk.ParseTypedEvent(event)
		require.NoError(t, err)
		emitted = append(emitted, *parsed.(*types.EventAllocation).Allocation)
	}
	require.Equal(t, changed, emitted)
	after := keepertest.StructsStoreRows(t, ctx)
	for _, row := range changed {
		key := string(types.KeyPrefix(types.AllocationKey)) + row.Id
		delete(before, key)
		delete(after, key)
	}
	require.Equal(t, before, after, "only the five public allocation rows change; all indices, power, permissions and counters survive")
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	before = keepertest.StructsStoreRows(t, ctx)
	require.NoError(t, v22.MigrateLegacyAllocationTypes(ctx, keepers))
	require.Equal(t, before, keepertest.StructsStoreRows(t, ctx))
	require.Empty(t, ctx.EventManager().Events())
}

func TestMigrateLegacyAllocationTypesRefusesConflictsAtomically(t *testing.T) {
	for _, conflict := range []string{"agreement", "non-companion", "companion-other-allocation", "hook", "id"} {
		t.Run(conflict, func(t *testing.T) {
			k, ctx := keepertest.StructsKeeper(t)
			for i := uint64(1); i <= 2; i++ {
				k.ImportAllocation(ctx, types.Allocation{Id: fmt.Sprintf("6-%d", i), Index: i, Type: 500, SourceObjectId: fmt.Sprintf("4-%d", i), Controller: "1-1"})
			}
			switch conflict {
			case "agreement":
				k.SetAgreement(ctx, types.Agreement{Id: "11-2", AllocationId: "6-2"})
			case "non-companion":
				k.SetAgreement(ctx, types.Agreement{Id: "11-999", AllocationId: "6-2"})
			case "companion-other-allocation":
				k.SetAgreement(ctx, types.Agreement{Id: "11-2", AllocationId: "6-999"})
			case "hook":
				k.SetAutoResizeAllocationSource(ctx, "6-2", "4-2")
			case "id":
				k.ImportAllocation(ctx, types.Allocation{Id: "6-3", Index: 77, Type: 500})
			}
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			before := keepertest.StructsStoreRows(t, ctx)
			require.Error(t, v22.MigrateLegacyAllocationTypes(ctx, &upgrades.Keepers{StructsKeeper: k}))
			require.Equal(t, before, keepertest.StructsStoreRows(t, ctx))
			require.Empty(t, ctx.EventManager().Events())
		})
	}
}

func TestMigrateLegacyAllocationTypesGasRollbackAndInventorySize(t *testing.T) {
	for _, size := range []int{16, 256} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			k, ctx := keepertest.StructsKeeper(t)
			for i := 1; i <= size; i++ {
				k.ImportAllocation(ctx, types.Allocation{Id: fmt.Sprintf("6-%d", i), Index: uint64(i), Type: -1, SourceObjectId: "1-1", Controller: "1-1"})
			}
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			keepers := &upgrades.Keepers{StructsKeeper: k}
			before := keepertest.StructsStoreRows(t, ctx)
			trial, _ := ctx.CacheContext()
			trial = trial.WithGasMeter(storetypes.NewInfiniteGasMeter())
			require.NoError(t, v22.MigrateLegacyAllocationTypes(trial, keepers))
			cost := trial.GasMeter().GasConsumed()
			t.Logf("%d legacy rows: %d gas", size, cost)
			for _, budget := range []uint64{cost / 2, cost - 1} {
				limited := ctx.WithGasMeter(storetypes.NewGasMeter(budget))
				require.Panics(t, func() { _ = v22.MigrateLegacyAllocationTypes(limited, keepers) })
				require.Equal(t, before, keepertest.StructsStoreRows(t, ctx))
				require.Empty(t, ctx.EventManager().Events())
			}
		})
	}
}
