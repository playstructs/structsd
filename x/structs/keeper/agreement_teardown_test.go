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
