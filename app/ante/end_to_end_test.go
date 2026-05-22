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

// End-to-end tests that exercise every custom Structs ante decorator in the
// exact order they appear in app/ante/ante.go::NewAnteHandler, around an
// identity stand-in for the SDK decorators (which are tested separately by
// the SDK). The point is to catch regressions where a bug flows through more
// than one of our decorators, like incident 2026-05.

// productionChainStructs builds the custom-decorator chain in production
// order for free Structs txs. SDK decorators (SetUpContext, signature
// verification, sequence increment, etc.) are stubbed because they require
// real keepers and are covered by the SDK's own test suite.
func productionChainStructs(mk *mockAnteKeeper) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		sante.NewTxSizeDecorator(0),
		sante.NewMsgCountDecorator(0),
		sante.NewGasRouterDecorator(0, 0),
		sante.NewConditionalMempoolFeeDecorator(),
		sante.NewCheckTxThrottleDecorator(5),
		sante.NewPubKeyDerivationDecorator(),
		sante.NewStructsDecorator(mk, 40),
		sante.NewThrottleDecorator(mk),
		sante.NewStakingThrottleDecorator(mk),
	)
}

// e2eErrIs is a small wrapper to test typed-error matches.
func e2eErrIs(err error, target *errorsmod.Error) bool {
	return err != nil && target != nil && target.Is(err)
}

// TestE2E_BugShape_DuplicateDefenseSetRejectedInAllPhases is the canonical
// regression test for incident 2026-05. We feed the bug-shape tx through
// the full custom chain in every phase and assert it always rejects with
// the typed error, so client SDKs and observability dashboards have a
// stable signal to switch on.
func TestE2E_BugShape_DuplicateDefenseSetRejectedInAllPhases(t *testing.T) {
	cases := []struct {
		name     string
		ctxMod   func(sdk.Context) sdk.Context
		simulate bool
	}{
		{"CheckTx", func(c sdk.Context) sdk.Context { return c.WithIsCheckTx(true) }, false},
		{"ReCheckTx", func(c sdk.Context) sdk.Context { return c.WithIsReCheckTx(true) }, false},
		{"DeliverTx", func(c sdk.Context) sdk.Context { return c }, false},
		{"Simulate", func(c sdk.Context) sdk.Context { return c }, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mk := newMockAnteKeeper()
			mk.playerIndexes["structs1alice"] = 5
			addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
			mk.permissions[addrPermId] = types.PermPlay

			handler := productionChainStructs(mk)
			ctx := tc.ctxMod(freeCtx().WithBlockHeight(100))

			msgs := []sdk.Msg{
				&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1657"},
				&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1703", ProtectedStructId: "5-1658"},
			}
			tx := mockTx{msgs: msgs}

			_, err := handler(ctx, tx, tc.simulate)
			require.Error(t, err, "bug-shape tx MUST be rejected in phase %s", tc.name)
			require.True(t, e2eErrIs(err, sante.ErrDuplicateChargeInTx),
				"phase %s: expected ErrDuplicateChargeInTx, got %v", tc.name, err)
		})
	}
}

// TestE2E_SingleDefenseSetPassesAllPhases is the corresponding happy-path
// regression: the same chain must NOT over-reject a valid single-charge tx.
func TestE2E_SingleDefenseSetPassesAllPhases(t *testing.T) {
	cases := []struct {
		name     string
		ctxMod   func(sdk.Context) sdk.Context
		simulate bool
	}{
		{"CheckTx", func(c sdk.Context) sdk.Context { return c.WithIsCheckTx(true) }, false},
		{"ReCheckTx", func(c sdk.Context) sdk.Context { return c.WithIsReCheckTx(true) }, false},
		{"DeliverTx", func(c sdk.Context) sdk.Context { return c }, false},
		{"Simulate", func(c sdk.Context) sdk.Context { return c }, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mk := newMockAnteKeeper()
			mk.playerIndexes["structs1alice"] = 5
			addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
			mk.permissions[addrPermId] = types.PermPlay

			handler := productionChainStructs(mk)
			ctx := tc.ctxMod(freeCtx().WithBlockHeight(100))

			msg := &types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1657"}
			tx := mockTx{msgs: []sdk.Msg{msg}}

			_, err := handler(ctx, tx, tc.simulate)
			require.NoError(t, err, "valid single-charge tx must pass in phase %s", tc.name)
		})
	}
}

// TestE2E_SequenceStuffingRegression simulates the downstream stuffed tx
// observed during incident 2026-05: a single MsgPlanetExplore at seq 11
// behind a stuck two-charge tx at seq 10. After the fix, the seq 10 tx is
// rejected at CheckTx and never enters the mempool, so the seq 11 tx is
// admitted as the new head of the account's pending stream.
func TestE2E_SequenceStuffingRegression(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 5
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	handler := productionChainStructs(mk)
	ctx := freeCtx().WithBlockHeight(688743).WithIsCheckTx(true)

	bugShape := mockTx{msgs: []sdk.Msg{
		&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1657"},
		&types.MsgStructDefenseSet{Creator: "structs1alice", DefenderStructId: "5-1702", ProtectedStructId: "5-1658"},
	}}
	_, err := handler(ctx, bugShape, false)
	require.Error(t, err)
	require.True(t, e2eErrIs(err, sante.ErrDuplicateChargeInTx))

	mk.throttleKeys = map[string]bool{}

	followup := mockTx{msgs: []sdk.Msg{
		&types.MsgPlanetExplore{Creator: "structs1alice", PlayerId: "1-229"},
	}}
	_, err = handler(ctx, followup, false)
	require.NoError(t, err, "single MsgPlanetExplore must be admitted once the bug-shape tx is rejected upstream")
}
