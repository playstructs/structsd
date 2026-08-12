package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// TestReactorInfusionDelegationRemovedClearsFuel is the unit-level regression
// test for the phantom infusion a full redelegation used to leave behind.
//
// Note what is deliberately not done here: the delegation is left in the mock
// staking keeper. That is faithful, not sloppy. Staking's RemoveDelegation fires
// BeforeDelegationRemoved before it deletes the row, so the delegation really is
// still readable, at its pre-decrement shares, while this runs.
func TestReactorInfusionDelegationRemovedClearsFuel(t *testing.T) {
	f := setupJailEnergy(t, "delremclears", 1000, "0.04")

	require.Equal(t, uint64(1000), f.infusion(t).Fuel)
	require.Equal(t, uint64(1000), f.infusion(t).Power)
	require.Equal(t, uint64(960), f.playerCapacity(), "the delegator holds the 96% share")
	require.Equal(t, uint64(40), f.reactorCapacity(), "the reactor holds its 4% commission")
	require.Equal(t, uint64(1000), f.reactorFuel())

	f.k.ReactorInfusionDelegationRemoved(f.ctx, f.playerAcc, f.valAddr)

	cleared := f.infusion(t)
	require.Equal(t, uint64(0), cleared.Fuel, "the removed delegation must leave no fuel")
	require.Equal(t, uint64(0), cleared.Power)
	require.Equal(t, uint64(0), f.playerCapacity(), "the delegator's capacity goes with the stake")
	require.Equal(t, uint64(0), f.reactorCapacity(), "so does the commission the reactor no longer earns")
	require.Equal(t, uint64(0), f.reactorFuel())

	// Nothing of value remains, so the record is released to the destruction
	// queue rather than deleted inline.
	_, err := f.k.EndBlocker(f.ctx)
	require.NoError(t, err)

	_, found := f.k.GetInfusion(f.ctx, f.reactor.Id, f.playerAcc.String())
	require.False(t, found, "an emptied infusion should be reclaimed by the destruction queue")
}

// TestReconcileCannotClearFuelWhileDelegationReadable pins the reason the hook
// zeroes fuel explicitly instead of delegating to the reconciler, which is the
// obvious-looking implementation and is wrong.
//
// Reconciliation reads the delegation and writes whatever it finds. In the
// window BeforeDelegationRemoved runs in, what it finds is the row staking is
// about to delete, so it rewrites the exact value it was called to clear.
func TestReconcileCannotClearFuelWhileDelegationReadable(t *testing.T) {
	f := setupJailEnergy(t, "delremreconcile", 1000, "0.04")

	f.k.ReconcileInfusionForDelegation(f.ctx, f.playerAcc, f.valAddr)

	require.Equal(t, uint64(1000), f.infusion(t).Fuel,
		"reconciling in this window restores the stale fuel, which is why the hook does not")

	f.k.ReactorInfusionDelegationRemoved(f.ctx, f.playerAcc, f.valAddr)
	require.Equal(t, uint64(0), f.infusion(t).Fuel)
}

// TestReactorInfusionDelegationRemovedPreservesDefusing covers the case where a
// delegator unbonds part of their stake and then moves the rest. The unbonding
// balance outlives the delegation record and is still owed to them, so the hook
// must clear the fuel without touching it.
func TestReactorInfusionDelegationRemovedPreservesDefusing(t *testing.T) {
	f := setupJailEnergy(t, "delremdefusing", 1000, "0.04")

	pending := f.infusion(t)
	pending.Defusing = 500
	f.k.SetInfusion(f.ctx, pending)

	f.k.ReactorInfusionDelegationRemoved(f.ctx, f.playerAcc, f.valAddr)

	after := f.infusion(t)
	require.Equal(t, uint64(0), after.Fuel)
	require.Equal(t, uint64(0), after.Power)
	require.Equal(t, uint64(500), after.Defusing, "the unbonding balance is not the hook's to clear")

	_, err := f.k.EndBlocker(f.ctx)
	require.NoError(t, err)

	survived, found := f.k.GetInfusion(f.ctx, f.reactor.Id, f.playerAcc.String())
	require.True(t, found, "a record still owing a defusing balance must survive")
	require.Equal(t, uint64(500), survived.Defusing)
}

// TestReactorInfusionDelegationRemovedIsIdempotent guards the re-run. A second
// call must not write, because SetInfusion emits an EventInfusion on every write
// and those events are a public API.
func TestReactorInfusionDelegationRemovedIsIdempotent(t *testing.T) {
	f := setupJailEnergy(t, "delremidempotent", 1000, "0.04")

	f.k.ReactorInfusionDelegationRemoved(f.ctx, f.playerAcc, f.valAddr)
	first := f.infusion(t)
	eventsAfterFirst := len(f.ctx.EventManager().Events())

	f.k.ReactorInfusionDelegationRemoved(f.ctx, f.playerAcc, f.valAddr)

	require.Equal(t, first, f.infusion(t), "a repeat call must change nothing")
	require.Equal(t, eventsAfterFirst, len(f.ctx.EventManager().Events()),
		"a repeat call must emit no further events")
}

// TestReactorInfusionDelegationRemovedNoOpWithoutReactor covers a validator that
// has no reactor. Staking fires delegation hooks for every validator on the
// chain, including any that predate the structs module, so this has to be quiet
// rather than fatal.
func TestReactorInfusionDelegationRemovedNoOpWithoutReactor(t *testing.T) {
	f := setupJailEnergy(t, "delremnoreactor", 1000, "0.04")

	strangerVal := sdk.ValAddress(fmt.Sprintf("%-36s", "delremstrangerval")[:36])

	require.NotPanics(t, func() {
		f.k.ReactorInfusionDelegationRemoved(f.ctx, f.playerAcc, strangerVal)
	})

	require.Equal(t, uint64(1000), f.infusion(t).Fuel, "an unrelated validator must not touch this reactor")
}

// TestReactorInfusionDelegationRemovedNoOpWithoutInfusion covers a delegator who
// never infused. The hook must not upsert a row on its way past, or every
// delegation removal on the chain leaves an empty record behind.
func TestReactorInfusionDelegationRemovedNoOpWithoutInfusion(t *testing.T) {
	f := setupJailEnergy(t, "delremnoinfusion", 1000, "0.04")

	stranger := sdk.AccAddress(fmt.Sprintf("%-36s", "delremstrangeracc")[:36])

	f.k.ReactorInfusionDelegationRemoved(f.ctx, stranger, f.valAddr)

	_, found := f.k.GetInfusion(f.ctx, f.reactor.Id, stranger.String())
	require.False(t, found, "the hook must not create the record it was asked to clear")

	require.Equal(t, uint64(1000), f.infusion(t).Fuel, "the real delegator is untouched")
}

// TestReactorInfusionDelegationRemovedGatedReactor covers a jailed validator,
// where the infusion is already at a zero ratio and carries no power but still
// holds the delegator's stake. The stake still has to go, since the delegation
// behind it is being removed.
func TestReactorInfusionDelegationRemovedGatedReactor(t *testing.T) {
	f := setupJailEnergy(t, "delremgated", 1000, "0.04")

	f.mock.JailValidator(f.valAddr)
	f.k.ReactorGateEnergy(f.ctx, f.valAddr)

	gated := f.infusion(t)
	require.Equal(t, uint64(0), gated.Ratio)
	require.Equal(t, uint64(1000), gated.Fuel, "gating withdraws energy, not stake")

	f.k.ReactorInfusionDelegationRemoved(f.ctx, f.playerAcc, f.valAddr)

	after := f.infusion(t)
	require.Equal(t, uint64(0), after.Fuel, "a removed delegation takes its fuel with it")
	require.Equal(t, uint64(0), after.Ratio)
	require.Equal(t, uint64(0), f.reactorFuel())
}
