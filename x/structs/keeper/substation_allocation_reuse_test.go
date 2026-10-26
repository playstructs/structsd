package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

/* Regression suite for one allocation minting unbounded substations.
 *
 * SubstationCreate checked that the allocation existed and that the caller could
 * connect it, and nothing more. SetDestination is a move - it decrements the old
 * destination's capacity and hands the power to the new one - so the same
 * allocation could be submitted again and again, each call minting a fresh
 * substation and a fresh permission record and leaving the previous substation
 * behind with nothing feeding it.
 *
 * Substation creation is free, so the only bound was the per-block message cap:
 * forty permanent, unfunded substations per player per block, from one
 * allocation, forever.
 *
 * The invariant is not new to the codebase - AllocationTransfer already refuses a
 * connected allocation - it was just missing on the creation path, which is also
 * why the simulator only ever offered allocations whose DestinationId was empty.
 */

type substationReuseFixture struct {
	k     keeperlib.Keeper
	ms    types.MsgServer
	goCtx sdk.Context
	owner types.Player
}

func setupSubstationReuseFixture(t *testing.T) *substationReuseFixture {
	t.Helper()

	k, ms, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	acc := sdk.AccAddress("substationreuse12345678901234567890")
	owner := testAppendPlayer(k, ctx, types.Player{Creator: acc.String(), PrimaryAddress: acc.String()})
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner.Id), 1000)

	return &substationReuseFixture{k: k, ms: ms, goCtx: ctx, owner: owner}
}

func (f *substationReuseFixture) newAllocation(t *testing.T, power uint64) string {
	t.Helper()

	cc := f.k.NewCurrentContext(f.goCtx)
	allocation, err := cc.NewAllocation(types.AllocationType_dynamic, f.owner.Id, "", f.owner.Creator, f.owner.Id, power)
	require.NoError(t, err)
	cc.CommitAll()

	return allocation.ID()
}

/* TestSubstationCreate_AllocationCannotBeReused is the direct regression. The
 * first creation must work; the second, on the now-connected allocation, must
 * not - and must leave no substation behind.
 */
func TestSubstationCreate_AllocationCannotBeReused(t *testing.T) {
	f := setupSubstationReuseFixture(t)
	allocationId := f.newAllocation(t, 100)

	first, err := f.ms.SubstationCreate(f.goCtx, &types.MsgSubstationCreate{
		Creator:      f.owner.Creator,
		AllocationId: allocationId,
	})
	require.NoError(t, err, "the first substation is the legitimate one")
	require.NotEmpty(t, first.SubstationId)

	countAfterFirst := len(f.k.GetAllSubstation(f.goCtx))
	require.Equal(t, 1, countAfterFirst)

	_, err = f.ms.SubstationCreate(f.goCtx, &types.MsgSubstationCreate{
		Creator:      f.owner.Creator,
		AllocationId: allocationId,
	})
	require.Error(t, err, "an allocation already feeding a substation must not create another")
	require.ErrorContains(t, err, "already feeds substation")

	require.Equal(t, countAfterFirst, len(f.k.GetAllSubstation(f.goCtx)),
		"a refused creation must not have persisted a substation")
}

// The state growth is the point, so measure it over a run rather than trusting
// a single rejection.
func TestSubstationCreate_RepeatedReuseGrowsNothing(t *testing.T) {
	f := setupSubstationReuseFixture(t)
	allocationId := f.newAllocation(t, 100)

	_, err := f.ms.SubstationCreate(f.goCtx, &types.MsgSubstationCreate{
		Creator:      f.owner.Creator,
		AllocationId: allocationId,
	})
	require.NoError(t, err)

	baseline := len(f.k.GetAllSubstation(f.goCtx))
	basePermissions := len(f.k.GetAllPermissionExport(f.goCtx))

	for i := 0; i < 20; i++ {
		_, err := f.ms.SubstationCreate(f.goCtx, &types.MsgSubstationCreate{
			Creator:      f.owner.Creator,
			AllocationId: allocationId,
		})
		require.Error(t, err)
	}

	require.Equal(t, baseline, len(f.k.GetAllSubstation(f.goCtx)),
		"twenty reuses minted twenty substations with nothing feeding them")
	require.Equal(t, basePermissions, len(f.k.GetAllPermissionExport(f.goCtx)),
		"and a permission record for each")
}

/* TestSubstationCreate_SucceedsAfterDisconnect pins the other direction: the
 * rule is about the allocation being connected, not about it having been used.
 * An allocation released from its substation is a legitimate seed again.
 */
func TestSubstationCreate_SucceedsAfterDisconnect(t *testing.T) {
	f := setupSubstationReuseFixture(t)
	allocationId := f.newAllocation(t, 100)

	_, err := f.ms.SubstationCreate(f.goCtx, &types.MsgSubstationCreate{
		Creator:      f.owner.Creator,
		AllocationId: allocationId,
	})
	require.NoError(t, err)

	_, err = f.ms.SubstationAllocationDisconnect(f.goCtx, &types.MsgSubstationAllocationDisconnect{
		Creator:      f.owner.Creator,
		AllocationId: allocationId,
	})
	require.NoError(t, err)

	_, err = f.ms.SubstationCreate(f.goCtx, &types.MsgSubstationCreate{
		Creator:      f.owner.Creator,
		AllocationId: allocationId,
	})
	require.NoError(t, err,
		"a disconnected allocation is free to seed a new substation; the guard is about connection, not history")
}

// Every ordinary creation uses a fresh allocation, so the guard must not make
// that path any harder.
func TestSubstationCreate_DistinctAllocationsStillWork(t *testing.T) {
	f := setupSubstationReuseFixture(t)

	for i := 0; i < 3; i++ {
		allocationId := f.newAllocation(t, 100)
		_, err := f.ms.SubstationCreate(f.goCtx, &types.MsgSubstationCreate{
			Creator:      f.owner.Creator,
			AllocationId: allocationId,
		})
		require.NoError(t, err, "creation %d with its own allocation", i)
	}

	require.Len(t, f.k.GetAllSubstation(f.goCtx), 3)
}
