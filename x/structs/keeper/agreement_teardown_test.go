package keeper_test

import (
	"context"
	"strings"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Regression tests for the reciprocal teardown that settled one agreement twice.
//
// Agreement teardown destroys the allocation and allocation teardown settles the
// agreement, and CurrentContext hands both legs the same AgreementCache, so every
// close used to pay out twice out of a collateral pool shared by all of that
// provider's agreements. Expiry was the worst of them: it paid the consumer a
// provider-cancellation penalty out of a share already exhausted by checkpoints,
// so every expiry drew on somebody else's collateral.
//
// These tests build a real agreement through AgreementOpen so the agreement and
// allocation ids line up the way the chain generates them. Fixtures that invent
// ids like "agreement-1" never trip the recursion at all.

const teardownDenom = "ualpha"

type teardownFixture struct {
	k   keeperlib.Keeper
	ms  types.MsgServer
	ctx context.Context

	consumer      types.Player
	consumerAcc   sdk.AccAddress
	ownerAcc      sdk.AccAddress
	provider      types.Provider
	substationId  string
	collateralAcc sdk.AccAddress
	earningsAcc   sdk.AccAddress
}

// setupTeardownFixture builds a funded consumer and an open-market provider whose
// substation has room for agreements.
func setupTeardownFixture(t *testing.T, rate int64, providerPenalty, consumerPenalty string) *teardownFixture {
	t.Helper()

	k, ms, ctx := setupMsgServer(t)

	consumerAcc := sdk.AccAddress("consumer1234567890123456789012345678")
	consumer := testAppendPlayer(k, ctx, types.Player{
		Creator:        consumerAcc.String(),
		PrimaryAddress: consumerAcc.String(),
	})

	ownerAcc := sdk.AccAddress("provowner1234567890123456789012345678")
	owner := testAppendPlayer(k, ctx, types.Player{
		Creator:        ownerAcc.String(),
		PrimaryAddress: ownerAcc.String(),
	})

	// A substation with plenty of headroom for the agreements under test.
	substation, _, err := testAppendSubstation(k, ctx, types.Allocation{}, owner)
	require.NoError(t, err)
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, substation.Id), 100000)

	providerPenaltyDec, err := math.LegacyNewDecFromStr(providerPenalty)
	require.NoError(t, err)
	consumerPenaltyDec, err := math.LegacyNewDecFromStr(consumerPenalty)
	require.NoError(t, err)

	// Go through NewProvider so the collateral and earnings pool accounts exist
	// and the checkpoint block starts at the current height, exactly as
	// ProviderCreate would leave them.
	cc := k.NewCurrentContext(ctx)
	providerCache := cc.NewProvider(types.Provider{
		Owner:                       owner.Id,
		Creator:                     owner.Creator,
		SubstationId:                substation.Id,
		Rate:                        sdk.NewCoin(teardownDenom, math.NewInt(rate)),
		AccessPolicy:                types.ProviderAccessPolicy_openMarket,
		CapacityMinimum:             1,
		CapacityMaximum:             10000,
		DurationMinimum:             1,
		DurationMaximum:             1000000,
		ProviderCancellationPenalty: providerPenaltyDec,
		ConsumerCancellationPenalty: consumerPenaltyDec,
	})
	cc.CommitAll()

	provider, found := k.GetProvider(ctx, providerCache.GetProviderId())
	require.True(t, found)

	return &teardownFixture{
		k:             k,
		ms:            ms,
		ctx:           ctx,
		consumer:      consumer,
		consumerAcc:   consumerAcc,
		ownerAcc:      ownerAcc,
		provider:      provider,
		substationId:  substation.Id,
		collateralAcc: keeperlib.GetProviderCollateralPoolLocation(provider.Id),
		earningsAcc:   keeperlib.GetProviderEarningsPoolLocation(provider.Id),
	}
}

func (f *teardownFixture) fund(t *testing.T, account sdk.AccAddress, amount int64) {
	t.Helper()

	coins := sdk.NewCoins(sdk.NewCoin(teardownDenom, math.NewInt(amount)))
	require.NoError(t, f.k.BankKeeper().MintCoins(f.ctx, types.ModuleName, coins))
	require.NoError(t, f.k.BankKeeper().SendCoinsFromModuleToAccount(f.ctx, types.ModuleName, account, coins))
}

func (f *teardownFixture) balance(account sdk.AccAddress) math.Int {
	return f.k.BankKeeper().SpendableCoin(f.ctx, account, teardownDenom).Amount
}

func (f *teardownFixture) agreementLoad() uint64 {
	return f.k.GetGridAttribute(f.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, f.provider.Id))
}

// openAgreement funds the consumer for the collateral and opens an agreement,
// returning it along with the collateral that was deposited.
func (f *teardownFixture) openAgreement(t *testing.T, capacity uint64, duration uint64) (types.Agreement, math.Int) {
	t.Helper()

	collateral := math.NewIntFromUint64(capacity).Mul(math.NewIntFromUint64(duration)).Mul(f.provider.Rate.Amount)
	f.fund(t, f.consumerAcc, collateral.Int64())

	before := f.k.GetAllAgreementIdByProviderIndex(f.ctx, f.provider.Id)

	_, err := f.ms.AgreementOpen(f.ctx, &types.MsgAgreementOpen{
		Creator:    f.consumer.Creator,
		ProviderId: f.provider.Id,
		Capacity:   capacity,
		Duration:   duration,
	})
	require.NoError(t, err)

	after := f.k.GetAllAgreementIdByProviderIndex(f.ctx, f.provider.Id)
	require.Len(t, after, len(before)+1, "expected exactly one new agreement")

	var agreementId string
	for _, candidate := range after {
		known := false
		for _, existing := range before {
			if existing == candidate {
				known = true
				break
			}
		}
		if !known {
			agreementId = candidate
		}
	}

	agreement, found := f.k.GetAgreement(f.ctx, agreementId)
	require.True(t, found)

	// The recursion depended on the agreement id being derivable from the
	// allocation index. If that ever stops being true these tests stop testing
	// anything, so assert it.
	allocation, allocationFound := f.k.GetAllocation(f.ctx, agreement.AllocationId)
	require.True(t, allocationFound)
	require.Equal(t, keeperlib.GetObjectID(types.ObjectType_agreement, allocation.Index), agreement.Id,
		"agreement id must be derived from its allocation index")

	return agreement, collateral
}

// requireSettledOnce asserts an agreement and every trace of it is gone.
func (f *teardownFixture) requireSettledOnce(t *testing.T, agreement types.Agreement) {
	t.Helper()

	_, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.False(t, found, "agreement should be removed from the store")

	require.NotContains(t, f.k.GetAllAgreementIdByProviderIndex(f.ctx, f.provider.Id), agreement.Id,
		"agreement should be out of the provider index")
	require.NotContains(t, f.k.GetAllAgreementIdByExpirationIndex(f.ctx, agreement.EndBlock), agreement.Id,
		"agreement should be out of the expiration index")

	_, allocationFound := f.k.GetAllocation(f.ctx, agreement.AllocationId)
	require.False(t, allocationFound, "backing allocation should be removed")
}

// TestTeardown_ConsumerCloseSettlesOnce is the direct regression on
// PrematureCloseByConsumer, which paid the remaining collateral and then paid it
// again through the allocation's re-entrant settlement.
func TestTeardown_ConsumerCloseSettlesOnce(t *testing.T) {
	// No penalties, so the consumer is owed exactly the unearned collateral and
	// a double payout is unmistakable.
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 50
	agreement, collateral := f.openAgreement(t, capacity, duration)

	require.Equal(t, collateral, f.balance(f.collateralAcc), "collateral should be sitting in the pool")
	require.Equal(t, math.ZeroInt(), f.balance(f.consumerAcc))
	require.Equal(t, uint64(capacity), f.agreementLoad())

	_, err := f.ms.AgreementClose(f.ctx, &types.MsgAgreementClose{
		Creator:     f.consumer.Creator,
		AgreementId: agreement.Id,
	})
	require.NoError(t, err)

	// Closing in the same block it opened means nothing was earned, so the whole
	// collateral comes back. Twice that would be the bug.
	require.Equal(t, collateral, f.balance(f.consumerAcc), "consumer must be refunded exactly once")
	require.Equal(t, math.ZeroInt(), f.balance(f.collateralAcc), "pool should be drained to zero, not overdrawn")
	require.Equal(t, uint64(0), f.agreementLoad(), "provider load must be decremented exactly once")

	f.requireSettledOnce(t, agreement)
}

// TestTeardown_AllocationDrivenSettlesOnce covers PrematureCloseByAllocation
// through the entry that actually reaches it: allocation teardown driven from
// outside the agreement, which is what a grid brownout and a destroyed struct do.
// This is the path with no transaction to roll it back, so single settlement here
// matters more than anywhere else.
func TestTeardown_AllocationDrivenSettlesOnce(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 50
	agreement, collateral := f.openAgreement(t, capacity, duration)

	cc := f.k.NewCurrentContext(f.ctx)
	cc.DestroyMultipleAllocations([]string{agreement.AllocationId})
	cc.CommitAll()

	require.Equal(t, collateral, f.balance(f.consumerAcc), "consumer must be refunded exactly once")
	require.Equal(t, math.ZeroInt(), f.balance(f.collateralAcc), "pool should be drained to zero, not overdrawn")
	require.Equal(t, uint64(0), f.agreementLoad(), "provider load must be decremented exactly once")

	f.requireSettledOnce(t, agreement)
}

// TestTeardown_AllocationDrivenSweepsEarnedRevenue is the money-conservation
// regression on the allocation-driven close.
//
// TestTeardown_AllocationDrivenSettlesOnce above drives the same entry point but
// tears down in the block it opened, so nothing has been earned yet and the
// checkpoint moves zero. That makes it blind to the failure this test exists
// for: settling without checkpointing first. Checkpoint bills the provider's
// current load across the span since the last checkpoint, so once
// AgreementLoadDecrease has run and the agreement is out of the store, no later
// checkpoint can account for the span this agreement was live. The revenue it
// earned simply stays in the collateral pool, and neither Checkpoint nor
// WithdrawBalanceAndCommit can reach it, because both are computed from load.
//
// The solvency invariant cannot catch it either: stranded revenue leaves the
// pool holding more than it owes, and the invariant only looks for shortfalls.
// So the guard has to be an exact accounting of where every coin went.
//
// This matters most on this path. Consumer close and expiry run in transactions
// or against an index that can be inspected; this one is reached from grid
// brownout and struct destruction inside block hooks, where nothing rolls back.
func TestTeardown_AllocationDrivenSweepsEarnedRevenue(t *testing.T) {
	// A non-zero provider cancellation penalty splits the elapsed span between
	// the consumer and the provider, so the penalty payout and the checkpoint
	// sweep are both non-zero and the test cannot pass by conflating them.
	f := setupTeardownFixture(t, 10, "0.5", "0")

	const capacity, duration, elapsed = 100, 50, 20
	agreement, collateral := f.openAgreement(t, capacity, duration)

	opened := uint64(sdk.UnwrapSDKContext(f.ctx).BlockHeight())
	require.Equal(t, opened, agreement.StartBlock,
		"service and the checkpoint clock must start in the same block for the arithmetic below to hold")
	require.Equal(t, collateral, f.balance(f.collateralAcc))

	// Let the provider genuinely earn part of the term before the allocation is
	// torn out from under the agreement.
	lateCtx := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(int64(opened + elapsed))
	f.ctx = lateCtx

	cc := f.k.NewCurrentContext(lateCtx)
	cc.DestroyMultipleAllocations([]string{agreement.AllocationId})
	cc.CommitAll()

	// The elapsed span splits in two and the rest of the term comes back, so the
	// three payouts must add up to the deposit exactly:
	//
	//	elapsed * rate * capacity * penalty        -> consumer, as compensation
	//	elapsed * rate * capacity * (1 - penalty)  -> provider earnings
	//	remaining * rate * capacity                -> consumer, unearned collateral
	rate := f.provider.Rate.Amount
	elapsedValue := math.NewInt(elapsed).MulRaw(capacity).Mul(rate)
	providerEarned := elapsedValue.QuoRaw(2)            // the (1 - 0.5) share
	consumerPenalty := elapsedValue.Sub(providerEarned) // the 0.5 share
	unearned := math.NewInt(duration - elapsed).MulRaw(capacity).Mul(rate)

	require.Equal(t, providerEarned, f.balance(f.earningsAcc),
		"the revenue earned before the forced closure must be swept, not left behind in the collateral pool")
	require.Equal(t, consumerPenalty.Add(unearned), f.balance(f.consumerAcc),
		"the consumer is owed the cancellation penalty on the served span plus all of the unserved one")

	// The whole point: the pool paid out everything it took in.
	require.Equal(t, math.ZeroInt(), f.balance(f.collateralAcc),
		"the collateral pool must end empty; anything left is stranded with no path to reclaim it")
	require.Equal(t, collateral, providerEarned.Add(consumerPenalty).Add(unearned),
		"the three payouts must account for the deposit exactly")

	require.Equal(t, uint64(0), f.agreementLoad(), "provider load must be released exactly once")
	f.requireSettledOnce(t, agreement)

	solvency, broken := keeperlib.ProviderCollateralSolvencyInvariant(f.k)(lateCtx)
	require.False(t, broken, solvency)
}

// TestTeardown_AllocationDeleteRejectsProviderAgreement pins that the agreement's
// own allocation cannot be deleted out from under it by message, and that the
// rejection settles nothing.
func TestTeardown_AllocationDeleteRejectsProviderAgreement(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 50
	agreement, collateral := f.openAgreement(t, capacity, duration)

	_, err := f.ms.AllocationDelete(f.ctx, &types.MsgAllocationDelete{
		Creator:      f.consumer.Creator,
		AllocationId: agreement.AllocationId,
	})
	require.Error(t, err, "a providerAgreement allocation is not directly deletable")

	_, stillThere := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, stillThere, "a rejected allocation delete must not settle the agreement")
	require.Equal(t, collateral, f.balance(f.collateralAcc), "collateral must be untouched")
	require.Equal(t, math.ZeroInt(), f.balance(f.consumerAcc))
	require.Equal(t, uint64(capacity), f.agreementLoad())
}

// TestTeardown_EmitsExactlyOneDeleteEventPerAgreement guards the public event
// stream. Teardown removes the agreement from the store and marks the cache
// deleted, and both of those can emit EventDelete; GRASS and structs-pg consume
// these, so a duplicate is a downstream bug.
func TestTeardown_EmitsExactlyOneDeleteEventPerAgreement(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 50
	agreement, _ := f.openAgreement(t, capacity, duration)

	// Fresh event manager so only the close's own events are counted.
	uctx := sdk.UnwrapSDKContext(f.ctx).WithEventManager(sdk.NewEventManager())
	f.ctx = uctx

	_, err := f.ms.AgreementClose(f.ctx, &types.MsgAgreementClose{
		Creator:     f.consumer.Creator,
		AgreementId: agreement.Id,
	})
	require.NoError(t, err)

	deleteEventType := proto.MessageName(&types.EventDelete{})

	deletes := 0
	for _, event := range uctx.EventManager().Events() {
		if event.Type != deleteEventType {
			continue
		}
		for _, attribute := range event.Attributes {
			if attribute.Key == "objectId" && strings.Trim(attribute.Value, `"`) == agreement.Id {
				deletes++
			}
		}
	}

	require.Equal(t, 1, deletes, "teardown must emit exactly one EventDelete for the agreement")
}

// TestTeardown_ProviderDeleteSettlesOnce covers PrematureCloseByProvider, reached
// through ProviderDelete.
func TestTeardown_ProviderDeleteSettlesOnce(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 50
	agreement, collateral := f.openAgreement(t, capacity, duration)

	_, err := f.ms.ProviderDelete(f.ctx, &types.MsgProviderDelete{
		Creator:    f.provider.Creator,
		ProviderId: f.provider.Id,
	})
	require.NoError(t, err)

	require.Equal(t, collateral, f.balance(f.consumerAcc), "consumer must be refunded exactly once")
	require.Equal(t, math.ZeroInt(), f.balance(f.collateralAcc))
	require.Equal(t, uint64(0), f.agreementLoad())

	f.requireSettledOnce(t, agreement)
}

// TestTeardown_ExpiryDoesNotPayConsumerCancellationPenalty is the worst case the
// bug produced. On expiry the provider keeps the cancellation penalty via the
// voided payout; the re-entrant settlement paid the consumer that same penalty
// again out of collateral belonging to other agreements.
func TestTeardown_ExpiryDoesNotPayConsumerCancellationPenalty(t *testing.T) {
	// A nonzero provider penalty is what made the double payout visible.
	f := setupTeardownFixture(t, 10, "0.5", "0")

	const capacity, duration = 100, 10
	agreement, collateral := f.openAgreement(t, capacity, duration)

	uctx := sdk.UnwrapSDKContext(f.ctx)
	expiryCtx := uctx.WithBlockHeight(int64(agreement.EndBlock))

	cc := f.k.NewCurrentContext(expiryCtx)
	cc.AgreementExpirations()
	cc.CommitAll()

	f.ctx = expiryCtx

	require.Equal(t, math.ZeroInt(), f.balance(f.consumerAcc),
		"an expired agreement owes the consumer nothing: they received the service they paid for")

	// Everything the consumer deposited is now the provider's, split between the
	// checkpointed revenue and the penalty they kept.
	require.Equal(t, collateral, f.balance(f.earningsAcc), "the full collateral should have become provider earnings")
	require.Equal(t, math.ZeroInt(), f.balance(f.collateralAcc), "the collateral pool should net to zero, not overdraw")
	require.Equal(t, uint64(0), f.agreementLoad(), "provider load must be decremented exactly once")

	f.requireSettledOnce(t, agreement)
}

// TestTeardown_OneAgreementCannotDrainAnother is the property that makes the pool
// safe to share. Tearing one agreement down must leave every other agreement's
// collateral fully intact, which is exactly what the double settlement broke.
func TestTeardown_OneAgreementCannotDrainAnother(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0")

	const capacity, duration = 100, 50
	victim, victimCollateral := f.openAgreement(t, capacity, duration)
	closing, closingCollateral := f.openAgreement(t, capacity, duration)

	require.Equal(t, victimCollateral.Add(closingCollateral), f.balance(f.collateralAcc))

	_, err := f.ms.AgreementClose(f.ctx, &types.MsgAgreementClose{
		Creator:     f.consumer.Creator,
		AgreementId: closing.Id,
	})
	require.NoError(t, err)

	require.Equal(t, victimCollateral, f.balance(f.collateralAcc),
		"the surviving agreement's collateral must be untouched")

	_, stillOpen := f.k.GetAgreement(f.ctx, victim.Id)
	require.True(t, stillOpen, "the other agreement should still be open")
	require.Equal(t, uint64(capacity), f.agreementLoad(), "only the closed agreement's load should be released")

	// And the survivor can still be closed and paid in full afterwards.
	_, err = f.ms.AgreementClose(f.ctx, &types.MsgAgreementClose{
		Creator:     f.consumer.Creator,
		AgreementId: victim.Id,
	})
	require.NoError(t, err)

	require.Equal(t, closingCollateral.Add(victimCollateral), f.balance(f.consumerAcc),
		"both consumers' collateral should have been returned in full")
	require.Equal(t, math.ZeroInt(), f.balance(f.collateralAcc))
}

// TestTeardown_UnusablePayoutAddressFailsLoudly covers the failure that needed no
// adversary: PlayerCache.GetPrimaryAccount drops the bech32 error and returns an
// empty address, and every payout discarded the SendCoins error that followed, so
// the collateral was destroyed along with the agreement.
func TestTeardown_UnusablePayoutAddressFailsLoudly(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 50
	agreement, collateral := f.openAgreement(t, capacity, duration)

	// Break the payout destination out from under the open agreement.
	broken := f.consumer
	broken.PrimaryAddress = "not-a-bech32-address"
	f.k.SetPlayer(f.ctx, broken)

	_, err := f.ms.AgreementClose(f.ctx, &types.MsgAgreementClose{
		Creator:     f.consumer.Creator,
		AgreementId: agreement.Id,
	})
	require.Error(t, err, "settlement must not proceed when the consumer cannot be paid")

	var settlementErr *types.AgreementSettlementError
	require.ErrorAs(t, err, &settlementErr)
	require.Equal(t, uint32(1720), settlementErr.Code())

	_, stillThere := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, stillThere, "the agreement must survive a failed settlement rather than being torn down unpaid")
	require.Equal(t, collateral, f.balance(f.collateralAcc), "the collateral must still be in the pool")
}

// TestTeardown_AllocationCloseDoesNotPayPartialOnShortPool pins the atomicity of
// the consumer payout on the block-hook close path. On a pool too short to cover
// the cancellation penalty AND the returned collateral, the two must settle
// together or not at all: paying the penalty and then aborting on the collateral
// would leave the agreement alive to be re-paid the penalty on a later teardown.
func TestTeardown_AllocationCloseDoesNotPayPartialOnShortPool(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0")

	const capacity, duration, elapsed = 100, 50, 20
	agreement, _ := f.openAgreement(t, capacity, duration)

	opened := uint64(sdk.UnwrapSDKContext(f.ctx).BlockHeight())
	lateCtx := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(int64(opened + elapsed))
	f.ctx = lateCtx

	// Checkpoint now so the close's own checkpoint sweeps nothing further and the
	// pool balance set below is exactly what the payout sees.
	cc0 := f.k.NewCurrentContext(lateCtx)
	require.NoError(t, cc0.GetProvider(f.provider.Id).Checkpoint())
	cc0.CommitAll()

	cc1 := f.k.NewCurrentContext(lateCtx)
	ac := cc1.GetAgreement(agreement.Id)
	require.True(t, ac.LoadAgreement())
	remaining := ac.GetRemainingCollateral()
	require.True(t, remaining.IsPositive())

	// Short the pool to exactly the remaining collateral: enough for that leg
	// alone, but not for the penalty on top of it. (A solvent pool always covers
	// both; this reproduces a pool already drained by some other defect.)
	poolBal := f.balance(f.collateralAcc)
	require.True(t, poolBal.GT(remaining), "test needs the pool above the remaining collateral to short it")
	burn := sdk.AccAddress("burnburnburnburnburnburnburnburn1234")
	require.NoError(t, f.k.BankKeeper().SendCoins(f.ctx, f.collateralAcc, burn,
		sdk.NewCoins(sdk.NewCoin(teardownDenom, poolBal.Sub(remaining)))))
	require.Equal(t, remaining, f.balance(f.collateralAcc))

	consumerBefore := f.balance(f.consumerAcc)

	err := ac.PrematureCloseByAllocation()
	require.Error(t, err, "a pool that cannot cover penalty+collateral must fail the close")

	require.Equal(t, consumerBefore, f.balance(f.consumerAcc),
		"no partial payout: the consumer receives nothing when the whole payout cannot settle")
	require.Equal(t, remaining, f.balance(f.collateralAcc),
		"the collateral pool is untouched by a failed close")
	_, stillThere := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, stillThere, "a failed close must leave the agreement in place")
}

// TestTeardown_ProviderRevenueNeverOverdrawsConsumerCollateral pins the priority
// rule that makes a shared pool safe: provider revenue is clamped to what the
// pool can spare, so a provider can never be paid out of collateral that belongs
// to a consumer.
func TestTeardown_ProviderRevenueNeverOverdrawsConsumerCollateral(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0")

	const capacity, duration = 100, 10
	_, collateral := f.openAgreement(t, capacity, duration)

	// Let the provider accrue well past the agreement's duration without expiring
	// it. Unclamped, the checkpoint would bill the whole span and drain the pool
	// past what the agreement is owed.
	uctx := sdk.UnwrapSDKContext(f.ctx)
	lateCtx := uctx.WithBlockHeight(int64(agreementEndBlock(f, duration)) + 10000)
	f.ctx = lateCtx

	cc := f.k.NewCurrentContext(lateCtx)
	require.NoError(t, cc.GetProvider(f.provider.Id).Checkpoint())
	cc.CommitAll()

	require.True(t, f.balance(f.collateralAcc).GTE(math.ZeroInt()), "the pool must never go negative")
	require.True(t, f.balance(f.earningsAcc).LTE(collateral),
		"the provider cannot earn more than was ever deposited against them")
}

// agreementEndBlock is the height an agreement opened in the current block ends.
// Service starts in the opening block, so the window is the duration itself.
func agreementEndBlock(f *teardownFixture, duration uint64) uint64 {
	uctx := sdk.UnwrapSDKContext(f.ctx)
	return uint64(uctx.BlockHeight()) + duration
}

// TestTeardown_SolvencyInvariantHoldsAcrossLifecycle exercises the registered
// invariant, which is the standing check that no future teardown path can drain
// the pool unnoticed.
func TestTeardown_SolvencyInvariantHoldsAcrossLifecycle(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	invariant := keeperlib.ProviderCollateralSolvencyInvariant(f.k)
	uctx := sdk.UnwrapSDKContext(f.ctx)

	requireSolvent := func(stage string) {
		msg, broken := invariant(uctx)
		require.False(t, broken, "%s: %s", stage, msg)
	}

	requireSolvent("no agreements")

	first, _ := f.openAgreement(t, 100, 50)
	requireSolvent("one agreement open")

	second, _ := f.openAgreement(t, 250, 20)
	requireSolvent("two agreements open")

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

// TestTeardown_RepeatedOpenCloseCannotErodePool composes the three defects that
// were filed and fixed separately: an agreement valued one block above what was
// escrowed, a settlement that ran twice through the reciprocal allocation
// teardown, and payout errors discarded so that neither surfaced. Individually
// each leaks a bounded amount per close. Together the leak was repeatable and
// silent, and because a provider's collateral pool is shared by all of its
// agreements, it was funded by other consumers' escrow.
//
// One cycle is already pinned exactly by TestTeardown_ConsumerCloseSettlesOnce.
// What only repetition catches is erosion that compounds: a leak of one block per
// cycle is small against a single deposit but empties the pool given enough
// cycles. So this runs the cycle repeatedly with a victim agreement standing
// alongside, and after every one requires that the pool holds precisely the
// victim's deposit and that the provider is still solvent.
func TestTeardown_RepeatedOpenCloseCannotErodePool(t *testing.T) {
	// No penalties, so each cycle is owed its whole deposit back and any
	// discrepancy is the leak rather than a fee.
	f := setupTeardownFixture(t, 10, "0", "0")

	invariant := keeperlib.ProviderCollateralSolvencyInvariant(f.k)

	const capacity, duration = 100, 50
	victim, victimCollateral := f.openAgreement(t, capacity, duration)
	require.Equal(t, victimCollateral, f.balance(f.collateralAcc))

	const cycles = 5
	var collateral math.Int

	for cycle := 1; cycle <= cycles; cycle++ {
		var agreement types.Agreement
		agreement, collateral = f.openAgreement(t, capacity, duration)

		_, err := f.ms.AgreementClose(f.ctx, &types.MsgAgreementClose{
			Creator:     f.consumer.Creator,
			AgreementId: agreement.Id,
		})
		require.NoError(t, err, "cycle %d", cycle)

		// openAgreement funds each deposit, so the refunds accumulate: after N
		// round trips the consumer is up exactly N deposits and no more.
		require.Equal(t, collateral.MulRaw(int64(cycle)), f.balance(f.consumerAcc),
			"cycle %d: consumer must recover exactly the deposit, never more", cycle)
		require.Equal(t, victimCollateral, f.balance(f.collateralAcc),
			"cycle %d: the pool must come back to the victim's deposit exactly", cycle)
		require.Equal(t, uint64(capacity), f.agreementLoad(),
			"cycle %d: only the victim's load should remain", cycle)

		msg, broken := invariant(sdk.UnwrapSDKContext(f.ctx))
		require.False(t, broken, "cycle %d: %s", cycle, msg)
	}

	// The victim never participated, and is still owed and paid in full.
	_, err := f.ms.AgreementClose(f.ctx, &types.MsgAgreementClose{
		Creator:     f.consumer.Creator,
		AgreementId: victim.Id,
	})
	require.NoError(t, err)

	require.Equal(t, collateral.MulRaw(cycles).Add(victimCollateral), f.balance(f.consumerAcc))
	require.Equal(t, math.ZeroInt(), f.balance(f.collateralAcc),
		"the pool should end empty, having paid out exactly what it took in")
}

// TestTeardown_ExpiryCannotStrandCollateral is the regression on the report that
// an expiry drains a provider's whole pool into an unreachable account.
//
// The mechanism was an off-by-one between two clocks: an agreement opened at
// height H checkpointed the provider at H but started service at H+1, so expiry
// at H+1+D billed D+1 blocks against a pool that had only ever been paid for D.
// A zero provider cancellation penalty is what made it bite, because then the
// checkpoint is the only thing standing between the deposit and the pool.
//
// The exact-balance assertions after a solo expiry cannot catch this on their
// own: the revenue clamp added alongside the fix caps an over-large sweep at
// whatever the pool holds, so a solo agreement's pool still lands on zero either
// way. What the extra block actually costs is somebody else's escrow, so this
// keeps a second agreement open throughout and pins what is left against it. The
// clamp turns the loss into a shortfall event rather than a failed transfer, so
// that is asserted too, and both invariants have to hold at the end.
func TestTeardown_ExpiryCannotStrandCollateral(t *testing.T) {
	// The report's own parameters: no provider cancellation penalty, so every
	// coin of the deposit has to move through the checkpoint and nowhere else.
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 10
	const victimCapacity, victimDuration = 40, 500

	victim, victimCollateral := f.openAgreement(t, victimCapacity, victimDuration)
	agreement, collateral := f.openAgreement(t, capacity, duration)

	// Both clocks start together. This is the property the whole settlement rests
	// on, and reverting it is what the rest of the test detects.
	opened := uint64(sdk.UnwrapSDKContext(f.ctx).BlockHeight())
	require.Equal(t, opened, agreement.StartBlock,
		"service must start in the opening block, the same block the provider was checkpointed in")
	require.Equal(t, opened+duration, agreement.EndBlock)

	require.Equal(t, victimCollateral.Add(collateral), f.balance(f.collateralAcc))

	// Fresh event manager so only the expiry's own events are counted.
	expiryCtx := sdk.UnwrapSDKContext(f.ctx).
		WithBlockHeight(int64(agreement.EndBlock)).
		WithEventManager(sdk.NewEventManager())

	cc := f.k.NewCurrentContext(expiryCtx)
	cc.AgreementExpirations()
	cc.CommitAll()

	f.ctx = expiryCtx

	shortfallType := proto.MessageName(&types.EventProviderRevenueShortfall{})
	for _, event := range expiryCtx.EventManager().Events() {
		require.NotEqual(t, shortfallType, event.Type,
			"expiry asked the pool for more than it held, which means it billed for service nobody bought")
	}

	f.requireSettledOnce(t, agreement)
	require.Equal(t, math.ZeroInt(), f.balance(f.consumerAcc),
		"a fully served agreement owes the consumer nothing")

	// The provider earns the expiring agreement's whole deposit, plus the service
	// the still-open agreement genuinely received over the same span.
	victimEarned := math.NewInt(duration).
		MulRaw(victimCapacity).
		Mul(f.provider.Rate.Amount)
	require.Equal(t, collateral.Add(victimEarned), f.balance(f.earningsAcc))

	// And what is left is precisely the surviving agreement's unearned collateral.
	// Under the off-by-one this is short by one block of both agreements' load.
	require.Equal(t, victimCollateral.Sub(victimEarned), f.balance(f.collateralAcc),
		"the expiry must not have touched the surviving agreement's escrow")
	require.Equal(t, uint64(victimCapacity), f.agreementLoad(),
		"only the expired agreement's load should have been released")

	solvency, broken := keeperlib.ProviderCollateralSolvencyInvariant(f.k)(expiryCtx)
	require.False(t, broken, solvency)

	liveness, broken := keeperlib.AgreementExpiryLivenessInvariant(f.k)(expiryCtx)
	require.False(t, broken, liveness)

	// The survivor is still whole and can be paid out in full.
	_, err := f.ms.AgreementClose(f.ctx, &types.MsgAgreementClose{
		Creator:     f.consumer.Creator,
		AgreementId: victim.Id,
	})
	require.NoError(t, err)

	require.Equal(t, victimCollateral.Sub(victimEarned), f.balance(f.consumerAcc),
		"the survivor must recover every block of service it did not receive")
	require.Equal(t, math.ZeroInt(), f.balance(f.collateralAcc),
		"the pool should end empty, having paid out exactly what it took in")
}

// TestTeardown_ExpiryCompletesWhenTheAllocationIsAlreadyGone pins that an expiry
// finishes even when the allocation it meant to tear down has already left the
// store.
//
// Expiry runs from the EndBlocker, which reads the expiration index at exactly
// the current height: there is no range scan and no retry, so an agreement that
// is not torn down on the one block it comes up is never revisited. Returning
// early therefore does not defer the teardown, it cancels it — the agreement
// keeps its capacity in the provider's load and Checkpoint goes on billing that
// capacity against the shared pool every block afterwards, out of other
// consumers' collateral.
//
// A missing allocation is nothing to tear down, and destroyAllocation already
// said so on the path where CurrentContext has never heard of it. This is the
// other path to the same state, where the allocation is cached from earlier in
// the operation and only Destroy's re-read notices it has gone.
func TestTeardown_ExpiryCompletesWhenTheAllocationIsAlreadyGone(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 10
	agreement, _ := f.openAgreement(t, capacity, duration)

	expiryCtx := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(int64(agreement.EndBlock))
	cc := f.k.NewCurrentContext(expiryCtx)

	// Cache the allocation, then delete the row behind it. GetAllocation answers
	// from the cache without re-reading, so the absence surfaces only inside
	// Destroy — which is the case that used to abort the whole expiry.
	_, found := cc.GetAllocation(agreement.AllocationId)
	require.True(t, found)
	f.k.RemoveAllocation(expiryCtx, agreement.AllocationId)

	require.NoError(t, cc.GetAgreement(agreement.Id).Expire(),
		"an allocation that is already gone is nothing to tear down, not a reason to abandon the expiry")
	cc.CommitAll()

	f.ctx = expiryCtx

	f.requireSettledOnce(t, agreement)
	require.Equal(t, uint64(0), f.agreementLoad(),
		"the provider must stop billing for an agreement that has ended")

	liveness, broken := keeperlib.AgreementExpiryLivenessInvariant(f.k)(expiryCtx)
	require.False(t, broken, liveness)
}

// TestTeardown_ProviderDeleteDrainsBothPools pins that deleting a provider takes
// its pools with it.
//
// Both pool addresses are derived from the provider id, and nothing but the
// provider record can reach them: once it is gone, WithdrawBalanceAndCommit can
// no longer load the provider, and Checkpoint bills against a load that is now
// zero. Anything left behind at that moment is unreachable for good. There is
// normally something, because every checkpoint and every penalty payout truncates
// its share down and leaves the remainder in the collateral pool.
func TestTeardown_ProviderDeleteDrainsBothPools(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 50
	agreement, collateral := f.openAgreement(t, capacity, duration)

	// Stand in for the accumulated truncation remainder directly, so the amount
	// is exact rather than a by-product of the rate and penalties chosen here.
	const collateralDust, earningsDust = 7, 3
	f.fund(t, f.collateralAcc, collateralDust)
	f.fund(t, f.earningsAcc, earningsDust)

	ownerBefore := f.balance(f.ownerAcc)

	_, err := f.ms.ProviderDelete(f.ctx, &types.MsgProviderDelete{
		Creator:    f.provider.Creator,
		ProviderId: f.provider.Id,
	})
	require.NoError(t, err)

	// The consumer is made whole out of the pool before any of it is swept, which
	// is the order that makes the sweep safe at all.
	require.Equal(t, collateral, f.balance(f.consumerAcc), "consumer must be refunded in full first")

	require.Equal(t, math.ZeroInt(), f.balance(f.collateralAcc), "the collateral pool must not outlive the provider")
	require.Equal(t, math.ZeroInt(), f.balance(f.earningsAcc), "the earnings pool must not outlive the provider")
	require.Equal(t, ownerBefore.AddRaw(collateralDust+earningsDust), f.balance(f.ownerAcc),
		"what neither the consumers nor the pools are owed belongs to the provider's owner")

	f.requireSettledOnce(t, agreement)
}

// TestExpiryLivenessInvariant_DetectsAnAgreementPastItsEnd covers the standing
// check on the one thing the solvency invariant structurally cannot see.
//
// Solvency clamps every span to the agreement's own window, so an agreement left
// behind past its end block reads as one that has simply been fully served, and
// it only breaks once the pool is already visibly short — by which point the
// draining has happened. Liveness catches the state itself, before the money
// moves.
func TestExpiryLivenessInvariant_DetectsAnAgreementPastItsEnd(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	invariant := keeperlib.AgreementExpiryLivenessInvariant(f.k)
	agreement, _ := f.openAgreement(t, 100, 50)

	atHeight := func(height uint64) (string, bool) {
		return invariant(sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(int64(height)))
	}

	msg, broken := atHeight(agreement.StartBlock)
	require.False(t, broken, "an agreement that has just opened is not overdue: %s", msg)

	msg, broken = atHeight(agreement.EndBlock - 1)
	require.False(t, broken, "an agreement still inside its window is not overdue: %s", msg)

	// The EndBlocker expires an agreement during its end block, and the crisis
	// module may run invariants either side of that, so the end block itself has
	// to be slack rather than a break.
	msg, broken = atHeight(agreement.EndBlock)
	require.False(t, broken, "an agreement has its whole end block to be expired in: %s", msg)

	msg, broken = atHeight(agreement.EndBlock + 1)
	require.True(t, broken, "an agreement still in state after its end block is stranded")
	require.Contains(t, msg, agreement.Id)

	// Expiring it clears the break, which is what says the invariant is measuring
	// the stranding rather than the passage of time.
	expiryCtx := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(int64(agreement.EndBlock))
	cc := f.k.NewCurrentContext(expiryCtx)
	cc.AgreementExpirations()
	cc.CommitAll()

	msg, broken = atHeight(agreement.EndBlock + 1)
	require.False(t, broken, "a settled agreement leaves nothing to flag: %s", msg)
}

// Collateral is only ever collected for EndBlock - StartBlock, so the whole of
// the accounting rests on past + remaining == duration holding at every height.
// This walks a window that has not started yet, which is the one shape that used
// to break the identity: settlement measured from a height before StartBlock
// priced more remaining duration than the agreement was long and refunded blocks
// nobody deposited, out of a pool shared with the provider's other agreements.
//
// Nothing reaches that state today — AgreementOpen starts service in the opening
// block and the capacity changes re-base to the current one — so the clamp in
// LoadDurationRemaining has no reachable trigger and, untested, would be
// indistinguishable from dead code. This is what says it is load-bearing if a
// future start block is ever reintroduced.
func TestAgreementDuration_IdentityHoldsBeforeServiceStarts(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration, lead = 100, 50, 7
	agreement, collateral := f.openAgreement(t, capacity, duration)

	// Move the window into the future, the shape a scheduled agreement would have.
	opened := uint64(sdk.UnwrapSDKContext(f.ctx).BlockHeight())
	stored, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, found)
	stored.StartBlock = opened + lead
	stored.EndBlock = stored.StartBlock + duration
	_, err := f.k.SetAgreement(f.ctx, stored)
	require.NoError(t, err)

	for _, height := range []uint64{
		opened,                 // well before the start block
		stored.StartBlock - 1,  // the block before service begins
		stored.StartBlock,      // the first funded block
		stored.StartBlock + 1,  // one block served
		stored.StartBlock + 25, // mid window
		stored.EndBlock,        // fully served
		stored.EndBlock + 5,    // settled late
	} {
		uctx := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(int64(height))
		cache := f.k.NewCurrentContext(uctx).GetAgreement(agreement.Id)
		require.True(t, cache.LoadAgreement())

		past := cache.GetDurationPast()
		remaining := cache.GetDurationRemaining()

		require.Equal(t, uint64(duration), cache.GetDuration(),
			"height %d: the funded duration is a property of the window, not the current height", height)
		require.Equal(t, uint64(duration), past+remaining,
			"height %d: past (%d) + remaining (%d) must equal the funded duration", height, past, remaining)

		// The identity in money terms: a payout can never exceed the deposit.
		require.True(t, cache.GetRemainingCollateral().LTE(collateral),
			"height %d: priced %s remaining against a %s deposit",
			height, cache.GetRemainingCollateral(), collateral)
	}
}
