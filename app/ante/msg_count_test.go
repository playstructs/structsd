package ante_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/app/ante"
)

func TestMsgCountDecorator_UnderLimit(t *testing.T) {
	dec := ante.NewMsgCountDecorator(5)
	next, called := identityHandler()

	msgs := make([]sdk.Msg, 3)
	for i := range msgs {
		msgs[i] = &mockMsg{}
	}
	tx := mockTx{msgs: msgs}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestMsgCountDecorator_AtLimit(t *testing.T) {
	dec := ante.NewMsgCountDecorator(5)
	next, called := identityHandler()

	msgs := make([]sdk.Msg, 5)
	for i := range msgs {
		msgs[i] = &mockMsg{}
	}
	tx := mockTx{msgs: msgs}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestMsgCountDecorator_OverLimit(t *testing.T) {
	dec := ante.NewMsgCountDecorator(5)
	next, _ := identityHandler()

	msgs := make([]sdk.Msg, 6)
	for i := range msgs {
		msgs[i] = &mockMsg{}
	}
	tx := mockTx{msgs: msgs}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cap is 5")
}

func TestMsgCountDecorator_DefaultCap(t *testing.T) {
	dec := ante.NewMsgCountDecorator(0)
	next, called := identityHandler()

	msgs := make([]sdk.Msg, 40)
	for i := range msgs {
		msgs[i] = &mockMsg{}
	}
	tx := mockTx{msgs: msgs}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}
