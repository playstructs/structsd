package keeper_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

/* Regression suite for the unbounded grid cascade.
 *
 * GridCascade used to drain its queue to exhaustion inside the EndBlocker,
 * which runs against an infinite gas meter. The graph it walks is attacker
 * sized and costs almost nothing to build: an allocation of one power feeds a
 * substation, that substation allocates the same one power onward, and two free
 * messages add a link. Destroying the root allocation then collapsed the entire
 * chain in a single block - deterministically, so every retry of that block did
 * the same thing, and the chain stayed down until operators intervened in state.
 *
 * The fix is a per-block work budget with the remainder carried forward, and a
 * FIFO queue so the carry-over cannot be gamed. Both halves are load-bearing and
 * are tested separately: the budget alone would turn the halt into a way to keep
 * a substation over-subscribed forever by always keeping cheaper entries in
 * front of it.
 */

type cascadeChain struct {
	k     keeperlib.Keeper
	ctx   sdk.Context
	owner types.Player

	rootAllocationId string
	substationIds    []string
}

/* buildCascadeChain builds the attack topology: one unit of power threaded
 * through links substations in series.
 *
 * Each link is exactly what a player can do with two ordinary messages -
 * AllocationCreate against the previous substation as source, then
 * SubstationCreate on that allocation - so nothing here needs a permission or a
 * balance the attacker would not have.
 */
func buildCascadeChain(t *testing.T, links int) *cascadeChain {
	t.Helper()

	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	ownerAcc := sdk.AccAddress("cascadechain12345678901234567890123")
	owner := testAppendPlayer(k, ctx, types.Player{
		Creator:        ownerAcc.String(),
		PrimaryAddress: ownerAcc.String(),
	})

	// The player is the root source and needs the one unit that travels the
	// whole chain.
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner.Id), 1)

	chain := &cascadeChain{k: k, ctx: ctx, owner: owner}

	sourceId := owner.Id
	for i := 0; i < links; i++ {
		cc := k.NewCurrentContext(ctx)
		ownerCache := cc.GetPlayer(owner.Id)

		allocation, err := cc.NewAllocation(
			types.AllocationType_dynamic,
			sourceId,
			"",
			owner.Creator,
			owner.Id,
			1,
		)
		require.NoError(t, err, "link %d allocation", i)

		substation, err := cc.NewSubstation(owner.Creator, ownerCache, allocation)
		require.NoError(t, err, "link %d substation", i)

		cc.CommitAll()

		if i == 0 {
			chain.rootAllocationId = allocation.ID()
		}
		chain.substationIds = append(chain.substationIds, substation.ID())
		sourceId = substation.ID()
	}

	// Nothing should be queued yet: building the chain only ever adds capacity.
	require.Zero(t, k.GetGridCascadeQueueLength(ctx), "building the chain must not queue a cascade")

	return chain
}

// pullTheRoot destroys the root allocation, which is the single cheap action
// that used to collapse the whole chain inside one block.
func (c *cascadeChain) pullTheRoot(t *testing.T) {
	t.Helper()

	cc := c.k.NewCurrentContext(c.ctx)
	allocation, found := cc.GetAllocation(c.rootAllocationId)
	require.True(t, found)
	require.NoError(t, allocation.Destroy())
	cc.CommitAll()

	require.NotZero(t, c.k.GetGridCascadeQueueLength(c.ctx),
		"destroying the root must queue the first substation")
}

// cascadeBlock runs one block's worth of cascade and returns how many
// substations are still over-subscribed.
func (c *cascadeChain) cascadeBlock(t *testing.T) int {
	t.Helper()

	cc := c.k.NewCurrentContext(c.ctx)
	cc.GridCascade()
	cc.CommitAll()

	return c.k.GetGridCascadeQueueLength(c.ctx)
}

func (c *cascadeChain) liveAllocations(t *testing.T) int {
	t.Helper()

	// Destroy removes the record, so what is left in the store is what is left
	// of the chain.
	return len(c.k.GetAllAllocation(c.ctx))
}

/* TestGridCascade_LongChainCannotBeCollapsedInOneBlock is the halt regression.
 *
 * The chain is longer than the budget, so one block must not finish it. Before
 * the budget existed this single call walked every link.
 */
func TestGridCascade_LongChainCannotBeCollapsedInOneBlock(t *testing.T) {
	links := types.GridCascadeBlockBudget + 40
	chain := buildCascadeChain(t, links)

	before := chain.liveAllocations(t)
	require.Equal(t, links, before, "each link is one allocation")

	chain.pullTheRoot(t)

	remaining := chain.cascadeBlock(t)

	destroyed := before - chain.liveAllocations(t)
	require.LessOrEqual(t, destroyed, types.GridCascadeBlockBudget,
		"one block destroyed more allocations than the budget allows")
	require.Greater(t, remaining, 0,
		"a chain longer than the budget must leave work queued for the next block")
}

/* TestGridCascade_DeferredWorkFinishesAcrossBlocks proves the budget defers
 * rather than cancels. An abandoned cascade would be worse than a slow one: the
 * substations would stay over-subscribed with nothing left to schedule them.
 */
func TestGridCascade_DeferredWorkFinishesAcrossBlocks(t *testing.T) {
	links := types.GridCascadeBlockBudget + 40
	chain := buildCascadeChain(t, links)
	chain.pullTheRoot(t)

	blocks := 0
	for {
		blocks++
		require.Less(t, blocks, 100, "the cascade is not converging")

		if chain.cascadeBlock(t) == 0 {
			break
		}
	}

	require.Greater(t, blocks, 1, "this chain is supposed to need more than one block")
	require.Zero(t, chain.liveAllocations(t), "every link should have been shed by the end")
}

/* TestGridCascade_ShortChainStillCompletesInOneBlock pins the other direction.
 * A cascade a real grid produces is far below the budget and must not become a
 * multi-block affair, or the fix would be a game-rule change for every player.
 */
func TestGridCascade_ShortChainStillCompletesInOneBlock(t *testing.T) {
	chain := buildCascadeChain(t, 8)
	chain.pullTheRoot(t)

	require.Zero(t, chain.cascadeBlock(t), "a small cascade must finish in the block it starts")
	require.Zero(t, chain.liveAllocations(t))
}

/* buildFanOut gives one source many one-power allocations, which is the other
 * shape a cascade can be made expensive in: not a long chain, a wide one.
 */
func buildFanOut(t *testing.T, allocations int) *cascadeChain {
	t.Helper()

	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	ownerAcc := sdk.AccAddress("cascadefanout1234567890123456789012")
	owner := testAppendPlayer(k, ctx, types.Player{
		Creator:        ownerAcc.String(),
		PrimaryAddress: ownerAcc.String(),
	})

	// One unit of capacity per allocation: nothing here needs more than the
	// source can actually supply.
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner.Id), uint64(allocations))

	chain := &cascadeChain{k: k, ctx: ctx, owner: owner}

	for i := 0; i < allocations; i++ {
		cc := k.NewCurrentContext(ctx)
		ownerCache := cc.GetPlayer(owner.Id)

		allocation, err := cc.NewAllocation(
			types.AllocationType_dynamic,
			owner.Id,
			"",
			owner.Creator,
			owner.Id,
			1,
		)
		require.NoError(t, err, "allocation %d", i)

		substation, err := cc.NewSubstation(owner.Creator, ownerCache, allocation)
		require.NoError(t, err, "substation %d", i)
		cc.CommitAll()

		chain.substationIds = append(chain.substationIds, substation.ID())
	}

	return chain
}

// cascadeBlockGas runs one cascade block against a metered context and reports
// what it cost. Gas is the observable that separates "shed 256 allocations" from
// "loaded 100k allocations and then shed 256": the budget already bounds the
// destroys, so only the reads distinguish the two.
func (c *cascadeChain) cascadeBlockGas(t *testing.T) uint64 {
	t.Helper()

	metered := c.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	cc := c.k.NewCurrentContext(sdk.UnwrapSDKContext(metered))

	before := metered.GasMeter().GasConsumed()
	cc.GridCascade()
	cc.CommitAll()

	return metered.GasMeter().GasConsumed() - before
}

// collapseSource cuts a source to zero capacity and queues it, so everything on
// it has to be shed.
func (c *cascadeChain) collapseSource(t *testing.T) {
	t.Helper()

	c.k.SetGridAttribute(c.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, c.owner.Id), 0)
	require.NoError(t, c.k.AppendGridCascadeQueue(c.ctx, c.owner.Id))
}

/* TestAllocationSourceIndex_BoundedRead pins the primitive the cascade leans on.
 *
 * The limit bounds the iteration rather than the result, and `more` is what lets
 * a caller tell "this source is done" from "I stopped early" - opposite
 * conclusions, since the first means the load cannot be shed and the second
 * means come back next block.
 */
func TestAllocationSourceIndex_BoundedRead(t *testing.T) {
	const fanOut = 12
	chain := buildFanOut(t, fanOut)

	require.True(t, chain.k.SourceHasAllocations(chain.ctx, chain.owner.Id))

	partial, more := chain.k.GetAllocationIdsBySourceIndexUpTo(chain.ctx, chain.owner.Id, 5)
	require.Len(t, partial, 5, "the read must stop at the limit")
	require.True(t, more, "a truncated read must say so")

	exact, more := chain.k.GetAllocationIdsBySourceIndexUpTo(chain.ctx, chain.owner.Id, fanOut)
	require.Len(t, exact, fanOut)
	require.False(t, more, "reading exactly the whole index is not truncation")

	over, more := chain.k.GetAllocationIdsBySourceIndexUpTo(chain.ctx, chain.owner.Id, fanOut+50)
	require.Len(t, over, fanOut)
	require.False(t, more)

	// A zero limit still has to answer the "is there more" question, and does it
	// with one index row rather than the whole prefix.
	none, more := chain.k.GetAllocationIdsBySourceIndexUpTo(chain.ctx, chain.owner.Id, 0)
	require.Empty(t, none)
	require.True(t, more)
}

func TestAllocationSourceIndex_EmptySource(t *testing.T) {
	chain := buildFanOut(t, 0)

	require.False(t, chain.k.SourceHasAllocations(chain.ctx, chain.owner.Id))

	ids, more := chain.k.GetAllocationIdsBySourceIndexUpTo(chain.ctx, chain.owner.Id, 10)
	require.Empty(t, ids)
	require.False(t, more)
}

/* TestGridCascade_WideSourceLoadsOnlyWhatItCanShed is the fan-out regression.
 *
 * The budget bounds how many allocations a block *destroys*. It said nothing
 * about how many it *loads*: GetAllAllocationBySource materializes every
 * allocation on a source - an index read, a store read and a retained cache
 * object each - before the loop can decide it has had enough. A source
 * fragmented into one-power allocations therefore put unbounded work straight
 * back into the EndBlocker that the budget was meant to take it out of.
 *
 * Measured in gas rather than asserted on a count, because the count is
 * internal to the CurrentContext. The destroys are identical in both arms - the
 * budget caps them - so any growth with fan-out is reads, which is exactly the
 * quantity under test. With the bound the two arms come out within a fraction of
 * a percent; without it the wide arm climbs with the width of the source.
 */
func TestGridCascade_WideSourceLoadsOnlyWhatItCanShed(t *testing.T) {
	narrow := buildFanOut(t, types.GridCascadeBlockBudget+10)
	narrow.collapseSource(t)
	narrowGas := narrow.cascadeBlockGas(t)

	wide := buildFanOut(t, types.GridCascadeBlockBudget*8)
	wide.collapseSource(t)
	wideGas := wide.cascadeBlockGas(t)

	require.Less(t, wideGas, narrowGas*5/4,
		"an eightfold wider source cost materially more to cascade, so the block is still loading the whole set: narrow=%d wide=%d",
		narrowGas, wideGas)
}

/* TestGridCascade_WideSourceStillFinishes is the other half. A bounded load must
 * defer the rest, not drop it: the source stays queued until everything on it is
 * shed.
 */
func TestGridCascade_WideSourceStillFinishes(t *testing.T) {
	fanOut := types.GridCascadeBlockBudget + 60
	chain := buildFanOut(t, fanOut)
	chain.collapseSource(t)

	before := chain.liveAllocations(t)
	remaining := chain.cascadeBlock(t)
	destroyed := before - chain.liveAllocations(t)

	require.LessOrEqual(t, destroyed, types.GridCascadeBlockBudget,
		"one block destroyed more than the budget allows")
	require.Greater(t, remaining, 0,
		"a source wider than the budget must be requeued, not abandoned")

	blocks := 1
	for chain.cascadeBlock(t) != 0 {
		blocks++
		require.Less(t, blocks, 100, "the fan-out cascade is not converging")
	}
	require.Zero(t, chain.liveAllocations(t))
}

/* TestGridCascade_TruncatedBatchIsRequeuedNotReportedAsStuck separates the two
 * ways the inner loop runs out of allocations. Reaching the end of a truncated
 * batch means come back next block; reaching the end of a complete one means the
 * source cannot be brought under capacity and is the "Grid Queue problem" warn.
 * Confusing them either abandons work or cries wolf every block.
 */
func TestGridCascade_TruncatedBatchIsRequeuedNotReportedAsStuck(t *testing.T) {
	chain := buildFanOut(t, types.GridCascadeBlockBudget+10)

	chain.k.SetGridAttribute(chain.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, chain.owner.Id), 0)
	require.NoError(t, chain.k.AppendGridCascadeQueue(chain.ctx, chain.owner.Id))

	require.NotZero(t, chain.cascadeBlock(t),
		"the source must still be queued after a truncated batch")
	require.Contains(t, chain.k.GetGridCascadeQueue(chain.ctx, false), chain.owner.Id)
}

/* TestGridCascadeQueue_IsFIFONotSorted is the anti-starvation property.
 *
 * With a budget, the order the queue is drained in decides who waits. Keying by
 * object id sorted lexicographically, so an attacker could pick a substation
 * whose id sorts late and keep cheaper ids in front of it forever - the
 * substation is never reached and goes on powering structs it has no capacity
 * for. Under a sequence, an entry queued first is drained first no matter what
 * arrives later or how it sorts.
 */
func TestGridCascadeQueue_IsFIFONotSorted(t *testing.T) {
	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Deliberately enqueued in reverse lexicographic order.
	first, second, third := "9-victim", "5-filler", "1-filler"
	for _, objectId := range []string{first, second, third} {
		require.NoError(t, k.AppendGridCascadeQueue(ctx, objectId))
	}

	require.Equal(t, []string{first, second, third}, k.GetGridCascadeQueue(ctx, false),
		"the queue must drain in arrival order; sorted order would put the victim last")
}

/* TestGridCascadeQueue_ReAppendKeepsPosition is the other half of the same
 * property. If a re-append moved an entry to the back, a stream of appends
 * against one object would push it backwards indefinitely and the sequence would
 * buy nothing.
 */
func TestGridCascadeQueue_ReAppendKeepsPosition(t *testing.T) {
	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	require.NoError(t, k.AppendGridCascadeQueue(ctx, "victim"))
	require.NoError(t, k.AppendGridCascadeQueue(ctx, "later"))

	// Re-appending while still pending is a no-op, not a move.
	require.NoError(t, k.AppendGridCascadeQueue(ctx, "victim"))

	require.Equal(t, []string{"victim", "later"}, k.GetGridCascadeQueue(ctx, false))
	require.Equal(t, 2, k.GetGridCascadeQueueLength(ctx), "a pending object must not get a second entry")
}

// TestGridCascadeQueue_RejectsEmptyId keeps the guard that stops a nil store key
// from panicking the EndBlocker.
func TestGridCascadeQueue_RejectsEmptyId(t *testing.T) {
	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	require.Error(t, k.AppendGridCascadeQueue(ctx, ""))
	require.Zero(t, k.GetGridCascadeQueueLength(ctx))
}

/* TestGridCascadeQueue_MigrationRekeysLegacyRows covers the upgrade path.
 *
 * Pre-v0.22.0 rows are keyed by object id and carry no ordering of their own, so
 * the migration sorts them - which is exactly the order the old cascade
 * processed them in, so the queue a restarted chain drains is the queue it would
 * have drained.
 */
func TestGridCascadeQueue_MigrationRekeysLegacyRows(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	for _, objectId := range []string{"5-30", "5-2", "5-100"} {
		keepertest.WriteRawGridCascadeQueueLegacyRow(t, ctx, objectId)
	}

	migrated, err := k.MigrateGridCascadeQueueToSequence(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, migrated)

	require.Equal(t, []string{"5-100", "5-2", "5-30"}, k.GetGridCascadeQueue(ctx, false),
		"migrated entries keep the sorted order the old cascade used")

	// A second run sees only new-shape rows. Re-keying an already-keyed queue is
	// a no-op in effect, so it must round-trip them rather than double them.
	again, err := k.MigrateGridCascadeQueueToSequence(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, again)
	require.Equal(t, 3, k.GetGridCascadeQueueLength(ctx))
}

// TestGridCascadeQueue_SurvivesExportImport pins the genesis round trip. The
// queue was exported and never imported, which only worked while the cascade
// drained it inside the block that filled it.
func TestGridCascadeQueue_SurvivesExportImport(t *testing.T) {
	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	pending := []string{"5-9", "5-1", "5-4"}
	for _, objectId := range pending {
		require.NoError(t, k.AppendGridCascadeQueue(ctx, objectId))
	}

	exported := k.GetGridCascadeQueueExport(ctx)
	require.Equal(t, pending, exported, "export must preserve arrival order")

	k.GetGridCascadeQueue(ctx, true)
	require.Zero(t, k.GetGridCascadeQueueLength(ctx))

	for _, objectId := range exported {
		require.NoError(t, k.AppendGridCascadeQueue(ctx, objectId))
	}

	require.Equal(t, pending, k.GetGridCascadeQueue(ctx, false),
		"an import must reproduce the queue that was exported")
}
