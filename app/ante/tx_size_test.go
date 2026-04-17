package ante_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/app/ante"
	"structs/x/structs/types"
)

func TestTxSizeDecorator_FreeTxUnderLimit(t *testing.T) {
	dec := ante.NewTxSizeDecorator(1024)
	next, called := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	ctx := newTestCtx().WithTxBytes(make([]byte, 512))
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestTxSizeDecorator_FreeTxOverLimit(t *testing.T) {
	dec := ante.NewTxSizeDecorator(256)
	next, _ := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	ctx := newTestCtx().WithTxBytes(make([]byte, 512))
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds cap")
}

func TestTxSizeDecorator_FreeTxAtExactLimit(t *testing.T) {
	dec := ante.NewTxSizeDecorator(256)
	next, called := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	ctx := newTestCtx().WithTxBytes(make([]byte, 256))
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestTxSizeDecorator_NonFreeTxBypasses(t *testing.T) {
	dec := ante.NewTxSizeDecorator(64)
	next, called := identityHandler()

	tx := mockTx{msgs: []sdk.Msg{&types.MsgUpdateParams{}}}

	ctx := newTestCtx().WithTxBytes(make([]byte, 8192))
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestTxSizeDecorator_DefaultCap(t *testing.T) {
	dec := ante.NewTxSizeDecorator(0)
	next, called := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	ctx := newTestCtx().WithTxBytes(make([]byte, 1024))
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}
