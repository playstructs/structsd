package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

/* Regression suite for a destroyed allocation coming back as a live cache hit.
 *
 * Destroy defers both removals to CommitAll: the cache stays in cc.allocations
 * and the record stays in the store. GetAllocation's cache-hit path returned
 * that cache with found=true, and every caller reads the boolean as existence -
 * the same shape as the phantom PlayerCache, where a context getter is a cache
 * allocator and says nothing about whether the object is real.
 *
 * What made it damaging rather than merely untidy is that Destroy never zeroes
 * the power attribute. GetPower reads the grid, so a second Destroy sees the
 * original power and takes it off the source's load again - and that load
 * belongs to whichever allocations are still sharing the source. The source then
 * reports headroom it does not have, which is the direction that oversubscribes
 * a grid rather than the direction that merely wastes it.
 */

// deletedAllocationFixture puts two allocations on one source, so a stray second
// decrement has somebody else's load to eat.
type deletedAllocationFixture struct {
	f        *autoResizeFixture
	victimId string
	srcLoad  string
	destCap  string
}

func setupDeletedAllocationFixture(t *testing.T) *deletedAllocationFixture {
	t.Helper()

	f := setupAutoResizeFixture(t, 500)

	cc := f.k.NewCurrentContext(f.ctx)
	owner := cc.GetPlayer(f.owner.Id)

	victim, err := cc.NewAllocation(types.AllocationType_dynamic, f.sourceId, f.destinationId, f.owner.Creator, owner.ID(), 100)
	require.NoError(t, err)
	_, err = cc.NewAllocation(types.AllocationType_dynamic, f.sourceId, f.destinationId, f.owner.Creator, owner.ID(), 150)
	require.NoError(t, err)
	cc.CommitAll()

	return &deletedAllocationFixture{
		f:        f,
		victimId: victim.ID(),
		srcLoad:  keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, f.sourceId),
		destCap:  keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, f.destinationId),
	}
}

// TestDeletedAllocation_IsNotFoundInTheSameContext is the direct regression.
func TestDeletedAllocation_IsNotFoundInTheSameContext(t *testing.T) {
	d := setupDeletedAllocationFixture(t)

	cc := d.f.k.NewCurrentContext(d.f.ctx)
	allocation, found := cc.GetAllocation(d.victimId)
	require.True(t, found, "fixture sanity: the allocation exists before it is destroyed")
	require.NoError(t, allocation.Destroy())

	_, foundAgain := cc.GetAllocation(d.victimId)
	require.False(t, foundAgain,
		"a destroyed allocation must not come back as a live cache hit; every caller reads this as existence")
}

/* TestDeletedAllocation_SecondDestroyDoesNotEatSurvivingLoad is the consequence.
 * 250 of load, 100 of it destroyed, must leave 150 - not 50.
 */
func TestDeletedAllocation_SecondDestroyDoesNotEatSurvivingLoad(t *testing.T) {
	d := setupDeletedAllocationFixture(t)

	cc := d.f.k.NewCurrentContext(d.f.ctx)
	require.Equal(t, uint64(250), cc.GetGridAttribute(d.srcLoad), "fixture sanity")

	allocation, _ := cc.GetAllocation(d.victimId)
	require.NoError(t, allocation.Destroy())
	require.Equal(t, uint64(150), cc.GetGridAttribute(d.srcLoad))

	// The stale handle a caller could still be holding.
	require.NoError(t, allocation.Destroy(), "destroying twice is a no-op, not a failure")
	require.Equal(t, uint64(150), cc.GetGridAttribute(d.srcLoad),
		"the second destroy took the surviving allocation's load with it")

	cc.CommitAll()
	require.Equal(t, uint64(150), d.f.k.GetGridAttribute(d.f.ctx, d.srcLoad))
}

// TestDeletedAllocation_SetPowerIsRejected covers the mutator the auto-resize
// path reaches. previousPower is still the pre-destroy value, so any delta
// computed from it moves the grid by an amount Destroy already removed.
func TestDeletedAllocation_SetPowerIsRejected(t *testing.T) {
	d := setupDeletedAllocationFixture(t)

	cc := d.f.k.NewCurrentContext(d.f.ctx)
	allocation, _ := cc.GetAllocation(d.victimId)
	require.NoError(t, allocation.Destroy())

	_, err := allocation.SetPower(40)
	require.Error(t, err, "resizing a destroyed allocation must be refused")
	require.Equal(t, uint64(150), cc.GetGridAttribute(d.srcLoad),
		"a refused resize must not have moved the source's load")
}

/* TestDeletedAllocation_AutoResizeReportsStaleHook ties it to the caller that
 * cares about the distinction. AutoResizeAllocation separates "allocation
 * missing", which means the hook is stale and the caller must fall back to
 * shedding load, from "resize failed", which must not shed. A destroyed
 * allocation is the first of those.
 */
func TestDeletedAllocation_AutoResizeReportsStaleHook(t *testing.T) {
	d := setupDeletedAllocationFixture(t)

	cc := d.f.k.NewCurrentContext(d.f.ctx)
	allocation, _ := cc.GetAllocation(d.victimId)
	require.NoError(t, allocation.Destroy())

	require.False(t, cc.AutoResizeAllocation(d.victimId, 40),
		"a destroyed allocation is a stale hook, not a live one that failed to resize")
}

// TestDeletedAllocation_DestroyMultipleSkipsRepeats covers the teardown helper,
// which takes a list of ids and must treat a repeat as already done.
func TestDeletedAllocation_DestroyMultipleSkipsRepeats(t *testing.T) {
	d := setupDeletedAllocationFixture(t)

	cc := d.f.k.NewCurrentContext(d.f.ctx)
	cc.DestroyMultipleAllocations([]string{d.victimId, d.victimId})

	require.Equal(t, uint64(150), cc.GetGridAttribute(d.srcLoad),
		"a duplicated id in a teardown list must not tear down twice")
}
