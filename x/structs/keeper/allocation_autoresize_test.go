package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Regression tests for the auto-resize hook that was cleared under the wrong key.
//
// The index maps a SOURCE object id to the automated allocation riding on it:
// SetAutoResizeAllocationSource does store.Set([]byte(sourceObjectId), []byte(allocationId)).
// AllocationCache.Destroy cleared it with cache.ID(), the allocation's own id, so
// the delete matched no key and the real entry outlived the allocation it named.
// The line immediately below it, RemoveAllocationSourceIndex, uses SourceObjectId
// correctly, which is what makes this a typo rather than a design.
//
// A leaked hook is not inert. SetSource refuses a new automated allocation on any
// source the index already mentions, without checking that the allocation still
// exists, so the source is bricked for good. And the infusion capacity path treats
// the hook as live: it calls AutoResizeAllocation on a missing allocation, which
// did nothing at all, and in doing so skips the branch that would have queued the
// source for grid cascade — so a capacity cut silently stopped shedding load.
//
// Automated allocations cannot be removed through MsgAllocationDelete, which only
// accepts dynamic ones. They reach Destroy through GridCascade, substation
// deletion, and power-generating struct offline or destruction.

// autoResizeFixture is a source object with capacity to give and a substation to
// give it to.
type autoResizeFixture struct {
	k     keeperlib.Keeper
	ctx   sdk.Context
	owner types.Player

	sourceId      string
	destinationId string
}

func setupAutoResizeFixture(t *testing.T, sourceCapacity uint64) *autoResizeFixture {
	t.Helper()

	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	ownerAcc := sdk.AccAddress("autoresize1234567890123456789012345")
	owner := testAppendPlayer(k, ctx, types.Player{
		Creator:        ownerAcc.String(),
		PrimaryAddress: ownerAcc.String(),
	})

	substation, _, err := testAppendSubstation(k, ctx, types.Allocation{}, owner)
	require.NoError(t, err)

	// An automated allocation takes its power from the source's capacity, so the
	// source needs some or there is nothing to allocate.
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner.Id), sourceCapacity)

	return &autoResizeFixture{
		k:             k,
		ctx:           ctx,
		owner:         owner,
		sourceId:      owner.Id,
		destinationId: substation.Id,
	}
}

// newAutomated creates an automated allocation the way production does, through
// NewAllocation and so through SetSource, which is what actually writes the
// index. testAppendAllocation goes through ImportAllocation and never touches it,
// which is why the existing allocation tests never saw this bug.
func (f *autoResizeFixture) newAutomated(t *testing.T) string {
	t.Helper()

	cc := f.k.NewCurrentContext(f.ctx)
	allocation, err := cc.NewAllocation(
		types.AllocationType_automated,
		f.sourceId,
		f.destinationId,
		f.owner.Creator,
		f.owner.Id,
		0,
	)
	require.NoError(t, err)
	cc.CommitAll()

	return allocation.ID()
}

func (f *autoResizeFixture) hook(t *testing.T) (string, bool) {
	t.Helper()
	return f.k.GetAutoResizeAllocationBySource(f.ctx, f.sourceId)
}

func (f *autoResizeFixture) destroy(t *testing.T, allocationId string) {
	t.Helper()

	cc := f.k.NewCurrentContext(f.ctx)
	allocation, found := cc.GetAllocation(allocationId)
	require.True(t, found)
	require.NoError(t, allocation.Destroy())
	cc.CommitAll()
}

// TestAutoResize_HookIsClearedOnDestroy is the direct regression: the index entry
// must not outlive the allocation it points at.
func TestAutoResize_HookIsClearedOnDestroy(t *testing.T) {
	f := setupAutoResizeFixture(t, 500)

	allocationId := f.newAutomated(t)

	// The hook is keyed by source, not by allocation. Assert that before relying
	// on it, so this test cannot pass by looking up the wrong key.
	hooked, found := f.hook(t)
	require.True(t, found, "creating an automated allocation must register the source hook")
	require.Equal(t, allocationId, hooked)
	require.NotEqual(t, allocationId, f.sourceId,
		"the allocation id and the source id must differ, or the wrong-key delete would coincidentally work")

	f.destroy(t, allocationId)

	_, stillHooked := f.hook(t)
	require.False(t, stillHooked,
		"the source hook outlived its allocation; it now names an allocation that no longer exists")
}

// TestAutoResize_SourceAcceptsReplacementAfterDestroy is the consequence a player
// would actually report: the source is permanently unusable for automated
// allocations, with no way to clear it short of a state migration.
func TestAutoResize_SourceAcceptsReplacementAfterDestroy(t *testing.T) {
	f := setupAutoResizeFixture(t, 500)

	first := f.newAutomated(t)
	f.destroy(t, first)

	cc := f.k.NewCurrentContext(f.ctx)
	replacement, err := cc.NewAllocation(
		types.AllocationType_automated,
		f.sourceId,
		f.destinationId,
		f.owner.Creator,
		f.owner.Id,
		0,
	)
	require.NoError(t, err,
		"a source whose automated allocation was destroyed must accept a new one; a stale hook rejects it as automated_conflict")
	cc.CommitAll()

	hooked, found := f.hook(t)
	require.True(t, found)
	require.Equal(t, replacement.ID(), hooked, "the hook must now name the replacement")
	require.NotEqual(t, first, hooked)
}

// TestAutoResize_StaleHookFallsThroughToCascade covers the second consequence,
// which is the expensive one.
//
// A capacity cut has two possible responses: an automated allocation tracking
// that source resizes to match, or, if nothing is tracking it, the source is
// queued for grid cascade so the excess load gets shed. A stale hook used to
// select the first and then do nothing, because AutoResizeAllocation silently
// returned on an allocation it could not find — so the cut shed no load and the
// source stayed over-subscribed with no record of it.
//
// After the key fix a stale hook should not arise, so this writes one directly.
// That is the point: this is the guard for the next one, from any cause.
func TestAutoResize_StaleHookFallsThroughToCascade(t *testing.T) {
	f := setupAutoResizeFixture(t, 0)

	const ratio, fuel = 10, 100
	commission := math.LegacyMustNewDecFromStr("0.5")

	// power = ratio*fuel = 1000, split evenly by the commission into 500 of
	// destination capacity and 500 of player capacity.
	infusion := types.CreateNewInfusion(
		types.ObjectType_substation,
		f.destinationId,
		f.owner.PrimaryAddress,
		f.owner.Id,
		fuel,
		commission,
		ratio,
	)
	testAppendInfusion(f.k, f.ctx, infusion)

	destinationCapacityId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, f.destinationId)
	playerCapacityId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, f.owner.Id)
	f.k.SetGridAttribute(f.ctx, destinationCapacityId, 500)
	f.k.SetGridAttribute(f.ctx, playerCapacityId, 500)

	// Hooks on both the destination and the player, each naming an allocation
	// that does not exist.
	f.k.SetAutoResizeAllocationSource(f.ctx, "6-98", f.destinationId)
	f.k.SetAutoResizeAllocationSource(f.ctx, "6-99", f.owner.Id)

	require.Empty(t, f.k.GetGridCascadeQueueExport(f.ctx))

	// Halve the fuel, which halves both capacities.
	cc := f.k.NewCurrentContext(f.ctx)
	cc.GetInfusion(f.destinationId, f.owner.PrimaryAddress).SetFuel(fuel / 2)
	cc.CommitAll()

	require.Equal(t, uint64(250), f.k.GetGridAttribute(f.ctx, destinationCapacityId), "the capacity cut must have landed")
	require.Equal(t, uint64(250), f.k.GetGridAttribute(f.ctx, playerCapacityId))

	queue := f.k.GetGridCascadeQueueExport(f.ctx)
	require.Contains(t, queue, f.destinationId,
		"a capacity cut with nothing tracking it must shed load; the stale hook swallowed the cascade")
	require.Contains(t, queue, f.owner.Id,
		"the player capacity cut must queue a cascade too")

	_, destinationHooked := f.k.GetAutoResizeAllocationBySource(f.ctx, f.destinationId)
	require.False(t, destinationHooked, "a hook naming a missing allocation must be dropped, not left to mislead the next cut")
	_, playerHooked := f.k.GetAutoResizeAllocationBySource(f.ctx, f.owner.Id)
	require.False(t, playerHooked)
}

// TestAutoResize_LiveHookResizesAndSkipsCascade is the control for the test
// above. A hook naming a live allocation must still resize it and must NOT queue
// a cascade, or the fall-through has turned into a permanent load-shedder.
func TestAutoResize_LiveHookResizesAndSkipsCascade(t *testing.T) {
	f := setupAutoResizeFixture(t, 500)

	const ratio, fuel = 10, 100
	commission := math.LegacyMustNewDecFromStr("0.5")

	infusion := types.CreateNewInfusion(
		types.ObjectType_substation,
		f.destinationId,
		f.owner.PrimaryAddress,
		f.owner.Id,
		fuel,
		commission,
		ratio,
	)
	testAppendInfusion(f.k, f.ctx, infusion)

	destinationCapacityId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, f.destinationId)
	f.k.SetGridAttribute(f.ctx, destinationCapacityId, 500)

	// A real automated allocation drawing on the destination.
	cc := f.k.NewCurrentContext(f.ctx)
	tracking, err := cc.NewAllocation(
		types.AllocationType_automated,
		f.destinationId,
		"",
		f.owner.Creator,
		f.owner.Id,
		0,
	)
	require.NoError(t, err)
	cc.CommitAll()

	f.k.SetGridAttribute(f.ctx, destinationCapacityId, 500)

	cc2 := f.k.NewCurrentContext(f.ctx)
	cc2.GetInfusion(f.destinationId, f.owner.PrimaryAddress).SetFuel(fuel / 2)
	cc2.CommitAll()

	hooked, found := f.k.GetAutoResizeAllocationBySource(f.ctx, f.destinationId)
	require.True(t, found, "a live hook must survive a capacity change")
	require.Equal(t, tracking.ID(), hooked)

	require.NotContains(t, f.k.GetGridCascadeQueueExport(f.ctx), f.destinationId,
		"an allocation is tracking this source, so there is nothing to shed")
}

// TestAutoResize_ConflictStillRejectedWhileLive is the other half. The clear must
// not become a licence to stack automated allocations on one source: while the
// first is alive, a second is still a conflict.
func TestAutoResize_ConflictStillRejectedWhileLive(t *testing.T) {
	f := setupAutoResizeFixture(t, 500)

	first := f.newAutomated(t)

	cc := f.k.NewCurrentContext(f.ctx)
	_, err := cc.NewAllocation(
		types.AllocationType_automated,
		f.sourceId,
		f.destinationId,
		f.owner.Creator,
		f.owner.Id,
		0,
	)
	require.Error(t, err, "a source already carrying a live automated allocation must reject a second")

	hooked, found := f.hook(t)
	require.True(t, found)
	require.Equal(t, first, hooked, "the rejected create must not have overwritten the live hook")
}
