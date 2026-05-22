package ante_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	sante "structs/app/ante"
	"structs/x/structs/types"
)

func deliverCtx() sdk.Context {
	return newTestCtx().WithValue(sante.FreeGasCtxKey(), true)
}

func checkCtx() sdk.Context {
	return deliverCtx().WithIsCheckTx(true)
}

func recheckCtx() sdk.Context {
	return deliverCtx().WithIsReCheckTx(true)
}

// errIs returns true if err is (or wraps) the given sentinel via the SDK
// errors package.
func errIs(err error, target *errorsmod.Error) bool {
	if err == nil || target == nil {
		return false
	}
	if target.Is(err) {
		return true
	}
	return false
}

func TestThrottleDecorator_FleetMoveFirstPass(t *testing.T) {
	mk := newMockAnteKeeper()
	dec := sante.NewThrottleDecorator(mk)
	next, called := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
	require.True(t, mk.throttleKeys["fleet/2-1"])
}

func TestThrottleDecorator_FleetMoveSameFleetRejected(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.throttleKeys["fleet/2-1"] = true
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-2"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.Error(t, err)
	require.True(t, errIs(err, sante.ErrObjectThrottledThisBlock))
}

func TestThrottleDecorator_DifferentFleetsPass(t *testing.T) {
	mk := newMockAnteKeeper()
	dec := sante.NewThrottleDecorator(mk)
	next, called := identityHandler()

	msgs := []sdk.Msg{
		&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"},
		&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-2", DestinationLocationId: "7-1"},
	}
	tx := mockTx{msgs: msgs}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestThrottleDecorator_SameFleetInOneTxRejected(t *testing.T) {
	mk := newMockAnteKeeper()
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	msgs := []sdk.Msg{
		&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"},
		&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-2"},
	}
	tx := mockTx{msgs: msgs}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.Error(t, err)
	require.True(t, errIs(err, sante.ErrDuplicateThrottleKeyInTx))
}

func TestThrottleDecorator_ProofThrottle(t *testing.T) {
	mk := newMockAnteKeeper()
	dec := sante.NewThrottleDecorator(mk)
	next, called := identityHandler()

	msg := &types.MsgStructBuildComplete{Creator: "structs1alice", StructId: "5-1", Proof: "abc", Nonce: "123"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
	require.True(t, mk.throttleKeys["proof/5-1"])
}

func TestThrottleDecorator_ProofSameStructRejected(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.throttleKeys["proof/5-1"] = true
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	msg := &types.MsgStructBuildComplete{Creator: "structs1alice", StructId: "5-1", Proof: "abc", Nonce: "123"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.Error(t, err)
	require.True(t, errIs(err, sante.ErrProofAlreadyAttemptedThisBlock))
}

func TestThrottleDecorator_SameProofInOneTxRejected(t *testing.T) {
	mk := newMockAnteKeeper()
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	msgs := []sdk.Msg{
		&types.MsgStructBuildComplete{Creator: "structs1alice", StructId: "5-1", Proof: "abc", Nonce: "1"},
		&types.MsgStructBuildComplete{Creator: "structs1alice", StructId: "5-1", Proof: "def", Nonce: "2"},
	}
	tx := mockTx{msgs: msgs}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.Error(t, err)
	require.True(t, errIs(err, sante.ErrDuplicateProofInTx))
}

// TestThrottleDecorator_RunsInCheckTx asserts the post-incident behavior:
// the throttle MUST enforce charge / proof / throttle keys at CheckTx so the
// mempool can never admit a tx that will later fail DeliverTx. The old code
// returned a no-op early, which is what caused the 2026-05 stuck-tx incident.
func TestThrottleDecorator_RunsInCheckTx(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 5
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	msgs := []sdk.Msg{
		&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1657"},
		&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1658"},
	}
	tx := mockTx{msgs: msgs}

	_, err := dec.AnteHandle(checkCtx(), tx, false, next)
	require.Error(t, err, "duplicate-charge tx MUST fail CheckTx after incident 2026-05")
	require.True(t, errIs(err, sante.ErrDuplicateChargeInTx))
}

// TestThrottleDecorator_RunsInReCheckTx asserts the same enforcement during
// ReCheckTx. Before the fix, ReCheckTx skipped, so already-admitted bad txs
// were never evicted.
func TestThrottleDecorator_RunsInReCheckTx(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 5
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	msgs := []sdk.Msg{
		&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1657"},
		&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1658"},
	}
	tx := mockTx{msgs: msgs}

	_, err := dec.AnteHandle(recheckCtx(), tx, false, next)
	require.Error(t, err, "duplicate-charge tx MUST fail ReCheckTx so the mempool can evict it")
	require.True(t, errIs(err, sante.ErrDuplicateChargeInTx))
}

// TestThrottleDecorator_RunsInSimulate keeps simulate honest too: an
// /eth_estimateGas / `simulate` call that bundles a bug-shape tx should return
// the same error the chain would have returned on-chain, so wallet UX can
// catch it before the user signs.
func TestThrottleDecorator_RunsInSimulate(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 5
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	msgs := []sdk.Msg{
		&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1657"},
		&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1658"},
	}
	tx := mockTx{msgs: msgs}

	_, err := dec.AnteHandle(deliverCtx(), tx, true /* simulate */, next)
	require.Error(t, err, "simulate MUST return the same reject so wallets surface it pre-sign")
	require.True(t, errIs(err, sante.ErrDuplicateChargeInTx))
}

func TestThrottleDecorator_PlanetExplore(t *testing.T) {
	mk := newMockAnteKeeper()
	dec := sante.NewThrottleDecorator(mk)
	next, called := identityHandler()

	msg := &types.MsgPlanetExplore{Creator: "structs1alice", PlayerId: "1-5"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
	require.True(t, mk.throttleKeys["explore/1-5"])
}

func TestThrottleDecorator_AddressRegister(t *testing.T) {
	mk := newMockAnteKeeper()
	dec := sante.NewThrottleDecorator(mk)
	next, called := identityHandler()

	msg := &types.MsgAddressRegister{Creator: "structs1alice", PlayerId: "1-5", Address: "structs1bob", ProofPubKey: "aabb", ProofSignature: "ccdd"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
	require.True(t, mk.throttleKeys["register/1-5"])
}

// ---------- Charge throttle (the actual incident bug shape) ----------

// TestThrottleDecorator_ChargeFirstPass exercises the happy path: one
// charge-consuming message from a registered player passes and reserves the
// charge slot.
func TestThrottleDecorator_ChargeFirstPass(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 5 // player 1-5
	dec := sante.NewThrottleDecorator(mk)
	next, called := identityHandler()

	msg := &types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1657"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
	require.True(t, mk.throttleKeys[fmt.Sprintf("charge/%d-5", types.ObjectType_player)])
}

// TestThrottleDecorator_DuplicateChargeInOneTxRejected is the regression test
// for incident 2026-05: two MsgStructDefenseSet messages from the same player
// in one tx must be rejected, in every phase, with the typed error.
func TestThrottleDecorator_DuplicateChargeInOneTxRejected(t *testing.T) {
	cases := []struct {
		name string
		ctx  sdk.Context
	}{
		{"DeliverTx", deliverCtx()},
		{"CheckTx", checkCtx()},
		{"ReCheckTx", recheckCtx()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mk := newMockAnteKeeper()
			mk.playerIndexes["structs1alice"] = 5
			dec := sante.NewThrottleDecorator(mk)
			next, _ := identityHandler()

			msgs := []sdk.Msg{
				&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1657"},
				&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1703", ProtectedStructId: "5-1658"},
			}
			tx := mockTx{msgs: msgs}

			_, err := dec.AnteHandle(tc.ctx, tx, false, next)
			require.Error(t, err, "phase %s", tc.name)
			require.True(t, errIs(err, sante.ErrDuplicateChargeInTx), "phase %s: got %v", tc.name, err)
		})
	}
}

// TestThrottleDecorator_MixedChargeMessagesInOneTxRejected: a player can't
// dodge the per-tx dedup by mixing different charge message types.
func TestThrottleDecorator_MixedChargeMessagesInOneTxRejected(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 5
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	msgs := []sdk.Msg{
		&types.MsgStructMove{Creator: "structs1alice", StructId: "5-1"},
		&types.MsgStructAttack{Creator: "structs1alice", OperatingStructId: "5-2"},
	}
	tx := mockTx{msgs: msgs}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.Error(t, err)
	require.True(t, errIs(err, sante.ErrDuplicateChargeInTx))
}

// TestThrottleDecorator_DifferentPlayersInOneTxPass: a tx with charge messages
// from two distinct players is permitted (e.g. proxied actions). The dedup key
// is per-player, not per-tx.
func TestThrottleDecorator_DifferentPlayersInOneTxPass(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 5
	mk.playerIndexes["structs1bob"] = 6
	dec := sante.NewThrottleDecorator(mk)
	next, called := identityHandler()

	msgs := []sdk.Msg{
		&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1657"},
		&types.MsgStructDefenseSet{Creator: "structs1bob", DefenderStructId: "5-1703", ProtectedStructId: "5-1658"},
	}
	tx := mockTx{msgs: msgs}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

// TestThrottleDecorator_CrossTxChargeRejected: even with single-msg txs, the
// transient store rejects the second charge from the same player in the same
// block. (The fallback layer.)
func TestThrottleDecorator_CrossTxChargeRejected(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 5
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	tx1 := mockTx{msgs: []sdk.Msg{&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1657"}}}
	tx2 := mockTx{msgs: []sdk.Msg{&types.MsgStructAttack{Creator: "structs1alice", OperatingStructId: "5-2"}}}

	_, err := dec.AnteHandle(deliverCtx(), tx1, false, next)
	require.NoError(t, err)

	_, err = dec.AnteHandle(deliverCtx(), tx2, false, next)
	require.Error(t, err)
	require.True(t, errIs(err, sante.ErrChargeAlreadyUsedThisBlock))
}

// TestThrottleDecorator_NoTransientStoreSkipsCrossTx: when the keeper reports
// no transient store available (genesis init, some test harnesses), the
// throttle silently degrades to per-tx-only enforcement. Per-tx dedup is
// stateless and must still work.
func TestThrottleDecorator_NoTransientStoreStillCatchesPerTxDuplicates(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.hasTransientStore = false
	mk.playerIndexes["structs1alice"] = 5
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	msgs := []sdk.Msg{
		&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1657"},
		&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1703", ProtectedStructId: "5-1658"},
	}
	tx := mockTx{msgs: msgs}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.Error(t, err)
	require.True(t, errIs(err, sante.ErrDuplicateChargeInTx))
}
