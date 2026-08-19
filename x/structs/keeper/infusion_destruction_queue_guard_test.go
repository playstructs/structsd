package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func queueContains(queue []string, id string) bool {
	for _, q := range queue {
		if q == id {
			return true
		}
	}
	return false
}

// A live infusion that has emptied out must still be queued for reclamation:
// the destruction-queue guard must not break the legitimate path.
func TestInfusionCommit_EmptyLiveInfusionIsQueued(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	dest := "5-1"
	addr := "structs1liveempty"
	// fuel 0 -> power 0 -> the record is empty but present in the store.
	keeper.SetInfusion(ctx, types.CreateNewInfusion(types.ObjectType_reactor, dest, addr, "1-1", 0, math.LegacyNewDec(0), 1))

	cc := keeper.NewCurrentContext(ctx)
	cc.GetInfusion(dest, addr)
	cc.CommitAll()

	require.True(t, queueContains(keeper.GetInfusionDestructionQueueExport(ctx), dest+"-"+addr),
		"an empty, present infusion should be queued for reclamation")
}

// Probing an infusion that never existed (the disownInfusion /
// ReactorInfusionDelegationRemoved pattern that loads a cache before its
// existence check) must not mint a permanent, self-perpetuating queue row.
func TestInfusionCommit_PhantomProbeIsNotQueued(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	dest := "5-1"
	ghost := "structs1ghost"

	cc := keeper.NewCurrentContext(ctx)
	cc.GetInfusion(dest, ghost) // registers a cache, loads nothing
	cc.CommitAll()

	require.False(t, queueContains(keeper.GetInfusionDestructionQueueExport(ctx), dest+"-"+ghost),
		"a probe of a nonexistent infusion must not enqueue a destruction-queue row")
}

// A destroyed infusion removes its own record and must not re-queue itself; if
// it did, the sweep would re-add the row every block forever.
func TestInfusionCommit_DestroyedInfusionDoesNotRequeue(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	dest := "5-1"
	addr := "structs1doomed"
	keeper.SetInfusion(ctx, types.CreateNewInfusion(types.ObjectType_reactor, dest, addr, "1-1", 100, math.LegacyNewDec(0), 1))

	cc := keeper.NewCurrentContext(ctx)
	cc.GetInfusion(dest, addr).Destroy()
	cc.CommitAll()

	_, found := keeper.GetInfusion(ctx, dest, addr)
	require.False(t, found, "destroyed infusion record should be gone")
	require.False(t, queueContains(keeper.GetInfusionDestructionQueueExport(ctx), dest+"-"+addr),
		"a destroyed infusion must not re-enqueue itself")
}
