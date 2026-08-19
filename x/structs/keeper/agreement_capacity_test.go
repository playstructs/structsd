package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Regression tests for agreement capacity changes.
//
// CapacityDecrease used to release the voided provider cancellation penalty as
// its very first statement, before checking whether the requested decrease was
// possible at all. That payout is a direct bank transfer rather than a deferred
// cache write, so on a rejected request nothing unwound it except the handler
// returning the error and baseapp discarding the transaction's store. These
// tests drive the cache methods directly, without that safety net, so they fail
// if a payout ever moves back ahead of validation.
//
// Neither method re-checked the provider's published capacity or duration range
// either. Only the open path did, so a consumer could decrease capacity to land
// below the advertised minimum, and because a capacity change re-prices the
// unearned span, a decrease toward capacity 1 stretched the remaining duration
// by the old capacity and left the advertised maximum far behind.

// tightenProvider narrows the published ranges, which setupTeardownFixture
// leaves deliberately wide.
func tightenProvider(t *testing.T, f *teardownFixture, capacityMin, capacityMax, durationMin, durationMax uint64) {
	t.Helper()

	provider, found := f.k.GetProvider(f.ctx, f.provider.Id)
	require.True(t, found)

	provider.CapacityMinimum = capacityMin
	provider.CapacityMaximum = capacityMax
	provider.DurationMinimum = durationMin
	provider.DurationMaximum = durationMax

	updated, err := f.k.SetProvider(f.ctx, provider)
	require.NoError(t, err)
	f.provider = updated
}

// advanceBlocks moves the fixture forward so the agreement accrues a served
// span, and with it a nonzero voided penalty. Without this every payout under
// test would be zero and prove nothing.
func advanceBlocks(f *teardownFixture, blocks int64) {
	uctx := sdk.UnwrapSDKContext(f.ctx)
	f.ctx = uctx.WithBlockHeight(uctx.BlockHeight() + blocks)
}

// decreaseCapacity and increaseCapacity deliberately omit the checkpoint the
// handlers run first, so the only thing that can move money is the capacity
// method itself. That isolation is the point: it makes a payout-before-validation
// regression visible without relying on the transaction rollback that masks it in
// production. Nothing is committed on failure, exactly as a rejected message
// would leave it.
func decreaseCapacity(f *teardownFixture, agreementId string, amount uint64) error {
	cc := f.k.NewCurrentContext(sdk.UnwrapSDKContext(f.ctx))
	if err := cc.GetAgreement(agreementId).CapacityDecrease(amount); err != nil {
		return err
	}
	cc.CommitAll()
	return nil
}

func increaseCapacity(f *teardownFixture, agreementId string, amount uint64) error {
	cc := f.k.NewCurrentContext(sdk.UnwrapSDKContext(f.ctx))
	if err := cc.GetAgreement(agreementId).CapacityIncrease(amount); err != nil {
		return err
	}
	cc.CommitAll()
	return nil
}

// checkpointedDecrease and checkpointedIncrease mirror the handlers exactly,
// checkpointing before the load changes. Anything asserting on pool solvency has
// to go through these: skipping the checkpoint bills the new load across the span
// the old one was serving, which is a state the chain never actually reaches.
func checkpointedDecrease(f *teardownFixture, agreementId string, amount uint64) error {
	cc := f.k.NewCurrentContext(sdk.UnwrapSDKContext(f.ctx))
	agreement := cc.GetAgreement(agreementId)
	if err := agreement.GetProvider().Checkpoint(); err != nil {
		return err
	}
	if err := agreement.CapacityDecrease(amount); err != nil {
		return err
	}
	cc.CommitAll()
	return nil
}

func checkpointedIncrease(f *teardownFixture, agreementId string, amount uint64) error {
	cc := f.k.NewCurrentContext(sdk.UnwrapSDKContext(f.ctx))
	agreement := cc.GetAgreement(agreementId)
	if err := agreement.GetProvider().Checkpoint(); err != nil {
		return err
	}
	if err := agreement.CapacityIncrease(amount); err != nil {
		return err
	}
	cc.CommitAll()
	return nil
}

// capacityPools is a snapshot of everything a stray payout would disturb.
type capacityPools struct {
	collateral math.Int
	earnings   math.Int
	capacity   uint64
	startBlock uint64
	load       uint64
}

func snapshotPools(t *testing.T, f *teardownFixture, agreementId string) capacityPools {
	t.Helper()

	agreement, found := f.k.GetAgreement(f.ctx, agreementId)
	require.True(t, found)

	return capacityPools{
		collateral: f.balance(f.collateralAcc),
		earnings:   f.balance(f.earningsAcc),
		capacity:   agreement.Capacity,
		startBlock: agreement.StartBlock,
		load:       f.agreementLoad(),
	}
}

func requireUntouched(t *testing.T, f *teardownFixture, agreementId string, before capacityPools) {
	t.Helper()

	after := snapshotPools(t, f, agreementId)
	require.True(t, before.collateral.Equal(after.collateral),
		"collateral pool moved on a rejected change: %s -> %s", before.collateral, after.collateral)
	require.True(t, before.earnings.Equal(after.earnings),
		"earnings pool moved on a rejected change: %s -> %s", before.earnings, after.earnings)
	require.Equal(t, before.capacity, after.capacity, "capacity changed on a rejected change")
	require.Equal(t, before.startBlock, after.startBlock, "start block reset on a rejected change")
	require.Equal(t, before.load, after.load, "provider load changed on a rejected change")
}

// TestCapacityChange_RejectedDecreaseMovesNoMoney is the ordering regression. A
// decrease that takes the agreement to zero capacity has always been rejected;
// what used to happen anyway was the penalty transfer.
func TestCapacityChange_RejectedDecreaseMovesNoMoney(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	agreement, _ := f.openAgreement(t, 100, 50)
	advanceBlocks(f, 10)

	before := snapshotPools(t, f, agreement.Id)
	require.True(t, before.collateral.IsPositive(), "the pool must hold collateral for this to prove anything")

	err := decreaseCapacity(f, agreement.Id, 100)
	require.Error(t, err)
	requireUntouched(t, f, agreement.Id, before)
}

// TestCapacityChange_RejectedIncreaseMovesNoMoney is the same property on the
// increase path, where the payout sat behind only the substation check.
func TestCapacityChange_RejectedIncreaseMovesNoMoney(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	agreement, _ := f.openAgreement(t, 100, 50)
	tightenProvider(t, f, 1, 150, 1, 1000000)
	advanceBlocks(f, 10)

	before := snapshotPools(t, f, agreement.Id)

	err := increaseCapacity(f, agreement.Id, 100)
	require.ErrorContains(t, err, "maximum capacity")
	requireUntouched(t, f, agreement.Id, before)
}

// TestCapacityChange_NoOpRejected covers a change of zero, which used to pass
// every guard and still sweep the penalty and reset the accounting window.
func TestCapacityChange_NoOpRejected(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	agreement, _ := f.openAgreement(t, 100, 50)
	advanceBlocks(f, 10)

	before := snapshotPools(t, f, agreement.Id)

	err := decreaseCapacity(f, agreement.Id, 0)
	require.ErrorContains(t, err, "would do nothing")
	requireUntouched(t, f, agreement.Id, before)

	err = increaseCapacity(f, agreement.Id, 0)
	require.ErrorContains(t, err, "would do nothing")
	requireUntouched(t, f, agreement.Id, before)
}

// TestCapacityChange_RespectsPublishedCapacityRange pins that a modification
// cannot land outside the range that gated opening the agreement.
func TestCapacityChange_RespectsPublishedCapacityRange(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	agreement, _ := f.openAgreement(t, 100, 50)
	tightenProvider(t, f, 50, 150, 1, 1000000)
	advanceBlocks(f, 10)

	before := snapshotPools(t, f, agreement.Id)

	err := decreaseCapacity(f, agreement.Id, 60)
	require.ErrorContains(t, err, "minimum capacity")
	requireUntouched(t, f, agreement.Id, before)

	err = increaseCapacity(f, agreement.Id, 60)
	require.ErrorContains(t, err, "maximum capacity")
	requireUntouched(t, f, agreement.Id, before)

	// A change that stays inside the published range still works.
	require.NoError(t, decreaseCapacity(f, agreement.Id, 40))

	updated, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, found)
	require.Equal(t, uint64(60), updated.Capacity)
}

// TestCapacityChange_RespectsPublishedDurationRange is the escape this change
// closes. A capacity change re-prices the unearned span, so dropping to the
// minimum capacity multiplies the remaining duration by the old capacity and
// used to commit an agreement running far past the advertised maximum.
func TestCapacityChange_RespectsPublishedDurationRange(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	agreement, _ := f.openAgreement(t, 100, 50)
	tightenProvider(t, f, 1, 10000, 1, 100)
	advanceBlocks(f, 10)

	before := snapshotPools(t, f, agreement.Id)

	// 40 blocks left at capacity 100 becomes 4000 blocks at capacity 1.
	err := decreaseCapacity(f, agreement.Id, 99)
	require.ErrorContains(t, err, "maximum duration")
	requireUntouched(t, f, agreement.Id, before)
}

// TestCapacityChange_ValidDecreasePaysVoidedPenaltyOnce pins the accounting a
// successful change performs, including that the penalty is a rolling forfeit
// measured from the last reset rather than from the agreement's own start.
func TestCapacityChange_ValidDecreasePaysVoidedPenaltyOnce(t *testing.T) {
	const rate, capacity, duration = 10, 100, 50
	f := setupTeardownFixture(t, rate, "0.5", "0.25")

	agreement, collateral := f.openAgreement(t, capacity, duration)
	endBlock := agreement.EndBlock

	advanceBlocks(f, 10)

	before := snapshotPools(t, f, agreement.Id)

	// Service starts in the opening block, so ten blocks on the clock is ten
	// blocks served. Derive it rather than assume it.
	currentBlock := uint64(sdk.UnwrapSDKContext(f.ctx).BlockHeight())
	served := currentBlock - agreement.StartBlock
	require.Equal(t, uint64(10), served, "fixture assumption about the served span")

	// Half the served value, being the published provider cancellation penalty on
	// time the provider did serve and did not cancel out of.
	expectedPenalty := math.NewIntFromUint64(served * rate * capacity / 2)

	require.NoError(t, decreaseCapacity(f, agreement.Id, 50))

	require.True(t, f.balance(f.earningsAcc).Sub(before.earnings).Equal(expectedPenalty),
		"earnings should have grown by exactly the voided penalty %s, grew by %s",
		expectedPenalty, f.balance(f.earningsAcc).Sub(before.earnings))
	require.True(t, before.collateral.Sub(f.balance(f.collateralAcc)).Equal(expectedPenalty),
		"the penalty should have come out of the collateral pool and nothing else")

	updated, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, found)
	require.Equal(t, uint64(50), updated.Capacity)

	// The window is re-based on the current height, and the unearned span is
	// re-priced: 40 blocks at capacity 100 buys 80 blocks at capacity 50.
	require.Equal(t, currentBlock, updated.StartBlock)
	require.Equal(t, currentBlock+((endBlock-currentBlock)*capacity/50), updated.EndBlock)

	require.Equal(t, uint64(50), f.agreementLoad(), "provider load should drop by the decrease exactly once")

	// Immediately again: no time has been served since the reset, so there is no
	// penalty left to void and no money moves.
	poolsAfterFirst := snapshotPools(t, f, agreement.Id)
	require.NoError(t, decreaseCapacity(f, agreement.Id, 10))

	require.True(t, f.balance(f.earningsAcc).Equal(poolsAfterFirst.earnings),
		"a second change in the same block must not sweep the penalty again")
	require.True(t, f.balance(f.collateralAcc).Equal(poolsAfterFirst.collateral),
		"a second change in the same block must not touch the collateral pool")

	require.True(t, f.balance(f.earningsAcc).LTE(collateral),
		"the provider cannot earn more than was ever deposited against them")
}

// TestCapacityChange_SolvencyInvariantHoldsAcrossCapacityChanges runs the
// registered invariant across a sequence of changes, so a future regression in
// the rescaling arithmetic is caught as undercollateralization rather than as a
// silently wrong end block.
//
// The partial provider penalty matters: it means Checkpoint and the voided
// penalty both move money, which is the case where the checkpoint clock and the
// agreement's service window have to agree block for block.
func TestCapacityChange_SolvencyInvariantHoldsAcrossCapacityChanges(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	invariant := keeperlib.ProviderCollateralSolvencyInvariant(f.k)

	requireSolvent := func(stage string) {
		msg, broken := invariant(sdk.UnwrapSDKContext(f.ctx))
		require.False(t, broken, "%s: %s", stage, msg)
	}

	first, _ := f.openAgreement(t, 100, 50)
	second, _ := f.openAgreement(t, 250, 40)
	requireSolvent("two agreements open")

	advanceBlocks(f, 10)
	require.NoError(t, checkpointedDecrease(f, first.Id, 50))
	requireSolvent("after decreasing the first")

	advanceBlocks(f, 5)
	require.NoError(t, checkpointedIncrease(f, second.Id, 100))
	requireSolvent("after increasing the second")

	advanceBlocks(f, 5)
	_, err := f.ms.AgreementClose(f.ctx, &types.MsgAgreementClose{
		Creator:     f.consumer.Creator,
		AgreementId: first.Id,
	})
	require.NoError(t, err)
	requireSolvent("after closing the first")

	_, err = f.ms.AgreementClose(f.ctx, &types.MsgAgreementClose{
		Creator:     f.consumer.Creator,
		AgreementId: second.Id,
	})
	require.NoError(t, err)
	requireSolvent("after closing the second")
}

// TestCapacityChange_CheckpointBillsOnlyServedBlocks pins the alignment between
// the checkpoint clock and an agreement's service window.
//
// AgreementOpen raises the provider's load immediately and checkpoints the
// provider at the opening height, and Checkpoint bills aggregate load from the
// checkpoint block. So service has to start in the opening block: when StartBlock
// was a block later, every agreement's first checkpoint span billed one block the
// consumer never received and left the pool short by
// capacity * rate * (1 - penalty), which the invariant reported as insolvency.
//
// A checkpoint on its own must leave the pool exactly solvent, with no dust either
// way, so this asserts the deficit is zero rather than merely non-breaking.
func TestCapacityChange_CheckpointBillsOnlyServedBlocks(t *testing.T) {
	const rate, capacity, duration = 10, 100, 50
	f := setupTeardownFixture(t, rate, "0.5", "0.25")

	agreement, collateral := f.openAgreement(t, capacity, duration)
	require.Equal(t, uint64(sdk.UnwrapSDKContext(f.ctx).BlockHeight()), agreement.StartBlock,
		"service must start in the block the agreement opens, matching the checkpoint clock")

	advanceBlocks(f, 10)

	cc := f.k.NewCurrentContext(sdk.UnwrapSDKContext(f.ctx))
	require.NoError(t, cc.GetProvider(f.provider.Id).Checkpoint())
	cc.CommitAll()

	invariant := keeperlib.ProviderCollateralSolvencyInvariant(f.k)
	msg, broken := invariant(sdk.UnwrapSDKContext(f.ctx))
	require.False(t, broken, "a checkpoint must not leave the pool short: %s", msg)

	// Exactly the served value at the non-penalty share, and no more: the penalty
	// share stays in the pool until the agreement ends.
	served := math.NewInt(10 * rate * capacity)
	require.True(t, f.balance(f.earningsAcc).Equal(served.QuoRaw(2)),
		"expected %s swept, got %s", served.QuoRaw(2), f.balance(f.earningsAcc))
	require.True(t, f.balance(f.collateralAcc).Equal(collateral.Sub(served.QuoRaw(2))),
		"the collateral pool should hold everything the sweep did not take")
}

// TestCapacityChange_HandlerCheckpointsBeforeRaisingLoad drives the message
// server rather than the cache, because the checkpoint a capacity change depends
// on lives in the handler and nothing else here exercises it.
//
// The rejection tests above deliberately skip that checkpoint, and
// TestCapacityChange_CheckpointBillsOnlyServedBlocks calls Checkpoint directly,
// so deleting agreement.GetProvider().Checkpoint() from the handler used to leave
// the entire suite green. What the deletion costs is not a missing call but
// retroactive billing: the checkpoint block stays at the opening height while
// load rises, so the next checkpoint charges the new capacity across the span the
// old one was serving, out of a collateral pool shared with every other agreement
// of that provider.
//
// The provider penalty is zero so the handler's checkpoint is the only thing that
// can move money — PayoutVoidedProviderCancellationPenalty prices off it and pays
// nothing — which makes the first assertion read zero rather than merely low if
// the checkpoint goes missing.
//
// The duration is long on purpose. An increase re-prices the unearned span as
// remaining*old/new, so enlarging a short agreement is refused for a rescaled
// duration below the provider's published minimum before any billing happens, and
// the test would pass whatever the checkpoint did.
func TestCapacityChange_HandlerCheckpointsBeforeRaisingLoad(t *testing.T) {
	const rate, capacity, duration, servedBlocks = 10, 100, 5000, 10
	f := setupTeardownFixture(t, rate, "0", "0")

	agreement, collateral := f.openAgreement(t, capacity, duration)
	advanceBlocks(f, servedBlocks)

	_, err := f.ms.AgreementCapacityIncrease(f.ctx, &types.MsgAgreementCapacityIncrease{
		Creator:          f.consumer.Creator,
		AgreementId:      agreement.Id,
		CapacityIncrease: capacity,
	})
	require.NoError(t, err)

	// The handler's checkpoint settled the span served at the old capacity, and
	// nothing beyond it.
	firstSpan := math.NewInt(servedBlocks * rate * capacity)
	require.True(t, f.balance(f.earningsAcc).Equal(firstSpan),
		"the handler must sweep the served span at the old capacity: expected %s, got %s",
		firstSpan, f.balance(f.earningsAcc))

	// The second span is then billed at the new capacity over its own blocks.
	// Without the handler's checkpoint this charges the doubled load across both
	// spans and takes one span too much out of the pool.
	advanceBlocks(f, servedBlocks)

	cc := f.k.NewCurrentContext(sdk.UnwrapSDKContext(f.ctx))
	require.NoError(t, cc.GetProvider(f.provider.Id).Checkpoint())
	cc.CommitAll()

	total := firstSpan.Add(math.NewInt(servedBlocks * rate * capacity * 2))
	require.True(t, f.balance(f.earningsAcc).Equal(total),
		"each capacity must be billed over its own span only: expected %s, got %s",
		total, f.balance(f.earningsAcc))
	require.True(t, f.balance(f.collateralAcc).Equal(collateral.Sub(total)),
		"the collateral pool should hold everything the sweeps did not take")

	// Overbilling surfaces as insolvency rather than as a wrong number, so assert
	// it the way the chain itself would notice.
	invariant := keeperlib.ProviderCollateralSolvencyInvariant(f.k)
	msg, broken := invariant(sdk.UnwrapSDKContext(f.ctx))
	require.False(t, broken, "retroactive billing leaves the pool short: %s", msg)
}

// TestCapacityChange_HandlerCheckpointsBeforeLoweringLoad is the same property on
// the decrease handler, which has the identical shape.
//
// Note that the invariant cannot stand in for the earnings assertion here. A
// stale checkpoint underbills a decrease rather than overbilling it, which leaves
// the pool over-funded instead of short — and revenue stranded there is
// unreachable, since both Checkpoint and WithdrawBalanceAndCommit are computed
// from load. Only the swept amount shows it.
func TestCapacityChange_HandlerCheckpointsBeforeLoweringLoad(t *testing.T) {
	const rate, capacity, duration, servedBlocks = 10, 200, 5000, 10
	f := setupTeardownFixture(t, rate, "0", "0")

	agreement, collateral := f.openAgreement(t, capacity, duration)
	advanceBlocks(f, servedBlocks)

	_, err := f.ms.AgreementCapacityDecrease(f.ctx, &types.MsgAgreementCapacityDecrease{
		Creator:          f.consumer.Creator,
		AgreementId:      agreement.Id,
		CapacityDecrease: capacity / 2,
	})
	require.NoError(t, err)

	firstSpan := math.NewInt(servedBlocks * rate * capacity)
	require.True(t, f.balance(f.earningsAcc).Equal(firstSpan),
		"the handler must sweep the served span at the old capacity: expected %s, got %s",
		firstSpan, f.balance(f.earningsAcc))

	advanceBlocks(f, servedBlocks)

	cc := f.k.NewCurrentContext(sdk.UnwrapSDKContext(f.ctx))
	require.NoError(t, cc.GetProvider(f.provider.Id).Checkpoint())
	cc.CommitAll()

	total := firstSpan.Add(math.NewInt(servedBlocks * rate * capacity / 2))
	require.True(t, f.balance(f.earningsAcc).Equal(total),
		"each capacity must be billed over its own span only: expected %s, got %s",
		total, f.balance(f.earningsAcc))
	require.True(t, f.balance(f.collateralAcc).Equal(collateral.Sub(total)),
		"the collateral pool should hold everything the sweeps did not take")

	invariant := keeperlib.ProviderCollateralSolvencyInvariant(f.k)
	msg, broken := invariant(sdk.UnwrapSDKContext(f.ctx))
	require.False(t, broken, "a decrease must leave the pool solvent: %s", msg)
}
