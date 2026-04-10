package ante_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sante "structs/app/ante"
	"structs/x/structs/types"
)

func TestGasRouterDecorator_FreeTx(t *testing.T) {
	dec := sante.NewGasRouterDecorator(10_000_000)
	next, called := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1test", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	newCtx, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
	require.True(t, sante.IsFreeTx(newCtx))
}

func TestGasRouterDecorator_PaidTx(t *testing.T) {
	dec := sante.NewGasRouterDecorator(10_000_000)
	next, called := identityHandler()

	// Use a real Structs UpdateParams (routed to paid) as a proxy for non-free
	tx := mockTx{msgs: []sdk.Msg{&types.MsgUpdateParams{}}}

	newCtx, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
	require.False(t, sante.IsFreeTx(newCtx))
}

func TestGasRouterDecorator_UpdateParamsNotFree(t *testing.T) {
	dec := sante.NewGasRouterDecorator(10_000_000)
	next, called := identityHandler()

	msg := &types.MsgUpdateParams{}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	newCtx, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
	require.False(t, sante.IsFreeTx(newCtx))
}
