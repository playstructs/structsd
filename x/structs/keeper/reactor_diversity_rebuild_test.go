package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

func TestDiversityReconstructionUsesCanonicalSharesAndIsIdempotent(t *testing.T) {
	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)
	owner := testAppendPlayer(k, ctx, types.Player{})
	rows := []types.Infusion{
		types.CreateNewInfusion(types.ObjectType_reactor, "3-1", "first", owner.Id, 140, math.LegacyNewDecWithPrec(5, 1), 1),
		types.CreateNewInfusion(types.ObjectType_reactor, "3-1", "second", owner.Id, 60, math.LegacyNewDecWithPrec(5, 1), 1),
		types.CreateNewInfusion(types.ObjectType_reactor, "3-2", "first", owner.Id, 200, math.LegacyNewDecWithPrec(5, 1), 1),
		types.CreateNewInfusion(types.ObjectType_struct, "5-1", "first", owner.Id, 100, math.LegacyZeroDec(), 1),
	}
	for _, row := range rows {
		k.SetInfusion(ctx, row)
	}
	k.SetGridAttribute(ctx, diversityCapacityKey(owner.Id), 300)
	require.ErrorContains(t, k.CheckReactorDiversityAccounting(ctx), "inventory mismatch")
	require.NoError(t, k.RebuildReactorDiversity(ctx))
	require.NoError(t, k.CheckReactorDiversityAccounting(ctx))
	state, found := k.GetOwnerReactorDiversity(ctx, owner.Id)
	require.True(t, found)
	require.Equal(t, "200", state.Power)
	require.Equal(t, "20000", state.SquaredPower)
	require.Equal(t, uint64(11), state.AppliedBonus)
	require.Equal(t, uint64(311), k.GetGridAttribute(ctx, diversityCapacityKey(owner.Id)))
	before := keepertest.StructsStoreRows(t, ctx)
	events := len(diversityEvents(t, ctx))
	require.NoError(t, k.RebuildReactorDiversity(ctx))
	require.Equal(t, before, keepertest.StructsStoreRows(t, ctx), "a second rebuild must not add a second bonus")
	require.Len(t, diversityEvents(t, ctx), events, "unchanged summaries are coalesced")

	rows[0].Fuel, rows[0].Power = 100, 100
	k.SetInfusion(ctx, rows[0])
	k.SetGridAttribute(ctx, diversityCapacityKey(owner.Id), 291) // base fell 20; old bonus is still installed
	require.ErrorContains(t, k.CheckReactorDiversityAccounting(ctx), "position mismatch")
	require.NoError(t, k.RebuildReactorDiversity(ctx))
	require.NoError(t, k.CheckReactorDiversityAccounting(ctx))
	require.Equal(t, uint64(289), k.GetGridAttribute(ctx, diversityCapacityKey(owner.Id)))
	queue := k.GetGridCascadeQueueBatch(ctx, 10)
	require.Len(t, queue, 1, "rebuild bonus reductions also owe cascade")
	require.Equal(t, owner.Id, queue[0].ObjectId)
}

func TestDiversityReconstructionFailurePublishesNothing(t *testing.T) {
	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)
	owner := testAppendPlayer(k, ctx, types.Player{})
	for _, reactor := range []string{"3-1", "3-2"} {
		k.SetInfusion(ctx, types.CreateNewInfusion(types.ObjectType_reactor, reactor, "first", owner.Id, 100, math.LegacyZeroDec(), 1))
	}
	k.SetGridAttribute(ctx, diversityCapacityKey(owner.Id), 200)
	require.NoError(t, k.RebuildReactorDiversity(ctx))
	bad := types.CreateNewInfusion(types.ObjectType_reactor, "3-3", "late", owner.Id, 100, math.LegacyNewDec(2), 1)
	k.SetInfusion(ctx, bad)
	before := keepertest.StructsStoreRows(t, ctx)
	events := len(ctx.EventManager().Events())
	require.ErrorContains(t, k.RebuildReactorDiversity(ctx), "commission")
	require.Equal(t, before, keepertest.StructsStoreRows(t, ctx))
	require.Len(t, ctx.EventManager().Events(), events)
}

func TestDiversityReconstructionClearsVanishedPortfolio(t *testing.T) {
	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)
	owner := testAppendPlayer(k, ctx, types.Player{})
	for _, reactor := range []string{"3-1", "3-2"} {
		k.SetInfusion(ctx, types.CreateNewInfusion(types.ObjectType_reactor, reactor, "first", owner.Id, 100, math.LegacyZeroDec(), 1))
	}
	k.SetGridAttribute(ctx, diversityCapacityKey(owner.Id), 200)
	require.NoError(t, k.RebuildReactorDiversity(ctx))
	for _, reactor := range []string{"3-1", "3-2"} {
		k.RemoveInfusion(ctx, reactor, "first")
	}
	k.SetGridAttribute(ctx, diversityCapacityKey(owner.Id), 11) // repaired base is zero
	require.NoError(t, k.RebuildReactorDiversity(ctx))
	require.NoError(t, k.CheckReactorDiversityAccounting(ctx))
	_, found := k.GetOwnerReactorDiversity(ctx, owner.Id)
	require.False(t, found)
	for _, reactor := range []string{"3-1", "3-2"} {
		_, found := k.GetReactorDiversityPosition(ctx, owner.Id, reactor)
		require.False(t, found)
	}
	require.Zero(t, k.GetGridAttribute(ctx, diversityCapacityKey(owner.Id)))
	rows := diversityEvents(t, ctx)
	require.Len(t, rows, 2)
	require.Equal(t, types.OwnerReactorDiversity{OwnerId: owner.Id, Power: "0", SquaredPower: "0"}, rows[1])
}
