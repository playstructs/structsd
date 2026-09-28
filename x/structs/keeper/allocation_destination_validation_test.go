package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestAllocationDestinationRefusalPreservesCachedSupplyAndIndexes(t *testing.T) {
	f := setupAutoResizeFixture(t, 100)
	id := f.newAutomated(t)
	beforeGrid := f.k.GetAllGridExport(f.ctx)
	beforeQueue := f.k.GetGridCascadeQueueExport(f.ctx)
	beforeIndex := f.k.GetAllAllocationIdByDestinationIndex(f.ctx, f.destinationId)
	beforeAllocation, found := f.k.GetAllocation(f.ctx, id)
	require.True(t, found)

	cc := f.k.NewCurrentContext(f.ctx)
	allocation, found := cc.GetAllocation(id)
	require.True(t, found)
	capacity := cc.GetSubstation(f.destinationId).GetCapacity()
	require.Positive(t, capacity)
	require.ErrorContains(t, allocation.SetDestination("4-999999"), "unacceptable_destination")
	require.Equal(t, beforeAllocation, allocation.GetAllocation())
	require.Equal(t, capacity, cc.GetSubstation(f.destinationId).GetCapacity())
	require.Equal(t, beforeIndex, f.k.GetAllAllocationIdByDestinationIndex(f.ctx, f.destinationId))
	require.Equal(t, beforeQueue, f.k.GetGridCascadeQueueExport(f.ctx))
	rows, more, visited := cc.GetAllocationsByDestinationUpTo(f.destinationId, 10)
	require.False(t, more)
	require.Equal(t, len(beforeIndex), visited)
	require.Len(t, rows, 1)
	require.Same(t, allocation, rows[0])

	// A cache-level caller is allowed to continue after a refused change. Even
	// committing that same context must not persist an abandoned capacity cut.
	cc.CommitAll()
	require.Equal(t, beforeGrid, f.k.GetAllGridExport(f.ctx))
	require.Equal(t, beforeIndex, f.k.GetAllAllocationIdByDestinationIndex(f.ctx, f.destinationId))
	require.Empty(t, f.k.GetAllAllocationIdByDestinationIndex(f.ctx, "4-999999"))
	stored, found := f.k.GetAllocation(f.ctx, id)
	require.True(t, found)
	require.Equal(t, beforeAllocation, stored)

	// An empty destination is an intentional disconnect, not an invalid id.
	cc = f.k.NewCurrentContext(f.ctx)
	allocation, found = cc.GetAllocation(id)
	require.True(t, found)
	require.NoError(t, allocation.SetDestination(""))
	cc.CommitAll()
	require.Empty(t, f.k.GetAllAllocationIdByDestinationIndex(f.ctx, f.destinationId))
	require.Zero(t, f.k.NewCurrentContext(f.ctx).GetSubstation(f.destinationId).GetCapacity())
	stored, found = f.k.GetAllocation(f.ctx, id)
	require.True(t, found)
	require.Empty(t, stored.DestinationId)
	require.Equal(t, types.AllocationType_automated, stored.Type)
}

func TestNewAllocationRejectsMissingDestinationBeforeReservingSource(t *testing.T) {
	for _, kind := range []types.AllocationType{types.AllocationType_automated, types.AllocationType_dynamic} {
		t.Run(kind.String(), func(t *testing.T) {
			f := setupAutoResizeFixture(t, 100)
			beforeGrid := f.k.GetAllGridExport(f.ctx)
			beforeCount := f.k.GetAllocationCount(f.ctx)
			beforeIndex := f.k.GetAllAllocationIdBySourceIndex(f.ctx, f.sourceId)
			cc := f.k.NewCurrentContext(f.ctx)
			_, err := cc.NewAllocation(kind, f.sourceId, "4-999999", f.owner.Creator, f.owner.Id, 10)
			require.ErrorContains(t, err, "unacceptable_destination")
			cc.CommitAll()
			require.Equal(t, beforeCount, f.k.GetAllocationCount(f.ctx))
			require.Equal(t, beforeGrid, f.k.GetAllGridExport(f.ctx))
			require.Equal(t, beforeIndex, f.k.GetAllAllocationIdBySourceIndex(f.ctx, f.sourceId))
			require.Empty(t, f.k.GetAllAllocationIdByDestinationIndex(f.ctx, "4-999999"))
			_, found := f.hook(t)
			require.False(t, found, "refused automated creation must not reserve the source hook")
		})
	}
}

func TestNewAllocationRefusalPreservesCounterIndexesAndCachedAllocations(t *testing.T) {
	for _, scenario := range []string{"zero_power", "excess_power", "automated_conflict"} {
		t.Run(scenario, func(t *testing.T) {
			f := setupAutoResizeFixture(t, 100)
			if scenario == "automated_conflict" {
				f.newAutomated(t)
			}
			kind, power := types.AllocationType_dynamic, uint64(0)
			if scenario == "excess_power" {
				power = 101
			} else if scenario == "automated_conflict" {
				kind = types.AllocationType_automated
			}
			beforeCount := f.k.GetAllocationCount(f.ctx)
			beforeGrid := f.k.GetAllGridExport(f.ctx)
			beforeQueue := f.k.GetGridCascadeQueueExport(f.ctx)
			beforeSource := f.k.GetAllAllocationIdBySourceIndex(f.ctx, f.sourceId)
			beforeDestination := f.k.GetAllAllocationIdByDestinationIndex(f.ctx, f.destinationId)
			beforeAllocations := f.k.GetAllAllocation(f.ctx)
			beforeHook, beforeHookFound := f.hook(t)
			cc := f.k.NewCurrentContext(f.ctx)
			_, err := cc.NewAllocation(kind, f.sourceId, f.destinationId, f.owner.Creator, f.owner.Id, power)
			require.Error(t, err)
			cc.CommitAll()
			require.Equal(t, beforeCount, f.k.GetAllocationCount(f.ctx))
			require.Equal(t, beforeGrid, f.k.GetAllGridExport(f.ctx))
			require.Equal(t, beforeQueue, f.k.GetGridCascadeQueueExport(f.ctx))
			require.Equal(t, beforeSource, f.k.GetAllAllocationIdBySourceIndex(f.ctx, f.sourceId))
			require.Equal(t, beforeDestination, f.k.GetAllAllocationIdByDestinationIndex(f.ctx, f.destinationId))
			require.Equal(t, beforeAllocations, f.k.GetAllAllocation(f.ctx), "a refused creation cannot publish a changed orphan cache")
			hook, hookFound := f.hook(t)
			require.Equal(t, beforeHookFound, hookFound)
			require.Equal(t, beforeHook, hook)
			if scenario != "automated_conflict" {
				cc = f.k.NewCurrentContext(f.ctx)
				_, err = cc.NewAllocation(kind, f.sourceId, f.destinationId, f.owner.Creator, f.owner.Id, power)
				require.Error(t, err)
				created, err := cc.NewAllocation(types.AllocationType_dynamic, f.sourceId, f.destinationId, f.owner.Creator, f.owner.Id, 10)
				require.NoError(t, err, "the same context can create a valid allocation after refusal")
				require.Equal(t, beforeCount, created.GetAllocation().Index)
				cc.CommitAll()
				require.Equal(t, beforeCount+1, f.k.GetAllocationCount(f.ctx))
				require.Equal(t, uint64(10), created.GetPower())
				stored, found := f.k.GetAllocation(f.ctx, created.ID())
				require.True(t, found)
				require.Equal(t, created.GetAllocation(), stored)
			}
		})
	}
}

func TestNewAllocationExhaustionNeverReusesAgreementIdentities(t *testing.T) {
	f := setupAutoResizeFixture(t, 100)
	f.k.SetAllocationCount(f.ctx, ^uint64(0))
	beforeGrid := f.k.GetAllGridExport(f.ctx)
	beforeRows := f.k.GetAllAllocation(f.ctx)
	cc := f.k.NewCurrentContext(f.ctx)
	_, err := cc.NewAllocation(types.AllocationType_dynamic, f.sourceId, f.destinationId, f.owner.Creator, f.owner.Id, 10)
	require.ErrorContains(t, err, "allocation_id_exhausted")
	cc.CommitAll()
	require.Equal(t, ^uint64(0), f.k.GetAllocationCount(f.ctx))
	require.Equal(t, beforeGrid, f.k.GetAllGridExport(f.ctx))
	require.Equal(t, beforeRows, f.k.GetAllAllocation(f.ctx))
}
