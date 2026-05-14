package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

// TestInfusionMaturitySweepQueueEnqueueDequeue covers the time-boundary
// semantics of the queue API: rows with CompletionTime <= blockTime are
// returned and removed; rows in the future are left alone.
func TestInfusionMaturitySweepQueueEnqueueDequeue(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	t0 := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	earlier := t0.Add(-time.Hour)
	exact := t0
	later := t0.Add(time.Hour)

	k.EnqueueInfusionMaturitySweep(ctx, earlier, "reactor-1-cosmos1aaa")
	k.EnqueueInfusionMaturitySweep(ctx, exact, "reactor-1-cosmos1bbb")
	k.EnqueueInfusionMaturitySweep(ctx, later, "reactor-1-cosmos1ccc")

	// Sweeping at t0 should drain the earlier+exact rows but leave the future row.
	matured := k.DequeueMatureInfusionSweeps(ctx, t0)
	require.ElementsMatch(t, []string{"reactor-1-cosmos1aaa", "reactor-1-cosmos1bbb"}, matured)

	// The remaining row should still be in the queue.
	remaining := k.GetInfusionMaturitySweepQueueExport(ctx)
	require.Len(t, remaining, 1)

	// And sweeping past it consumes it.
	matured = k.DequeueMatureInfusionSweeps(ctx, later.Add(time.Second))
	require.Equal(t, []string{"reactor-1-cosmos1ccc"}, matured)
	require.Empty(t, k.GetInfusionMaturitySweepQueueExport(ctx))
}

// TestInfusionMaturitySweepQueueDeduplicates verifies that multiple UBD
// entries on the same (delegator, validator) pair maturing in the same block
// only trigger one reconciliation entry (so the EndBlocker doesn't reconcile
// the same infusion N times).
func TestInfusionMaturitySweepQueueDeduplicates(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	t0 := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	infusionId := "reactor-1-cosmos1same"

	k.EnqueueInfusionMaturitySweep(ctx, t0, infusionId)
	k.EnqueueInfusionMaturitySweep(ctx, t0.Add(time.Second), infusionId)
	k.EnqueueInfusionMaturitySweep(ctx, t0.Add(2*time.Second), infusionId)

	matured := k.DequeueMatureInfusionSweeps(ctx, t0.Add(time.Hour))
	require.Equal(t, []string{infusionId}, matured)
	require.Empty(t, k.GetInfusionMaturitySweepQueueExport(ctx))
}

// TestReactorInfusionUnbondingEnqueuesPerEntry asserts that each
// UnbondingDelegationEntry seen by ReactorInfusionUnbonding produces a
// queue row at its CompletionTime. This is the regression test for the
// missing AfterUnbondingComplete hook.
func TestReactorInfusionUnbondingEnqueuesPerEntry(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	playerAcc := sdk.AccAddress("playera_padding____________________01")
	player := types.Player{Creator: playerAcc.String(), PrimaryAddress: playerAcc.String()}
	player = testAppendPlayer(k, ctx, player)

	validatorAddress := sdk.ValAddress(playerAcc.Bytes())
	testAddValidator(k, validatorAddress, math.NewInt(1000))
	reactor := testAppendReactor(k, ctx, types.Reactor{
		Validator:  validatorAddress.String(),
		RawAddress: validatorAddress.Bytes(),
	})

	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)

	// Two UBD entries with distinct CompletionTimes against the same pair.
	completion1 := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	completion2 := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	require.NoError(t, mock.SetUnbondingDelegation(ctx, stakingtypes.UnbondingDelegation{
		DelegatorAddress: playerAcc.String(),
		ValidatorAddress: validatorAddress.String(),
		Entries: []stakingtypes.UnbondingDelegationEntry{
			{CompletionTime: completion1, Balance: math.NewInt(40), InitialBalance: math.NewInt(40), UnbondingId: 7},
			{CompletionTime: completion2, Balance: math.NewInt(60), InitialBalance: math.NewInt(60), UnbondingId: 8},
		},
	}))

	// The hook is keyed by unbondingId; either id maps back to the same UBD
	// in the mock and triggers the per-entry enqueue loop.
	k.ReactorInfusionUnbonding(ctx, 7)

	queue := k.GetInfusionMaturitySweepQueueExport(ctx)
	require.Len(t, queue, 2)

	// Defusing should reflect the sum of both UBD entry balances.
	infusion, found := k.GetInfusion(ctx, reactor.Id, playerAcc.String())
	require.True(t, found)
	require.Equal(t, uint64(100), infusion.Defusing)

	// Re-running the same hook on the same UBD should be idempotent
	// (overwrites existing rows; queue length stays at 2).
	k.ReactorInfusionUnbonding(ctx, 7)
	require.Len(t, k.GetInfusionMaturitySweepQueueExport(ctx), 2)
}

// TestProcessInfusionMaturitySweepClearsDefusing simulates the production bug:
// an infusion has Defusing > 0 because AfterUnbondingInitiated fired, but the
// underlying UBD has since matured silently. The structs EndBlocker sweep at
// the maturity block should reconcile Defusing back to zero.
func TestProcessInfusionMaturitySweepClearsDefusing(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	playerAcc := sdk.AccAddress("playerb_padding____________________02")
	player := testAppendPlayer(k, ctx, types.Player{Creator: playerAcc.String(), PrimaryAddress: playerAcc.String()})

	validatorAddress := sdk.ValAddress(playerAcc.Bytes())
	testAddValidator(k, validatorAddress, math.NewInt(1000))
	reactor := testAppendReactor(k, ctx, types.Reactor{
		Validator:  validatorAddress.String(),
		RawAddress: validatorAddress.Bytes(),
	})

	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)

	completion := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	require.NoError(t, mock.SetUnbondingDelegation(ctx, stakingtypes.UnbondingDelegation{
		DelegatorAddress: playerAcc.String(),
		ValidatorAddress: validatorAddress.String(),
		Entries: []stakingtypes.UnbondingDelegationEntry{
			{CompletionTime: completion, Balance: math.NewInt(50), InitialBalance: math.NewInt(50), UnbondingId: 1},
		},
	}))

	// Trigger the AfterUnbondingInitiated path so Defusing gets set + queue row written.
	k.ReactorInfusionUnbonding(ctx, 1)

	infusion, found := k.GetInfusion(ctx, reactor.Id, playerAcc.String())
	require.True(t, found)
	require.Equal(t, uint64(50), infusion.Defusing)
	require.Len(t, k.GetInfusionMaturitySweepQueueExport(ctx), 1)
	_ = player

	// Simulate Cosmos staking silently completing the unbonding: the UBD is gone.
	mock.MatureUnbondingDelegation(playerAcc, validatorAddress)

	// Advance the block-time on the context so the structs sweep treats the
	// completion timestamp as mature. The keeper reads HeaderInfo().Time, so
	// that's the field we have to override.
	maturedTime := completion.Add(time.Minute)
	maturedCtx := sdk.UnwrapSDKContext(ctx).WithHeaderInfo(header.Info{Time: maturedTime})

	k.ProcessInfusionMaturitySweep(maturedCtx)

	infusion, found = k.GetInfusion(ctx, reactor.Id, playerAcc.String())
	require.True(t, found)
	require.Equal(t, uint64(0), infusion.Defusing, "matured UBD should clear Defusing back to zero")
	require.Empty(t, k.GetInfusionMaturitySweepQueueExport(ctx), "drained queue rows should be deleted")
}

// TestProcessInfusionMaturitySweepLeavesUnmaturedAlone confirms that an
// infusion whose UBD has not yet matured is not touched by the sweep.
func TestProcessInfusionMaturitySweepLeavesUnmaturedAlone(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	playerAcc := sdk.AccAddress("playerc_padding____________________03")
	testAppendPlayer(k, ctx, types.Player{Creator: playerAcc.String(), PrimaryAddress: playerAcc.String()})

	validatorAddress := sdk.ValAddress(playerAcc.Bytes())
	testAddValidator(k, validatorAddress, math.NewInt(1000))
	reactor := testAppendReactor(k, ctx, types.Reactor{
		Validator:  validatorAddress.String(),
		RawAddress: validatorAddress.Bytes(),
	})

	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)

	future := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, mock.SetUnbondingDelegation(ctx, stakingtypes.UnbondingDelegation{
		DelegatorAddress: playerAcc.String(),
		ValidatorAddress: validatorAddress.String(),
		Entries: []stakingtypes.UnbondingDelegationEntry{
			{CompletionTime: future, Balance: math.NewInt(75), InitialBalance: math.NewInt(75), UnbondingId: 1},
		},
	}))
	k.ReactorInfusionUnbonding(ctx, 1)

	// Sweep at a time before maturity: nothing should drain.
	earlyCtx := sdk.UnwrapSDKContext(ctx).WithHeaderInfo(header.Info{Time: future.Add(-24 * time.Hour)})
	k.ProcessInfusionMaturitySweep(earlyCtx)

	infusion, found := k.GetInfusion(ctx, reactor.Id, playerAcc.String())
	require.True(t, found)
	require.Equal(t, uint64(75), infusion.Defusing, "unmatured UBD should leave Defusing untouched")
	require.Len(t, k.GetInfusionMaturitySweepQueueExport(ctx), 1, "queue row should remain for future sweep")
}

// TestReconcileInfusionForDelegationIdempotent exercises the helper directly:
// running it twice on the same state produces the same result.
func TestReconcileInfusionForDelegationIdempotent(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	playerAcc := sdk.AccAddress("playerd_padding____________________04")
	testAppendPlayer(k, ctx, types.Player{Creator: playerAcc.String(), PrimaryAddress: playerAcc.String()})

	validatorAddress := sdk.ValAddress(playerAcc.Bytes())
	testAddValidator(k, validatorAddress, math.NewInt(1000))
	reactor := testAppendReactor(k, ctx, types.Reactor{
		Validator:  validatorAddress.String(),
		RawAddress: validatorAddress.Bytes(),
	})

	// Seed an Infusion that thinks 100 is defusing, but no UBD on the staking side.
	infusion := types.Infusion{
		DestinationType: types.ObjectType_reactor,
		DestinationId:   reactor.Id,
		Address:         playerAcc.String(),
		Defusing:        100,
		Commission:      math.LegacyZeroDec(),
	}
	testAppendInfusion(k, ctx, infusion)

	k.ReconcileInfusionForDelegation(ctx, playerAcc, validatorAddress)
	got, found := k.GetInfusion(ctx, reactor.Id, playerAcc.String())
	require.True(t, found)
	require.Equal(t, uint64(0), got.Defusing)

	// Second pass should be a no-op state-wise.
	k.ReconcileInfusionForDelegation(ctx, playerAcc, validatorAddress)
	got, found = k.GetInfusion(ctx, reactor.Id, playerAcc.String())
	require.True(t, found)
	require.Equal(t, uint64(0), got.Defusing)
}

// TestReconcileInfusionForDelegationNoReactor exercises the silent-skip path
// when a reconciliation request lands on a validator with no reactor record.
// This guards against panics if the queue row outlives the reactor for any
// reason (genesis race conditions, manual store edits, etc.).
func TestReconcileInfusionForDelegationNoReactor(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	playerAcc := sdk.AccAddress("playere_padding____________________05")
	validatorAddress := sdk.ValAddress(playerAcc.Bytes())

	require.NotPanics(t, func() {
		k.ReconcileInfusionForDelegation(ctx, playerAcc, validatorAddress)
	})
}
