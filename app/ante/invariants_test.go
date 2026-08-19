package ante_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	sante "structs/app/ante"
	"structs/x/structs/types"
)

// Invariant under test (incident 2026-05):
//
//	For any free Structs transaction T and any chain state S, the
//	ThrottleDecorator must return the same admission outcome (success or
//	the same typed error) when run in CheckTx, ReCheckTx, DeliverTx, and
//	simulate phases against identical fresh state S.
//
// Violating this invariant is what caused the testnet stuck-tx incident: the
// decorator skipped CheckTx/ReCheckTx so the mempool admitted txs that the
// chain then refused at DeliverTx with no way to evict.
//
// The property is checked over randomly generated single- and multi-message
// txs drawn from the same shapes the chain actually sees: fleet moves, planet
// explores, address registers, proof messages, and charge messages.

// genCreator picks one of a small pool of test addresses.
func genCreator(t *rapid.T) string {
	return rapid.SampledFrom([]string{"structs1alice", "structs1bob", "structs1carol", "structs1dave"}).Draw(t, "creator")
}

// genObjectID generates an object id string of the form "<typeNum>-<idx>".
func genObjectID(t *rapid.T, typeNum int) string {
	idx := rapid.IntRange(1, 50).Draw(t, "idx")
	return fmt.Sprintf("%d-%d", typeNum, idx)
}

// genStructID returns a struct id in the small id space we use for tests.
func genStructID(t *rapid.T) string {
	return genObjectID(t, int(types.ObjectType_struct))
}

// genFleetID returns a fleet id in the small id space we use for tests.
func genFleetID(t *rapid.T) string {
	return genObjectID(t, int(types.ObjectType_fleet))
}

// genPlayerID returns a player id in the small id space we use for tests.
func genPlayerID(t *rapid.T) string {
	return genObjectID(t, int(types.ObjectType_player))
}

// genMsg draws one Structs message at random across all categories the
// ThrottleDecorator cares about (charge / proof / throttle-key).
func genMsg(t *rapid.T) sdk.Msg {
	kind := rapid.IntRange(0, 6).Draw(t, "kind")
	creator := genCreator(t)
	switch kind {
	case 0:
		return &types.MsgFleetMove{Creator: creator, FleetId: genFleetID(t), DestinationLocationId: "7-1"}
	case 1:
		return &types.MsgPlanetExplore{Creator: creator, PlayerId: genPlayerID(t)}
	case 2:
		return &types.MsgAddressRegister{Creator: creator, PlayerId: genPlayerID(t), Address: "structs1zzz", ProofPubKey: "aa", ProofSignature: "bb"}
	case 3:
		return &types.MsgStructBuildComplete{Creator: creator, StructId: genStructID(t), Proof: "abc", Nonce: "1"}
	case 4:
		return &types.MsgStructDefenseSet{Creator: creator, DefenderStructId: genStructID(t), ProtectedStructId: genStructID(t)}
	case 5:
		return &types.MsgStructMove{Creator: creator, StructId: genStructID(t)}
	default:
		return &types.MsgStructAttack{Creator: creator, OperatingStructId: genStructID(t)}
	}
}

// freshKeeper builds a mock keeper preloaded with player indexes for each
// of the canonical test creators, so the charge throttle has a path to
// playerId resolution.
func freshKeeper() *mockAnteKeeper {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 5
	mk.playerIndexes["structs1bob"] = 6
	mk.playerIndexes["structs1carol"] = 7
	mk.playerIndexes["structs1dave"] = 8
	return mk
}

// runThrottleOnce builds a fresh keeper and a context for the requested
// phase, runs the throttle decorator on tx, and returns the resulting error.
func runThrottleOnce(
	tx sdk.Tx,
	phase string,
) error {
	_, err := runThrottleWithAuth(tx, phase, nil)
	return err
}

// runThrottleWithAuth is runThrottleOnce with a target-authorization policy and
// the reserved key set returned alongside the outcome, so a property can check
// what the decorator wrote and not only what it answered.
func runThrottleWithAuth(
	tx sdk.Tx,
	phase string,
	deny func(creator, targetId string) bool,
) (map[string]bool, error) {
	mk := freshKeeper()
	mk.throttleAuthDenyFn = deny
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	ctx := deliverCtx()
	simulate := false
	switch phase {
	case "checkTx":
		ctx = ctx.WithIsCheckTx(true)
	case "reCheckTx":
		ctx = ctx.WithIsReCheckTx(true)
	case "simulate":
		simulate = true
	case "deliverTx":
	}

	_, err := dec.AnteHandle(ctx, tx, simulate, next)
	return mk.throttleKeys, err
}

// throttleSentinels enumerates every typed error the ThrottleDecorator is
// allowed to return. The invariant requires the SAME sentinel to fire in
// every phase, not merely "some error vs no error".
var throttleSentinels = []*errorsmod.Error{
	sante.ErrDuplicateChargeInTx,
	sante.ErrChargeAlreadyUsedThisBlock,
	sante.ErrDuplicateProofInTx,
	sante.ErrProofAlreadyAttemptedThisBlock,
	sante.ErrDuplicateThrottleKeyInTx,
	sante.ErrObjectThrottledThisBlock,
	sante.ErrMissingCreator,
}

// sameAdmissionOutcome returns true if both errors are nil OR both wrap the
// SAME typed sentinel from the structs-ante codespace.
func sameAdmissionOutcome(t *testing.T, a, b error) bool {
	t.Helper()
	if (a == nil) != (b == nil) {
		t.Logf("admission disagreement (nil mismatch): a=%v b=%v", a, b)
		return false
	}
	if a == nil {
		return true
	}
	for _, s := range throttleSentinels {
		aIs := s.Is(a)
		bIs := s.Is(b)
		if aIs && bIs {
			return true
		}
		if aIs != bIs {
			t.Logf("admission disagreement on %s: a=%v b=%v", s.Error(), a, b)
			return false
		}
	}
	t.Logf("admission both errored but no recognized sentinel matched: a=%v b=%v", a, b)
	return false
}

// TestInvariant_ThrottleSameOutcomeAcrossPhases is the central property test:
// the same tx against fresh state must produce the same admission outcome in
// every phase. Run with -rapid.checks for more thorough exploration.
func TestInvariant_ThrottleSameOutcomeAcrossPhases(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		msgCount := rapid.IntRange(1, 4).Draw(rt, "msgCount")
		msgs := make([]sdk.Msg, msgCount)
		for i := 0; i < msgCount; i++ {
			msgs[i] = genMsg(rt)
		}
		tx := mockTx{msgs: msgs}

		deliverErr := runThrottleOnce(tx, "deliverTx")
		checkErr := runThrottleOnce(tx, "checkTx")
		recheckErr := runThrottleOnce(tx, "reCheckTx")
		simErr := runThrottleOnce(tx, "simulate")

		require.True(rt, sameAdmissionOutcome(t, deliverErr, checkErr),
			"CheckTx must match DeliverTx outcome")
		require.True(rt, sameAdmissionOutcome(t, deliverErr, recheckErr),
			"ReCheckTx must match DeliverTx outcome")
		require.True(rt, sameAdmissionOutcome(t, deliverErr, simErr),
			"simulate must match DeliverTx outcome")
	})
}

// denyOddTargets refuses authorization over half the generated id space. Which
// half does not matter; what matters is that the answer depends only on the
// message, so every phase is asking the same question.
func denyOddTargets(_ string, targetId string) bool {
	if targetId == "" {
		return false
	}
	last := targetId[len(targetId)-1]
	return (last-'0')%2 == 1
}

// TestInvariant_ThrottleReservationsSameAcrossPhases extends the admission
// invariant to the writes. Gating the reservation on target authorization added
// a second observable output, and it has to agree across phases for the same
// reason the first one does: a node that reserves in CheckTx but not in
// DeliverTx refuses at admission what consensus would have accepted, which is
// how incident 2026-05 stranded transactions in the mempool.
func TestInvariant_ThrottleReservationsSameAcrossPhases(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		msgCount := rapid.IntRange(1, 4).Draw(rt, "msgCount")
		msgs := make([]sdk.Msg, msgCount)
		for i := 0; i < msgCount; i++ {
			msgs[i] = genMsg(rt)
		}
		tx := mockTx{msgs: msgs}

		deliverKeys, deliverErr := runThrottleWithAuth(tx, "deliverTx", denyOddTargets)

		for _, phase := range []string{"checkTx", "reCheckTx", "simulate"} {
			phaseKeys, phaseErr := runThrottleWithAuth(tx, phase, denyOddTargets)

			require.True(rt, sameAdmissionOutcome(t, deliverErr, phaseErr),
				"%s must match DeliverTx outcome", phase)
			require.Equal(rt, deliverKeys, phaseKeys,
				"%s reserved a different key set than DeliverTx", phase)
		}
	})
}

// An unauthorized signer reserves nothing, whatever else the transaction is
// doing. The per-player charge key is exempt: it is derived from the signer's
// own player index rather than anything the transaction names, so there is no
// other player's key to poison.
func TestInvariant_UnauthorizedTargetsAreNeverReserved(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		msgCount := rapid.IntRange(1, 4).Draw(rt, "msgCount")
		msgs := make([]sdk.Msg, msgCount)
		for i := 0; i < msgCount; i++ {
			msgs[i] = genMsg(rt)
		}

		denyAll := func(string, string) bool { return true }
		keys, _ := runThrottleWithAuth(mockTx{msgs: msgs}, "deliverTx", denyAll)

		for key := range keys {
			require.True(rt, strings.HasPrefix(key, "charge/"),
				"reserved %s for a target the signer is not authorized over", key)
		}
	})
}
