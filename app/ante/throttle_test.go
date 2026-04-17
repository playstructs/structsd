package ante_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sante "structs/app/ante"
	"structs/x/structs/types"
)

func deliverCtx() sdk.Context {
	return newTestCtx().WithValue(sante.FreeGasCtxKey(), true)
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
	mk.throttleKeys["fleet/2-1"] = true // already moved this block
	dec := sante.NewThrottleDecorator(mk)
	next, _ := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-2"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "throttled this block")
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
	require.Contains(t, err.Error(), "throttled this block")
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
	require.Contains(t, err.Error(), "proof already attempted")
}

func TestThrottleDecorator_SkipsDuringCheckTx(t *testing.T) {
	mk := newMockAnteKeeper()
	dec := sante.NewThrottleDecorator(mk)
	next, called := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	ctx := newTestCtx().WithValue(sante.FreeGasCtxKey(), true).WithIsCheckTx(true)

	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
	require.False(t, mk.throttleKeys["fleet/2-1"])
}

func TestThrottleDecorator_SkipsDuringSimulate(t *testing.T) {
	mk := newMockAnteKeeper()
	dec := sante.NewThrottleDecorator(mk)
	next, called := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(deliverCtx(), tx, true, next) // simulate=true
	require.NoError(t, err)
	require.True(t, *called)
	require.False(t, mk.throttleKeys["fleet/2-1"])
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
