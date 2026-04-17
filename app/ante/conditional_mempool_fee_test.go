package ante_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sante "structs/app/ante"
	"structs/x/structs/types"
)

func TestConditionalMempoolFeeDecorator_FreeTxSkipsMempoolFeeCheck(t *testing.T) {
	dec := sante.NewConditionalMempoolFeeDecorator()
	next, called := identityHandler()

	tx := mockTx{msgs: []sdk.Msg{
		&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"},
	}}

	ctx := newTestCtx().
		WithIsCheckTx(true).
		WithValue(sante.FreeGasCtxKey(), true)

	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestConditionalMempoolFeeDecorator_PaidTxRunsMempoolFeeCheck(t *testing.T) {
	dec := sante.NewConditionalMempoolFeeDecorator()
	next, called := identityHandler()

	// mockTx does not implement FeeTx, so invoking the SDK mempool fee decorator
	// should fail if the paid path correctly reaches the inner decorator.
	tx := mockTx{msgs: []sdk.Msg{&types.MsgUpdateParams{}}}
	ctx := newTestCtx().WithIsCheckTx(true)

	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "FeeTx")
	require.False(t, *called)
}
