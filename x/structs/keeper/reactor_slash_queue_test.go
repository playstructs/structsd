package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

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
