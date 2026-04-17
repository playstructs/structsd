package ante_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sante "structs/app/ante"
	"structs/x/structs/types"
)

func freeCtx() sdk.Context {
	return newTestCtx().WithValue(sante.FreeGasCtxKey(), true)
}

func TestStructsDecorator_UnregisteredAddress(t *testing.T) {
	mk := newMockAnteKeeper()
	dec := sante.NewStructsDecorator(mk, 40)
	next, _ := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1unknown", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(freeCtx(), tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not registered as player")
}

func TestStructsDecorator_RegisteredWithCorrectPerm(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	dec := sante.NewStructsDecorator(mk, 40)
	next, called := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	ctx := freeCtx().WithIsCheckTx(true)
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestStructsDecorator_RegisteredWithWrongPerm(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermAdmin // has Admin but not Play

	dec := sante.NewStructsDecorator(mk, 40)
	next, _ := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	ctx := freeCtx().WithIsCheckTx(true)
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "lacks permission")
}

func TestStructsDecorator_DynamicPermSkipsCheck(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	// No permissions set at all -- dynamic perm message should skip the check

	dec := sante.NewStructsDecorator(mk, 40)
	next, called := identityHandler()

	msg := &types.MsgPermissionGrantOnAddress{Creator: "structs1alice", Address: "structs1bob", Permissions: uint64(types.PermPlay)}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	ctx := freeCtx().WithIsCheckTx(true)
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestStructsDecorator_NonFreeTxPassesThrough(t *testing.T) {
	mk := newMockAnteKeeper()
	dec := sante.NewStructsDecorator(mk, 40)
	next, called := identityHandler()

	tx := mockTx{msgs: []sdk.Msg{&mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}}}

	ctx := newTestCtx() // no free flag
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestStructsDecorator_ChargeCheckRejectsSameBlock(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	playerId := fmt.Sprintf("%d-%d", types.ObjectType_player, 1)
	lastActionAttrId := fmt.Sprintf("%d-%s", types.GridAttributeType_lastAction, playerId)
	mk.gridAttrs[lastActionAttrId] = 100 // discharged at block 100

	dec := sante.NewStructsDecorator(mk, 40)
	next, _ := identityHandler()

	msg := &types.MsgStructMove{Creator: "structs1alice", StructId: "5-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	ctx := freeCtx().WithBlockHeight(100) // same block as lastAction
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "zero charge")
}

func TestStructsDecorator_ChargeCheckPassesDifferentBlock(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	playerId := fmt.Sprintf("%d-%d", types.ObjectType_player, 1)
	lastActionAttrId := fmt.Sprintf("%d-%s", types.GridAttributeType_lastAction, playerId)
	mk.gridAttrs[lastActionAttrId] = 99 // discharged at block 99

	dec := sante.NewStructsDecorator(mk, 40)
	next, called := identityHandler()

	msg := &types.MsgStructMove{Creator: "structs1alice", StructId: "5-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	ctx := freeCtx().WithBlockHeight(100).WithIsCheckTx(true)
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestStructsDecorator_PlayerMsgCapExceeded(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	cap := uint64(3)
	dec := sante.NewStructsDecorator(mk, cap)
	next, _ := identityHandler()

	msgs := make([]sdk.Msg, 4)
	for i := range msgs {
		msgs[i] = &types.MsgFleetMove{Creator: "structs1alice", FleetId: fmt.Sprintf("2-%d", i), DestinationLocationId: "7-1"}
	}
	tx := mockTx{msgs: msgs}

	// DeliverTx context: not CheckTx, not simulate
	ctx := freeCtx().WithBlockHeight(100)
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeded per-block message cap")
}

func TestStructsDecorator_PlayerMsgCapPassesUnderLimit(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	dec := sante.NewStructsDecorator(mk, 40)
	next, called := identityHandler()

	msgs := make([]sdk.Msg, 5)
	for i := range msgs {
		msgs[i] = &types.MsgFleetMove{Creator: "structs1alice", FleetId: fmt.Sprintf("2-%d", i), DestinationLocationId: "7-1"}
	}
	tx := mockTx{msgs: msgs}

	ctx := freeCtx().WithBlockHeight(100)
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestStructsDecorator_PlayerMsgCapSkippedDuringCheckTx(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	cap := uint64(2)
	dec := sante.NewStructsDecorator(mk, cap)
	next, called := identityHandler()

	msgs := make([]sdk.Msg, 5)
	for i := range msgs {
		msgs[i] = &types.MsgFleetMove{Creator: "structs1alice", FleetId: fmt.Sprintf("2-%d", i), DestinationLocationId: "7-1"}
	}
	tx := mockTx{msgs: msgs}

	// CheckTx: msg cap enforcement is skipped
	ctx := freeCtx().WithBlockHeight(100).WithIsCheckTx(true)
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestStructsDecorator_DefaultPlayerMsgCapApplied(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	dec := sante.NewStructsDecorator(mk, 0)
	next, _ := identityHandler()

	msgs := make([]sdk.Msg, 41)
	for i := range msgs {
		msgs[i] = &types.MsgFleetMove{Creator: "structs1alice", FleetId: fmt.Sprintf("2-%d", i), DestinationLocationId: "7-1"}
	}
	tx := mockTx{msgs: msgs}

	ctx := freeCtx().WithBlockHeight(100)
	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeded per-block message cap")
}
