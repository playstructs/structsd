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

// throttleTargetCases enumerates every message whose throttle key names an
// object the transaction chose, which is exactly the set an attacker can aim at
// somebody else's property.
type throttleTargetCase struct {
	name     string
	targetId string
	wantKey  string
	newMsg   func(creator, targetId string) sdk.Msg
}

func throttleTargetCases() []throttleTargetCase {
	return []throttleTargetCase{
		{"fleet move", "9-77", "fleet/9-77", func(creator, target string) sdk.Msg {
			return &types.MsgFleetMove{Creator: creator, FleetId: target, DestinationLocationId: "2-1"}
		}},
		{"planet explore", "1-77", "explore/1-77", func(creator, target string) sdk.Msg {
			return &types.MsgPlanetExplore{Creator: creator, PlayerId: target}
		}},
		{"address register", "1-77", "register/1-77", func(creator, target string) sdk.Msg {
			return &types.MsgAddressRegister{Creator: creator, PlayerId: target, Address: "structs1zzz", ProofPubKey: "aa", ProofSignature: "bb", Permissions: uint64(types.PermPlay)}
		}},
		{"struct build complete", "5-77", "proof/5-77", func(creator, target string) sdk.Msg {
			return &types.MsgStructBuildComplete{Creator: creator, StructId: target, Proof: "abc", Nonce: "1"}
		}},
		{"ore miner complete", "5-77", "proof/5-77", func(creator, target string) sdk.Msg {
			return &types.MsgStructOreMinerComplete{Creator: creator, StructId: target, Proof: "abc", Nonce: "1"}
		}},
		{"ore refinery complete", "5-77", "proof/5-77", func(creator, target string) sdk.Msg {
			return &types.MsgStructOreRefineryComplete{Creator: creator, StructId: target, Proof: "abc", Nonce: "1"}
		}},
		{"planet raid complete", "9-77", "proof/9-77", func(creator, target string) sdk.Msg {
			return &types.MsgPlanetRaidComplete{Creator: creator, FleetId: target, Proof: "abc", Nonce: "1"}
		}},
	}
}

// TestThrottleDecorator_UnauthorizedTargetReservesNothing is the regression for
// the key-poisoning attack. The ante's address-level check passes — the attacker
// is an ordinary player whose primary address holds PermAll — and the SDK
// commits ante writes even when the message later fails, so a reservation made
// here would outlive the handler's rejection and censor the victim for the rest
// of the block.
//
// The message is deliberately still admitted. The handler is where the
// unauthorized action gets its error; the throttle's only job is to not hand out
// the victim's slot.
func TestThrottleDecorator_UnauthorizedTargetReservesNothing(t *testing.T) {
	for _, tc := range throttleTargetCases() {
		t.Run(tc.name, func(t *testing.T) {
			mk := newMockAnteKeeper()
			mk.playerIndexes["structs1attacker"] = 9
			mk.denyThrottleTarget("structs1attacker", tc.targetId)
			dec := sante.NewThrottleDecorator(mk)
			next, called := identityHandler()

			tx := mockTx{msgs: []sdk.Msg{tc.newMsg("structs1attacker", tc.targetId)}}

			_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
			require.NoError(t, err, "the throttle must not reject; the handler owns that error")
			require.True(t, *called)
			require.False(t, mk.throttleKeys[tc.wantKey], "reserved %s for an object the signer has no standing on", tc.wantKey)
			require.Empty(t, mk.throttleKeys)
		})
	}
}

// The control: an authorized signer still reserves, so the fix did not simply
// turn the throttle off.
func TestThrottleDecorator_AuthorizedTargetStillReserves(t *testing.T) {
	for _, tc := range throttleTargetCases() {
		t.Run(tc.name, func(t *testing.T) {
			mk := newMockAnteKeeper()
			mk.playerIndexes["structs1owner"] = 9
			dec := sante.NewThrottleDecorator(mk)
			next, _ := identityHandler()

			tx := mockTx{msgs: []sdk.Msg{tc.newMsg("structs1owner", tc.targetId)}}

			_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
			require.NoError(t, err)
			require.True(t, mk.throttleKeys[tc.wantKey])
		})
	}
}

// The point of the whole change: after the attacker's transaction has been and
// gone, the victim's own transaction for the same object still goes through.
func TestThrottleDecorator_VictimNotThrottledByUnauthorizedAttempt(t *testing.T) {
	for _, tc := range throttleTargetCases() {
		t.Run(tc.name, func(t *testing.T) {
			mk := newMockAnteKeeper()
			mk.playerIndexes["structs1attacker"] = 9
			mk.playerIndexes["structs1victim"] = 10
			mk.denyThrottleTarget("structs1attacker", tc.targetId)
			dec := sante.NewThrottleDecorator(mk)
			next, _ := identityHandler()

			attack := mockTx{msgs: []sdk.Msg{tc.newMsg("structs1attacker", tc.targetId)}}
			_, err := dec.AnteHandle(deliverCtx(), attack, false, next)
			require.NoError(t, err)

			victim := mockTx{msgs: []sdk.Msg{tc.newMsg("structs1victim", tc.targetId)}}
			_, err = dec.AnteHandle(deliverCtx(), victim, false, next)
			require.NoError(t, err, "the victim was locked out of its own object for the block")
			require.True(t, mk.throttleKeys[tc.wantKey])
		})
	}
}

// The reported worst case: one free transaction carrying the maximum number of
// messages, each naming a different victim object.
func TestThrottleDecorator_FullTxOfVictimTargetsReservesNothing(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1attacker"] = 9
	dec := sante.NewThrottleDecorator(mk)
	next, called := identityHandler()

	msgs := make([]sdk.Msg, 0, sante.DefaultMaxMsgCount)
	for i := 0; i < sante.DefaultMaxMsgCount; i++ {
		fleetId := fmt.Sprintf("9-%d", i)
		mk.denyThrottleTarget("structs1attacker", fleetId)
		msgs = append(msgs, &types.MsgFleetMove{Creator: "structs1attacker", FleetId: fleetId, DestinationLocationId: "2-1"})
	}

	_, err := dec.AnteHandle(deliverCtx(), mockTx{msgs: msgs}, false, next)
	require.NoError(t, err)
	require.True(t, *called)
	require.Empty(t, mk.throttleKeys, "one transaction parked %d victim objects", len(mk.throttleKeys))
}

// A key already claimed this block still rejects, whatever the signer's standing
// on the object. Skipping the reservation must not also skip the read: an
// unauthorized message is doomed either way, and letting it through here would
// split the outcome from the one CheckTx computed.
func TestThrottleDecorator_UnauthorizedStillRejectedOnClaimedKey(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1attacker"] = 9
	mk.throttleKeys["fleet/9-77"] = true
	mk.denyThrottleTarget("structs1attacker", "9-77")
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	tx := mockTx{msgs: []sdk.Msg{&types.MsgFleetMove{Creator: "structs1attacker", FleetId: "9-77", DestinationLocationId: "2-1"}}}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.Error(t, err)
	require.True(t, errIs(err, sante.ErrObjectThrottledThisBlock))
}

// Per-tx dedup is stateless and answers before authorization is consulted, so
// two messages naming the same object are still refused even when neither of
// them could have reserved anything.
func TestThrottleDecorator_UnauthorizedDuplicateInTxStillRejected(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1attacker"] = 9
	mk.denyThrottleTarget("structs1attacker", "5-77")
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	msgs := []sdk.Msg{
		&types.MsgStructBuildComplete{Creator: "structs1attacker", StructId: "5-77", Proof: "abc", Nonce: "1"},
		&types.MsgStructOreMinerComplete{Creator: "structs1attacker", StructId: "5-77", Proof: "abc", Nonce: "2"},
	}

	_, err := dec.AnteHandle(deliverCtx(), mockTx{msgs: msgs}, false, next)
	require.Error(t, err)
	require.True(t, errIs(err, sante.ErrDuplicateProofInTx))
}

// Authorization is only consulted to decide a write, so a keeper with no
// transient store must not start caring about it.
func TestThrottleDecorator_NoTransientStoreIgnoresAuthorization(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.hasTransientStore = false
	mk.playerIndexes["structs1attacker"] = 9
	mk.denyThrottleTarget("structs1attacker", "9-77")
	dec := sante.NewThrottleDecorator(mk)
	next, called := identityHandler()

	tx := mockTx{msgs: []sdk.Msg{&types.MsgFleetMove{Creator: "structs1attacker", FleetId: "9-77", DestinationLocationId: "2-1"}}}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
	require.Empty(t, mk.throttleKeys)
}

// The refusal has to hold in the mempool too. A node that reserved in CheckTx
// would reject the victim's transaction at admission instead of at consensus,
// which is the same censorship one layer up.
func TestThrottleDecorator_UnauthorizedReservesNothingInEveryPhase(t *testing.T) {
	newAttack := func() (*mockAnteKeeper, sdk.Tx) {
		mk := newMockAnteKeeper()
		mk.playerIndexes["structs1attacker"] = 9
		mk.denyThrottleTarget("structs1attacker", "5-77")
		return mk, mockTx{msgs: []sdk.Msg{&types.MsgStructBuildComplete{Creator: "structs1attacker", StructId: "5-77", Proof: "abc", Nonce: "1"}}}
	}

	phases := map[string]struct {
		ctx      sdk.Context
		simulate bool
	}{
		"deliverTx": {deliverCtx(), false},
		"checkTx":   {checkCtx(), false},
		"reCheckTx": {recheckCtx(), false},
		"simulate":  {deliverCtx(), true},
	}

	for name, phase := range phases {
		t.Run(name, func(t *testing.T) {
			mk, tx := newAttack()
			next, _ := identityHandler()

			_, err := sante.NewThrottleDecorator(mk).AnteHandle(phase.ctx, tx, phase.simulate, next)
			require.NoError(t, err)
			require.Empty(t, mk.throttleKeys)
		})
	}
}
