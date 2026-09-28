package keeper_test

import (
	"fmt"
	"sort"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// This measures queue liveness under arrivals, not staking reconciliation: the
// mock has no hooks. Later creations must not extend the captured repair job.
func TestReactorSlashQueue_LivenessWithArrivals(t *testing.T) {
	for _, arrivals := range []bool{false, true} {
		t.Run(fmt.Sprint(arrivals), func(t *testing.T) {
			k, ctx := keepertest.StructsKeeper(t)
			const id = "3-1"
			budget := types.ReactorSlashReconcileBudget
			k.SetReactor(ctx, types.Reactor{Id: id, Validator: sdk.ValAddress("slashvalidator123456").String()})
			addresses := make([]string, 10*budget)
			for i := range addresses {
				addresses[i] = sdk.AccAddress(fmt.Sprintf("arrivaldelegator%020d", i)).String()
			}
			sort.Strings(addresses)
			insert := func(start, end int) {
				for _, address := range addresses[start:end] {
					k.SetInfusion(ctx, types.Infusion{DestinationType: types.ObjectType_reactor, DestinationId: id, Address: address, Fuel: 100})
				}
			}
			insert(0, 2*budget)
			k.EnqueueReactorSlashReconcile(ctx, id)
			inserted := 2 * budget
			completedAt := 0
			for block := 1; block <= 20; block++ {
				if arrivals && block >= 2 && block <= 5 {
					insert(inserted, inserted+2*budget)
					inserted += 2 * budget
				}
				k.ProcessReactorSlashReconcileQueue(ctx)
				queue := k.GetReactorSlashReconcileQueue(ctx)
				if len(queue) == 0 {
					completedAt = block
					break
				}
				require.Len(t, queue, 1)
				// The unvisited cursor cannot pass more rows than the shared
				// budget allowed across all processing calls so far.
				position := sort.SearchStrings(addresses, queue[0].Cursor)
				require.LessOrEqual(t, position, block*budget)
			}
			require.NotZero(t, completedAt, "repair must eventually drain once arrivals stop")
			require.Equal(t, 2, completedAt, "later arrivals cannot extend the captured job")
			t.Logf("initial_entries=%d inserted_entries=%d completed_at_block=%d", 2*budget, inserted, completedAt)
		})
	}
}

func TestReactorRepairFailureDoesNotStarveLaterPositions(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const id = "3-1"
	validator := sdk.ValAddress("slashvalidator123456")
	k.SetReactor(ctx, types.Reactor{Id: id, Validator: validator.String(), DefaultCommission: math.LegacyZeroDec()})
	k.SetReactorValidatorBytes(ctx, id, validator)
	k.StakingKeeper().(*keepertest.MockStakingKeeper).AddValidator(validator, math.NewInt(1000))
	k.SetInfusion(ctx, types.Infusion{DestinationId: id, DestinationType: types.ObjectType_reactor, Address: "invalid-first", Fuel: 1})
	addresses := seedReactorInfusions(t, k, ctx, id, 2*types.ReactorSlashReconcileBudget)
	k.EnqueueReactorRestart(ctx, id)
	for pass := 0; pass < 4; pass++ {
		k.ProcessReactorSlashReconcileQueue(ctx)
	}
	last, found := k.GetInfusion(ctx, id, addresses[len(addresses)-1])
	require.True(t, found)
	require.Zero(t, last.Fuel, "a persistently failing first row must not prevent later repairs")
	require.Len(t, k.GetReactorSlashReconcileQueue(ctx), 1, "failure remains pending rather than being certified")
	snapshot, found, err := k.GetReactorRepairSnapshot(ctx, id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, snapshot.Through, snapshot.After, "the main finite pass must finish despite failed rows")
}

func TestReactorRepairRetryWorkIsBoundedByBudget(t *testing.T) {
	measure := func(count int) uint64 {
		k, ctx := keepertest.StructsKeeper(t)
		const id = "3-1"
		k.SetReactor(ctx, types.Reactor{Id: id, Validator: sdk.ValAddress("slashvalidator123456").String()})
		for i := 0; i < count; i++ {
			k.SetInfusion(ctx, types.Infusion{DestinationId: id, DestinationType: types.ObjectType_reactor,
				Address: fmt.Sprintf("invalid%08d", i), Fuel: 1})
		}
		k.EnqueueReactorRestart(ctx, id)
		k.ProcessReactorSlashReconcileQueue(ctx)
		metered := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		k.ProcessReactorSlashReconcileQueue(metered)
		snapshot, found, err := k.GetReactorRepairSnapshot(ctx, id)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(types.ReactorSlashReconcileBudget+types.ReactorSlashReconcileBudget/2), snapshot.After,
			"retries share the budget without monopolizing the main pass")
		return metered.GasMeter().GasConsumed()
	}
	small := measure(2 * types.ReactorSlashReconcileBudget)
	large := measure(20 * types.ReactorSlashReconcileBudget)
	require.Equal(t, small, large, "per-call work must not grow with the failed-position inventory")
	t.Logf("repair gas with pending failures: %d for either inventory", small)
}

func TestReactorRepairSnapshotExcludesRecreationAndResetsForNewPass(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const id = "3-1"
	k.SetReactor(ctx, types.Reactor{Id: id, Validator: sdk.ValAddress("slashvalidator123456").String()})
	addresses := seedReactorInfusions(t, k, ctx, id, types.ReactorSlashReconcileBudget+1)
	k.EnqueueReactorRestart(ctx, id)
	k.ProcessReactorSlashReconcileQueue(ctx)
	snapshot, found, err := k.GetReactorRepairSnapshot(ctx, id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(types.ReactorSlashReconcileBudget), snapshot.After)
	require.Equal(t, uint64(len(addresses)), snapshot.Through)
	// Delete/recreate the last unprocessed address. Its old position is gone;
	// the new row's normal hooks, not the earlier repair, own its accounting.
	k.RemoveInfusion(ctx, id, addresses[len(addresses)-1])
	k.SetInfusion(ctx, types.Infusion{DestinationId: id, DestinationType: types.ObjectType_reactor, Address: addresses[len(addresses)-1], Fuel: 100})
	k.EnqueueReactorRestart(ctx, id)
	unchanged, _, err := k.GetReactorRepairSnapshot(ctx, id)
	require.NoError(t, err)
	require.Equal(t, snapshot, unchanged)
	k.ProcessReactorSlashReconcileQueue(ctx)
	require.Empty(t, k.GetReactorSlashReconcileQueue(ctx))
	_, found, err = k.GetReactorRepairSnapshot(ctx, id)
	require.NoError(t, err)
	require.False(t, found, "completion clears its snapshot as well as the queue row")
	k.EnqueueReactorRestart(ctx, id)
	k.ProcessReactorSlashReconcileQueue(ctx)
	progressed, _, err := k.GetReactorRepairSnapshot(ctx, id)
	require.NoError(t, err)
	require.NotZero(t, progressed.After)
	// A genuine requeue starts a new pass and captures every position now known.
	k.EnqueueReactorSlashReconcile(ctx, id)
	reset, found, err := k.GetReactorRepairSnapshot(ctx, id)
	require.NoError(t, err)
	require.True(t, found)
	require.Zero(t, reset.After)
	require.Equal(t, uint64(len(addresses)+1), reset.Through)
}

func TestReactorSlashQueue_FailedBatchDoesNotStarveAnotherReactor(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	for _, id := range []string{"3-1", "3-2"} {
		k.SetReactor(ctx, types.Reactor{Id: id, Validator: sdk.ValAddress("slashvalidator123456").String()})
	}
	for i := 0; i < types.ReactorSlashReconcileBudget+10; i++ {
		k.SetInfusion(ctx, types.Infusion{DestinationId: "3-1", DestinationType: types.ObjectType_reactor,
			Address: fmt.Sprintf("invalid%08d", i), Fuel: 1})
	}
	k.EnqueueReactorRestart(ctx, "3-1")
	k.EnqueueReactorRestart(ctx, "3-2")
	k.ProcessReactorSlashReconcileQueue(ctx)
	queue := k.GetReactorSlashReconcileQueue(ctx)
	require.Len(t, queue, 2, "the first failed batch consumes the shared work allowance")
	require.Equal(t, "invalid00000127", queue[0].Cursor)
	k.ProcessReactorSlashReconcileQueue(ctx)
	queue = k.GetReactorSlashReconcileQueue(ctx)
	require.Len(t, queue, 1, "round-robin scheduling must still finish the other reactor")
	require.Equal(t, "3-1", queue[0].ReactorId)
	require.Equal(t, "invalid00000064", queue[0].Cursor, "failed retries rotate behind other pending rows")
	// A manual repair request cannot skip the bad row or reset away its retry.
	k.EnqueueReactorRestart(ctx, "3-1")
	require.Equal(t, queue, k.GetReactorSlashReconcileQueue(ctx))
}

// Missing reactors still cost reads and queue deletes. They must not bypass
// the per-block bound merely because no infusion is available to charge to it.
func TestReactorSlashQueue_BoundsMissingReactorCleanup(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	count := types.ReactorSlashReconcileBudget*3 + 1
	for i := 0; i < count; i++ {
		k.EnqueueReactorSlashReconcile(ctx, fmt.Sprintf("3-%d", 100000+i))
	}
	for remaining := count; remaining > 0; {
		k.ProcessReactorSlashReconcileQueue(ctx)
		remaining -= types.ReactorSlashReconcileBudget
		if remaining < 0 {
			remaining = 0
		}
		require.Len(t, k.GetReactorSlashReconcileQueue(ctx), remaining)
	}
}

func TestReactorSlashQueue_ReadCostDoesNotGrowWithBacklog(t *testing.T) {
	measure := func(count int) uint64 {
		k, ctx := keepertest.StructsKeeper(t)
		for i := 0; i < count; i++ {
			k.EnqueueReactorSlashReconcile(ctx, fmt.Sprintf("3-%d", 100000+i))
		}
		metered := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		k.ProcessReactorSlashReconcileQueue(metered)
		return metered.GasMeter().GasConsumed()
	}
	// Both calls see the same fixed-width first batch; the larger queue must
	// not first materialize thousands of rows it cannot process this block.
	require.Equal(t, measure(1000), measure(10000))
}

func TestActiveBondedHookCostDoesNotGrowWithInfusions(t *testing.T) {
	measure := func(count int) uint64 {
		k, ctx := keepertest.StructsKeeper(t)
		validator := sdk.ValAddress("slashvalidator123456")
		reactor := k.AppendReactor(ctx, types.Reactor{Validator: validator.String(), RawAddress: validator.Bytes()})
		seedReactorInfusions(t, k, ctx, reactor.Id, count)

		metered := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		require.NoError(t, k.Hooks().AfterValidatorBonded(metered, nil, validator))
		gas := metered.GasMeter().GasConsumed()
		queue := k.GetReactorSlashReconcileQueue(ctx)
		require.Len(t, queue, 1, "constant cost must enqueue real work, not skip the reactor")
		require.Equal(t, reactor.Id, queue[0].ReactorId)
		return gas
	}
	require.Equal(t, measure(100), measure(10000))
}

func TestReactorSlashQueue_RepeatedEarlierSlashCannotStarveLaterReactor(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	// The later key shares the earlier key's prefix, exercising the exclusive
	// resume boundary as well as the eventual wrap to the start.
	for _, id := range []string{"3-1", "3-10"} {
		k.SetReactor(ctx, types.Reactor{Id: id, Validator: sdk.ValAddress("slashvalidator123456").String()})
		seedReactorInfusions(t, k, ctx, id, types.ReactorSlashReconcileBudget+30)
		k.EnqueueReactorSlashReconcile(ctx, id)
	}
	k.ProcessReactorSlashReconcileQueue(ctx)
	queue := k.GetReactorSlashReconcileQueue(ctx)
	require.Len(t, queue, 2)
	require.NotEmpty(t, queue[0].Cursor)
	require.Empty(t, queue[1].Cursor)
	// A new slash correctly resets the earlier reactor's infusion cursor. It
	// must not also hand it every subsequent block's entire processing budget.
	k.EnqueueReactorSlashReconcile(ctx, "3-1")
	k.ProcessReactorSlashReconcileQueue(ctx)
	queue = k.GetReactorSlashReconcileQueue(ctx)
	require.Len(t, queue, 2)
	require.Empty(t, queue[0].Cursor)
	require.NotEmpty(t, queue[1].Cursor, "the next reactor gets a turn despite the earlier reset")
	for i := 0; i < 3; i++ {
		k.ProcessReactorSlashReconcileQueue(ctx)
	}
	require.Empty(t, k.GetReactorSlashReconcileQueue(ctx), "round-robin wrap must drain both remainders")
}

/* Regression suite for the unbounded slashing hook.
 *
 * BeforeValidatorSlashed used to reconcile every active delegation to the
 * validator inline. It runs in BeginBlock against an infinite gas meter, and the
 * delegator set is unbounded and sized by the delegators, so one slash cost
 * whatever they had accumulated - a delegation read, an unbonding-delegation
 * read and a grid rewrite each.
 *
 * A slash leaves delegation shares untouched, so AfterDelegationModified never
 * fires for it and this hook is the module's only signal. Reconciliation can
 * therefore be deferred but never dropped, which is what the queue is for.
 */

// seedReactorInfusions writes count infusions against one reactor, the way a
// reactor with many delegators looks on disk.
func seedReactorInfusions(t *testing.T, k keeperlib.Keeper, ctx sdk.Context, reactorId string, count int) []string {
	t.Helper()

	addresses := make([]string, 0, count)
	for i := 0; i < count; i++ {
		address := sdk.AccAddress(fmt.Sprintf("slashdelegator%021d", i)).String()
		k.SetInfusion(ctx, types.Infusion{
			DestinationType: types.ObjectType_reactor,
			DestinationId:   reactorId,
			Address:         address,
			Fuel:            100,
		})
		addresses = append(addresses, address)
	}

	return addresses
}

func TestReactorSlashQueue_EnqueueAndDrain(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const reactorId = "3-1"

	require.Empty(t, k.GetReactorSlashReconcileQueue(ctx))

	k.EnqueueReactorSlashReconcile(ctx, reactorId)

	queue := k.GetReactorSlashReconcileQueue(ctx)
	require.Len(t, queue, 1)
	require.Equal(t, reactorId, queue[0].ReactorId)
	require.Empty(t, queue[0].Cursor, "a fresh entry starts at the beginning")
}

// An empty reactor id must not become a queue row; the store panics on an empty
// key and the EndBlocker is not the place to find that out.
func TestReactorSlashQueue_IgnoresEmptyReactorId(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	k.EnqueueReactorSlashReconcile(ctx, "")
	require.Empty(t, k.GetReactorSlashReconcileQueue(ctx))
}

/* TestReactorSlashQueue_BatchCursorNeitherSkipsNorRepeats is the property the
 * whole design rests on. The cursor is the *next* key to read rather than the
 * last one read, so walking a destination in batches has to reproduce a single
 * full scan exactly - a skip silently leaves an infusion carrying pre-slash fuel
 * forever, and a repeat is wasted block time on every pass.
 */
func TestReactorSlashQueue_BatchCursorNeitherSkipsNorRepeats(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const reactorId = "3-1"

	seedReactorInfusions(t, k, ctx, reactorId, 25)

	full := make([]string, 0, 25)
	for _, infusion := range k.GetAllInfusionsByDestination(ctx, reactorId) {
		full = append(full, infusion.Address)
	}
	require.Len(t, full, 25)

	var walked []string
	cursor := ""
	for pass := 0; ; pass++ {
		require.Less(t, pass, 20, "the batch walk is not converging")

		batch, next := k.ReadInfusionAddressBatchForTest(ctx, reactorId, cursor, 7)
		walked = append(walked, batch...)
		if next == "" {
			break
		}
		cursor = next
	}

	require.Equal(t, full, walked,
		"the batched walk must visit exactly the same addresses, in the same order, as one full scan")
}

func TestReactorSlashQueue_BatchStopsAtTheLimit(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const reactorId = "3-1"

	seedReactorInfusions(t, k, ctx, reactorId, 10)

	batch, next := k.ReadInfusionAddressBatchForTest(ctx, reactorId, "", 4)
	require.Len(t, batch, 4)
	require.NotEmpty(t, next, "a truncated batch must hand back a resume point")

	batch, next = k.ReadInfusionAddressBatchForTest(ctx, reactorId, "", 10)
	require.Len(t, batch, 10)
	require.Empty(t, next, "reading exactly the whole destination is not truncation")

	batch, next = k.ReadInfusionAddressBatchForTest(ctx, reactorId, "", 50)
	require.Len(t, batch, 10)
	require.Empty(t, next)
}

/* TestReactorSlashQueue_ProcessingIsBoundedAndResumes is the regression itself:
 * a reactor with more infusions than one block's budget must not be finished in
 * one block, and must not be abandoned either.
 */
func TestReactorSlashQueue_ProcessingIsBoundedAndResumes(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const reactorId = "3-1"

	// A reactor record the processor can resolve, with a validator address that
	// parses. Reconciliation itself is a no-op against the mock staking keeper;
	// what is under test is the batching, not the arithmetic.
	k.SetReactor(ctx, types.Reactor{
		Id:        reactorId,
		Validator: sdk.ValAddress("slashvalidator123456").String(),
	})

	seedReactorInfusions(t, k, ctx, reactorId, types.ReactorSlashReconcileBudget+30)
	k.EnqueueReactorSlashReconcile(ctx, reactorId)

	k.ProcessReactorSlashReconcileQueue(ctx)

	queue := k.GetReactorSlashReconcileQueue(ctx)
	require.Len(t, queue, 1, "a reactor larger than the budget must stay queued")
	require.NotEmpty(t, queue[0].Cursor, "and must remember where to resume")
	for attempt := 0; attempt < 10; attempt++ {
		k.EnqueueReactorRestart(ctx, reactorId)
	}
	require.Equal(t, queue, k.GetReactorSlashReconcileQueue(ctx), "permissionless retries must not rewind the cursor")

	k.ProcessReactorSlashReconcileQueue(ctx)
	require.Empty(t, k.GetReactorSlashReconcileQueue(ctx),
		"the remainder must finish on a later block, not be dropped")
}

// A reactor small enough to finish in one block must leave no queue row behind.
func TestReactorSlashQueue_SmallReactorFinishesInOneBlock(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const reactorId = "3-1"

	k.SetReactor(ctx, types.Reactor{
		Id:        reactorId,
		Validator: sdk.ValAddress("slashvalidator123456").String(),
	})
	seedReactorInfusions(t, k, ctx, reactorId, 5)
	k.EnqueueReactorSlashReconcile(ctx, reactorId)

	k.ProcessReactorSlashReconcileQueue(ctx)
	require.Empty(t, k.GetReactorSlashReconcileQueue(ctx))
}

// A queued reactor whose record has since gone must drop out rather than being
// retried against nothing every block.
func TestReactorSlashQueue_MissingReactorIsDropped(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	k.EnqueueReactorSlashReconcile(ctx, "3-404")
	k.ProcessReactorSlashReconcileQueue(ctx)

	require.Empty(t, k.GetReactorSlashReconcileQueue(ctx))
}
