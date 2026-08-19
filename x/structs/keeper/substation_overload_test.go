package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Regression tests for the availability gate that underflowed during a brownout.
//
// A substation's capacity is the power allocated into it and its load is the
// power allocated out of it, so load > capacity is a real, reachable state: any
// allocation feeding a substation can shrink or disappear mid-block, and the
// only thing that forces load back under capacity is GridCascade, which runs in
// the EndBlocker. Until then — and for longer, since GridCascade gives up and
// logs "Grid Queue problem" when there are not enough allocations to shed — the
// substation sits over-subscribed while messages keep being processed against
// it.
//
// SubstationCache.GetAvailableCapacity computed capacity - load in uint64, so in
// exactly that window it did not report "nothing available", it wrapped to
// almost 2^64. Both gates built on it therefore let power be sold out of a
// substation that had none: opening a new agreement, and enlarging an existing
// one. The bound that remained was the provider's own published capacity
// maximum, so this is not the unlimited overload it first looks like, but it is
// still new load added to a grid that is already shedding, and every unit of it
// is paid for by GridCascade destroying somebody else's allocation.
//
// PlayerCache.GetAvailableCapacity already clamped, and SetInitialPower and
// SetDynamicPower already refused before subtracting. The substation getter was
// the one that did not.

// overload drives the substation into brownout by dropping its capacity below
// its current load, the state a vanishing inbound allocation produces, and
// returns the load it left in place.
func overload(t *testing.T, f *teardownFixture) uint64 {
	t.Helper()

	loadId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, f.substationId)
	capacityId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, f.substationId)

	load := f.k.GetGridAttribute(f.ctx, loadId)
	require.NotZero(t, load, "the substation must be carrying load for an overload to mean anything")

	// One unit short is the smallest real brownout and the worst case for a
	// subtraction that wraps: it produces the largest possible bogus headroom.
	f.k.SetGridAttribute(f.ctx, capacityId, load-1)

	return load
}

// substationAvailableCapacity reads the gate's own input.
func substationAvailableCapacity(f *teardownFixture) uint64 {
	return f.k.NewCurrentContext(f.ctx).GetSubstation(f.substationId).GetAvailableCapacity()
}

// TestBrownout_AvailableCapacityClampsInsteadOfWrapping is the direct unit on
// the getter both gates depend on.
func TestBrownout_AvailableCapacityClampsInsteadOfWrapping(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 50
	f.openAgreement(t, capacity, duration)

	load := overload(t, f)

	available := substationAvailableCapacity(f)
	require.Equal(t, uint64(0), available,
		"an over-subscribed substation has no capacity to sell; capacity - load underflowed to %d", available)
	require.Less(t, available, load,
		"available capacity can never exceed the load already on the substation")
}

// TestBrownout_CannotEnlargeAgreementDuringBrownout is the exploit. Unlike the
// open path below, nothing downstream catches this one: CapacityIncrease resizes
// through AllocationCache.SetPower, which — unlike SetInitialPower and
// SetDynamicPower — does no load-versus-capacity check of its own and leaves its
// caller to have gated it. The availability gate was that gate.
//
// AgreementCapacityIncrease needs no privileged role: AgreementOpen grants the
// consumer PermAgreementAll, so the ordinary counterparty holds PermUpdate.
//
// The duration here is deliberately long. An increase re-prices the unearned
// span as remaining*old/new, so enlarging a short agreement drives the rescaled
// duration under the provider's published minimum and is refused for that
// reason instead — which is an unrelated gate, and a test that tripped it would
// pass no matter what the availability check did. Verified: at duration 50 this
// same call is rejected with "duration (0) cannot be lower than minimum", while
// at 5000 it went through and took capacity from 100 to 10000 on a substation
// with nothing left to give.
func TestBrownout_CannotEnlargeAgreementDuringBrownout(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 5000
	agreement, _ := f.openAgreement(t, capacity, duration)

	overload(t, f)

	providerLoadBefore := f.agreementLoad()

	// Everything up to the provider's published maximum, which is the only bound
	// left once the availability gate is vacuous.
	_, err := f.ms.AgreementCapacityIncrease(f.ctx, &types.MsgAgreementCapacityIncrease{
		Creator:          f.consumer.Creator,
		AgreementId:      agreement.Id,
		CapacityIncrease: f.provider.CapacityMaximum - capacity,
	})
	require.Error(t, err,
		"a substation already shedding load must not be able to sell more of it")

	after, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, found)
	require.Equal(t, uint64(capacity), after.Capacity, "the rejected increase must not have been written")
	require.Equal(t, providerLoadBefore, f.agreementLoad(), "the rejected increase must not have raised provider load")
}

// TestBrownout_CannotOpenAgreementDuringBrownout covers the second gate built on
// the same getter, ProviderCache.AgreementVerify, which admits a brand new
// agreement. The report did not name this one.
//
// It is not a hole today, and this test passed before the clamp existed: the
// underflowed gate waves the open through, but creating the agreement's
// allocation calls SetInitialPower, which refuses on its own correct comparison
// ("allocation source does not have capacity (0) for power (100)"). So what is
// pinned here is the layered defence, not a fix. It is worth keeping precisely
// because the outer gate being wrong was invisible while the inner one held —
// and CapacityIncrease above is what that looks like when it does not.
func TestBrownout_CannotOpenAgreementDuringBrownout(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 50
	f.openAgreement(t, capacity, duration)

	overload(t, f)

	before := f.k.GetAllAgreementIdByProviderIndex(f.ctx, f.provider.Id)

	f.fund(t, f.consumerAcc, 1_000_000)
	_, err := f.ms.AgreementOpen(f.ctx, &types.MsgAgreementOpen{
		Creator:    f.consumer.Creator,
		ProviderId: f.provider.Id,
		Capacity:   capacity,
		Duration:   duration,
	})
	require.Error(t, err,
		"a substation with no headroom must not admit a new agreement")

	require.Len(t, f.k.GetAllAgreementIdByProviderIndex(f.ctx, f.provider.Id), len(before),
		"the rejected open must not have created an agreement")
}

// TestBrownout_HeadroomStillSellableWhenHealthy is the other half of the fix:
// the clamp must only bite while over-subscribed. A substation with genuine
// headroom has to keep selling it, or the guard has quietly become an outage.
func TestBrownout_HeadroomStillSellableWhenHealthy(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 50
	agreement, _ := f.openAgreement(t, capacity, duration)

	loadId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, f.substationId)
	capacityId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, f.substationId)

	load := f.k.GetGridAttribute(f.ctx, loadId)
	const headroom = 500
	f.k.SetGridAttribute(f.ctx, capacityId, load+headroom)

	require.Equal(t, uint64(headroom), substationAvailableCapacity(f),
		"a healthy substation must report its real headroom")

	_, err := f.ms.AgreementCapacityIncrease(f.ctx, &types.MsgAgreementCapacityIncrease{
		Creator:          f.consumer.Creator,
		AgreementId:      agreement.Id,
		CapacityIncrease: headroom,
	})
	require.NoError(t, err, "an increase that fits inside real headroom must still be allowed")

	after, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, found)
	require.Equal(t, uint64(capacity+headroom), after.Capacity)
}

// TestBrownout_PlayerAccessorsClampWhenOverloaded pins the player-side siblings.
//
// GetAvailableCapacity and CanSupportLoadAddition already clamped; the fix
// brought GetAllocatableCapacity into line with them. It has no callers today,
// which is exactly why it is worth a test: the next caller to reach for it would
// have inherited the substation bug, and nothing else would have said so.
func TestBrownout_PlayerAccessorsClampWhenOverloaded(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	playerId := f.consumer.Id
	capacityId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, playerId)
	loadId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, playerId)

	f.k.SetGridAttribute(f.ctx, capacityId, 100)
	f.k.SetGridAttribute(f.ctx, loadId, 500)

	cc := f.k.NewCurrentContext(f.ctx)
	player := cc.GetPlayer(playerId)

	// Without these the test would pass vacuously: if the accessors read
	// different attributes than the ones set above, both would be zero, zero
	// clamps to zero, and the assertions below would hold for no reason.
	require.Equal(t, uint64(100), player.GetCapacity(), "test setup must actually reach the accessor")
	require.Equal(t, uint64(500), player.GetLoad(), "test setup must actually reach the accessor")

	require.Equal(t, uint64(0), player.GetAllocatableCapacity(),
		"an over-subscribed player has nothing to allocate; the subtraction must not wrap")
	require.Equal(t, uint64(0), player.GetAvailableCapacity())
	require.False(t, player.CanSupportLoadAddition(1),
		"an over-subscribed player cannot take on more load")
}

// TestBrownout_ExactlyFullSubstationSellsNothing pins the boundary between the
// two tests above, where an off-by-one in the guard would hide.
func TestBrownout_ExactlyFullSubstationSellsNothing(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	const capacity, duration = 100, 50
	f.openAgreement(t, capacity, duration)

	loadId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, f.substationId)
	capacityId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, f.substationId)

	load := f.k.GetGridAttribute(f.ctx, loadId)
	f.k.SetGridAttribute(f.ctx, capacityId, load)

	require.Equal(t, uint64(0), substationAvailableCapacity(f),
		"a substation whose load exactly equals its capacity has nothing left to sell")
}
