package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
)

/* Regression suite for a malformed destruction-queue key panicking the block.
 *
 * The queue is read in the EndBlocker and every row is handed to
 * GetInfusionById, which resolves an infusion by splitting its key into three
 * parts. A key that does not split had no destination and no address to build a
 * cache around, so what came back was a zero InfusionCache with a nil
 * CurrentContext - and the first method call on it dereferenced that nil.
 *
 * The retry is what made it a halt rather than a crash: the queue read clears
 * its rows in the same block, the panic discards that delete, and the next block
 * reads the same row again. For ever.
 */

func TestProcessInfusionDestructionQueue_DiscardsMalformedKey(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	// The key an empty destination and address produce, which is exactly what a
	// genesis-imported infusion with unset fields would have written.
	k.AppendInfusionDestructionQueue(ctx, "-")
	k.AppendInfusionDestructionQueue(ctx, "not-an-infusion-key-at-all")

	require.Len(t, k.GetInfusionDestructionQueueExport(ctx), 2, "fixture sanity")

	cc := k.NewCurrentContext(ctx)
	require.NotPanics(t, func() { cc.ProcessInfusionDestructionQueue() },
		"a malformed queue key must not panic the EndBlocker")
	cc.CommitAll()

	require.Empty(t, k.GetInfusionDestructionQueueExport(ctx),
		"an unresolvable row must be discarded, not left to be read again next block")
}

// A well-formed key naming an infusion that is not there is an ordinary miss,
// not corruption, and must still be consumed without complaint.
func TestProcessInfusionDestructionQueue_AbsentInfusionIsNotAnError(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	k.AppendInfusionDestructionQueue(ctx, "3-1-structs1nobody")

	cc := k.NewCurrentContext(ctx)
	require.NotPanics(t, func() { cc.ProcessInfusionDestructionQueue() })
	cc.CommitAll()

	require.Empty(t, k.GetInfusionDestructionQueueExport(ctx))
}

// The resolver itself reports the failure rather than handing back a cache that
// cannot be used.
func TestGetInfusionById_ReportsUnparseableKeys(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	cc := k.NewCurrentContext(ctx)

	_, found := cc.GetInfusionById("-")
	require.False(t, found, "an unparseable key has no infusion behind it")

	_, found = cc.GetInfusionById("3-1-structs1abc")
	require.True(t, found, "a well-formed key still resolves")
}
