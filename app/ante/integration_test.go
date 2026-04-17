package ante_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sante "structs/app/ante"
	"structs/x/structs/types"
)

func structsSubChain(mk *mockAnteKeeper) sdk.AnteHandler {
	decorators := []sdk.AnteDecorator{
		sante.NewCheckTxThrottleDecorator(5),
		sante.NewStructsDecorator(mk, 40),
		sante.NewThrottleDecorator(mk),
	}
	return sdk.ChainAnteDecorators(decorators...)
}

func TestIntegration_ValidFreeTxPassesSubChain(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	handler := structsSubChain(mk)

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	// DeliverTx context with free flag
	ctx := freeCtx().WithBlockHeight(100)
	_, err := handler(ctx, tx, false)
	require.NoError(t, err)

	// Verify throttle key was set
	require.True(t, mk.throttleKeys["fleet/2-1"])
}

func TestIntegration_ThrottleRejectsSecondFleetMove(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	handler := structsSubChain(mk)

	msg1 := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	msg2 := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-2"}

	ctx := freeCtx().WithBlockHeight(100)

	_, err := handler(ctx, mockTx{msgs: []sdk.Msg{msg1}}, false)
	require.NoError(t, err)

	_, err = handler(ctx, mockTx{msgs: []sdk.Msg{msg2}}, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "throttled this block")
}

func TestIntegration_ChargeThrottleBlocksSecondChargeAction(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	handler := structsSubChain(mk)
	ctx := freeCtx().WithBlockHeight(100)

	move := &types.MsgStructMove{Creator: "structs1alice", StructId: "5-1"}
	_, err := handler(ctx, mockTx{msgs: []sdk.Msg{move}}, false)
	require.NoError(t, err)

	attack := &types.MsgStructAttack{Creator: "structs1alice", OperatingStructId: "5-2"}
	_, err = handler(ctx, mockTx{msgs: []sdk.Msg{attack}}, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already used charge action this block")
}

func TestIntegration_CheckTxThrottleEnforced(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	cap := uint64(2)
	decorators := []sdk.AnteDecorator{
		sante.NewCheckTxThrottleDecorator(cap),
		sante.NewStructsDecorator(mk, 40),
		sante.NewThrottleDecorator(mk),
	}
	handler := sdk.ChainAnteDecorators(decorators...)

	ctx := freeCtx().WithBlockHeight(100).WithIsCheckTx(true)

	for i := 0; i < int(cap); i++ {
		msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: fmt.Sprintf("2-%d", i), DestinationLocationId: "7-1"}
		_, err := handler(ctx, mockTx{msgs: []sdk.Msg{msg}}, false)
		require.NoError(t, err, "tx %d should pass", i)
	}

	// One over the cap
	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-99", DestinationLocationId: "7-1"}
	_, err := handler(ctx, mockTx{msgs: []sdk.Msg{msg}}, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeded CheckTx free-tx cap")
}

func TestIntegration_CheckTxThrottleResetsOnNewBlock(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	cap := uint64(1)
	decorators := []sdk.AnteDecorator{
		sante.NewCheckTxThrottleDecorator(cap),
		sante.NewStructsDecorator(mk, 40),
		sante.NewThrottleDecorator(mk),
	}
	handler := sdk.ChainAnteDecorators(decorators...)

	ctx100 := freeCtx().WithBlockHeight(100).WithIsCheckTx(true)
	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	_, err := handler(ctx100, mockTx{msgs: []sdk.Msg{msg}}, false)
	require.NoError(t, err)

	// Same block: should be rejected
	_, err = handler(ctx100, mockTx{msgs: []sdk.Msg{msg}}, false)
	require.Error(t, err)

	// New block: counter resets
	ctx101 := freeCtx().WithBlockHeight(101).WithIsCheckTx(true)
	_, err = handler(ctx101, mockTx{msgs: []sdk.Msg{msg}}, false)
	require.NoError(t, err)
}

func TestIntegration_NonFreeTxBypassesAllStructsChecks(t *testing.T) {
	mk := newMockAnteKeeper()
	handler := structsSubChain(mk)

	tx := mockTx{msgs: []sdk.Msg{&mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}}}
	ctx := newTestCtx().WithBlockHeight(100)

	_, err := handler(ctx, tx, false)
	require.NoError(t, err)
}
