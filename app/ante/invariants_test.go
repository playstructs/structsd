package ante_test

import (
	"fmt"
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
	mk := freshKeeper()
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
	return err
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
