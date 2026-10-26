package keeper_test

import (
	"fmt"
	"math"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func diversityCapacityKey(owner string) string {
	return keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner)
}

func diversityEvents(t *testing.T, ctx sdk.Context) []types.OwnerReactorDiversity {
	t.Helper()
	var rows []types.OwnerReactorDiversity
	for _, event := range ctx.EventManager().ABCIEvents() {
		if event.Type != "structs.structs.EventOwnerReactorDiversity" {
			continue
		}
		message, err := sdk.ParseTypedEvent(event)
		require.NoError(t, err)
		rows = append(rows, *message.(*types.EventOwnerReactorDiversity).Diversity)
	}
	return rows
}

func TestDiversityCacheAggregatesAddressesAndCoalescesEvents(t *testing.T) {
	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	const owner = "1-1"
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-1", 0, 70)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-1", 0, 30)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-2", 0, 100)
	require.Equal(t, uint64(211), cc.GetGridAttribute(diversityCapacityKey(owner)))
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-1", 70, 80)
	state := cc.GetOwnerReactorDiversity(owner).GetState()
	require.Equal(t, "210", state.Power)
	require.Equal(t, "22100", state.SquaredPower, "square the 110 aggregate, not separate 80 and 30 rows")
	require.Equal(t, uint64(11), state.AppliedBonus)
	_, found := k.GetOwnerReactorDiversity(ctx, owner)
	require.False(t, found, "cache mutations do not publish derived state early")
	require.Empty(t, diversityEvents(t, ctx))
	cc.CommitAll()
	row, found := k.GetReactorDiversityPosition(ctx, owner, "3-1")
	require.True(t, found)
	require.Equal(t, "110", row.Power)
	require.Equal(t, []types.OwnerReactorDiversity{state}, diversityEvents(t, ctx))
	require.Equal(t, uint64(221), k.GetGridAttribute(ctx, diversityCapacityKey(owner)))
	// A new context loads the stored summary once and reports no event when
	// intermediate changes return it to the same final state.
	cc = k.NewCurrentContext(ctx)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-1", 80, 90)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-1", 90, 80)
	cc.CommitAll()
	require.Len(t, diversityEvents(t, ctx), 1)
}

func TestDiversityCacheRefreshesBeforeCapacityReadAndCommit(t *testing.T) {
	for _, read := range []bool{false, true} {
		t.Run(fmt.Sprint(read), func(t *testing.T) {
			k, _, ctx := setupMsgServer(t)
			cc := k.NewCurrentContext(ctx)
			// Stage the canonical base and portfolio separately to exercise a
			// genuinely dirty summary, without the combined delta helper.
			cc.SetGridAttribute(diversityCapacityKey("1-1"), 200)
			cc.ChangeReactorDiversityPosition("1-1", "3-1", 0, 100)
			cc.ChangeReactorDiversityPosition("1-1", "3-2", 0, 100)
			require.True(t, cc.GetOwnerReactorDiversity("1-1").Dirty)
			if read {
				require.Equal(t, uint64(211), cc.GetGridAttribute(diversityCapacityKey("1-1")))
				require.False(t, cc.GetOwnerReactorDiversity("1-1").Dirty)
			}
			cc.CommitAll()
			require.Equal(t, uint64(211), k.GetGridAttribute(ctx, diversityCapacityKey("1-1")))
			row, found := k.GetOwnerReactorDiversity(ctx, "1-1")
			require.True(t, found)
			require.Equal(t, uint64(11), row.AppliedBonus)
		})
	}
}

func TestDiversityCacheDepositReductionQueuesCascade(t *testing.T) {
	k, _, ctx := setupMsgServer(t)
	cc := k.NewCurrentContext(ctx)
	const owner = "1-1"
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-1", 0, 600)
	for i := 2; i <= 21; i++ {
		cc.ApplyOwnerInfusionCapacityDelta(owner, fmt.Sprintf("3-%d", i), 0, 70)
	}
	cc.CommitAll()
	require.Equal(t, uint64(2859), k.GetGridAttribute(ctx, diversityCapacityKey(owner)))
	require.Empty(t, k.GetGridCascadeQueueBatch(ctx, 10))
	cc = k.NewCurrentContext(ctx)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-1", 600, 650)
	cc.CommitAll()
	require.Equal(t, uint64(2855), k.GetGridAttribute(ctx, diversityCapacityKey(owner)))
	queue := k.GetGridCascadeQueueBatch(ctx, 10)
	require.Len(t, queue, 1)
	require.Equal(t, owner, queue[0].ObjectId)
}

func TestDiversityCacheZeroTransitionRemovesDerivedRows(t *testing.T) {
	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	const owner = "0-1"
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-1", 0, 100)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-2", 0, 100)
	cc.CommitAll()
	cc = k.NewCurrentContext(ctx)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-1", 100, 0)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-2", 100, 0)
	cc.CommitAll()
	_, found := k.GetOwnerReactorDiversity(ctx, owner)
	require.False(t, found)
	for _, reactor := range []string{"3-1", "3-2"} {
		_, found := k.GetReactorDiversityPosition(ctx, owner, reactor)
		require.False(t, found)
	}
	rows := diversityEvents(t, ctx)
	require.Len(t, rows, 2)
	require.Equal(t, types.OwnerReactorDiversity{OwnerId: owner, Power: "0", SquaredPower: "0"}, rows[1])
	require.Zero(t, k.GetGridAttribute(ctx, diversityCapacityKey(owner)))
}

func TestDiversityCacheClipsBonusWithoutLosingBasePower(t *testing.T) {
	k, _, ctx := setupMsgServer(t)
	cc := k.NewCurrentContext(ctx)
	const owner = "1-1"
	const generator = uint64(math.MaxUint64 - 205)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "", 0, generator)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-1", 0, 100)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-2", 0, 100)
	require.Equal(t, uint64(math.MaxUint64), cc.GetGridAttribute(diversityCapacityKey(owner)))
	require.Equal(t, uint64(5), cc.GetOwnerReactorDiversity(owner).AppliedBonus)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "", generator, generator-10)
	require.Equal(t, uint64(math.MaxUint64-4), cc.GetGridAttribute(diversityCapacityKey(owner)))
	require.Equal(t, uint64(11), cc.GetOwnerReactorDiversity(owner).AppliedBonus)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "", generator-10, generator)
	require.Equal(t, uint64(math.MaxUint64), cc.GetGridAttribute(diversityCapacityKey(owner)))
	require.Equal(t, uint64(5), cc.GetOwnerReactorDiversity(owner).AppliedBonus)
	cc.ApplyOwnerInfusionCapacityDelta(owner, "3-1", 100, 0)
	cc.CommitAll()
	require.Equal(t, uint64(math.MaxUint64-105), k.GetGridAttribute(ctx, diversityCapacityKey(owner)))
	row, found := k.GetOwnerReactorDiversity(ctx, owner)
	require.True(t, found)
	require.Equal(t, "100", row.Power, "generator does not enter portfolio pricing")
	require.Zero(t, row.AppliedBonus)
}

func TestDiversityCacheResizesAutomatedAllocationAfterBonusReduction(t *testing.T) {
	f := setupAutoResizeFixture(t, 0)
	cc := f.k.NewCurrentContext(f.ctx)
	cc.ApplyOwnerInfusionCapacityDelta(f.owner.Id, "3-1", 0, 600)
	for i := 2; i <= 21; i++ {
		cc.ApplyOwnerInfusionCapacityDelta(f.owner.Id, fmt.Sprintf("3-%d", i), 0, 70)
	}
	cc.CommitAll()
	id := f.newAutomated(t)
	before, found := f.k.GetAllocation(f.ctx, id)
	require.True(t, found)
	require.Equal(t, uint64(2859), f.k.GetGridAttribute(f.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, before.Id)))
	cc = f.k.NewCurrentContext(f.ctx)
	cc.ApplyOwnerInfusionCapacityDelta(f.owner.Id, "3-1", 600, 650)
	cc.CommitAll()
	require.Equal(t, uint64(2855), f.k.GetGridAttribute(f.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, id)))
}

func TestDiversityCacheUpdateCostDoesNotWalkOtherOwners(t *testing.T) {
	measure := func(otherOwners int) uint64 {
		k, _, goCtx := setupMsgServer(t)
		ctx := sdk.UnwrapSDKContext(goCtx)
		cc := k.NewCurrentContext(ctx)
		cc.ApplyOwnerInfusionCapacityDelta("1-1", "3-1", 0, 100)
		cc.ApplyOwnerInfusionCapacityDelta("1-1", "3-2", 0, 100)
		for i := 0; i < otherOwners; i++ {
			cc.ApplyOwnerInfusionCapacityDelta(fmt.Sprintf("1-%d", i+2), "3-1", 0, 100)
		}
		cc.CommitAll()
		metered := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		cc = k.NewCurrentContext(metered)
		cc.ApplyOwnerInfusionCapacityDelta("1-1", "3-1", 100, 110)
		cc.CommitAll()
		return metered.GasMeter().GasConsumed()
	}
	small, large := measure(10), measure(2000)
	require.Equal(t, small, large)
	require.Less(t, large, uint64(20_000_000))
	t.Logf("other_owners_10_gas=%d other_owners_2000_gas=%d", small, large)
}
