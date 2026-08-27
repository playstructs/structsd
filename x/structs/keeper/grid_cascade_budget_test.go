package keeper_test

import (
	"testing"

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
