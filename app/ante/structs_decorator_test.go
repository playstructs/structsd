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
	require.True(t, sante.ErrUnregisteredAddress.Is(err))
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
	require.True(t, sante.ErrMissingPermission.Is(err))
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
	require.True(t, sante.ErrPlayerDischargedThisBlock.Is(err))
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
	require.True(t, sante.ErrPlayerMsgCapExceeded.Is(err))
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

/* TestStructsDecorator_PlayerMsgCapAppliesAtAdmission covers both halves of the
 * split between the admission cap and the authoritative one.
 *
 * The authoritative per-player counter lives in the transient store and is only
 * touched in DeliverTx: counting the same transaction at CheckTx and again at
 * delivery would double-charge the quota. That left admission bounded per
 * *address* while the cap it predicts is per *player*, so a player with several
 * registered addresses could fill the mempool with transactions delivery was
 * always going to refuse - free to send, and paid for in block bytes.
 *
 * A node-local count closes that without touching the authoritative one, which
 * is what the second assertion here pins.
 */
func TestStructsDecorator_PlayerMsgCapAppliesAtAdmission(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	addrPermId := fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")
	mk.permissions[addrPermId] = types.PermPlay

	dec := sante.NewStructsDecorator(mk, 2)
	next, called := identityHandler()

	msgs := make([]sdk.Msg, 5)
	for i := range msgs {
		msgs[i] = &types.MsgFleetMove{Creator: "structs1alice", FleetId: fmt.Sprintf("2-%d", i), DestinationLocationId: "7-1"}
	}

	ctx := freeCtx().WithBlockHeight(100).WithIsCheckTx(true)
	_, err := dec.AnteHandle(ctx, mockTx{msgs: msgs}, false, next)
	require.Error(t, err, "a transaction that delivery will refuse must not be admitted")
	require.True(t, sante.ErrPlayerMsgCapExceeded.Is(err))
	require.False(t, *called)

	require.Zero(t, mk.msgCounts["1-1"],
		"admission must not touch the authoritative counter; that is what would double-charge at delivery")
}

/* TestStructsDecorator_AdmissionCapIsPerPlayerNotPerAddress is the attack the
 * address-keyed throttle could not see. Two addresses of one player each stay
 * inside their own admission quota while together exceeding the player cap that
 * delivery enforces.
 */
func TestStructsDecorator_AdmissionCapIsPerPlayerNotPerAddress(t *testing.T) {
	mk := newMockAnteKeeper()
	for _, addr := range []string{"structs1alice", "structs1alice2"} {
		mk.playerIndexes[addr] = 1
		mk.permissions[fmt.Sprintf("%d-%s@0", types.ObjectType_address, addr)] = types.PermPlay
	}

	dec := sante.NewStructsDecorator(mk, 2)
	ctx := freeCtx().WithBlockHeight(100).WithIsCheckTx(true)

	next, _ := identityHandler()
	_, err := dec.AnteHandle(ctx, mockTx{msgs: []sdk.Msg{
		&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"},
		&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-2", DestinationLocationId: "7-1"},
	}}, false, next)
	require.NoError(t, err, "the player's first two messages are within the cap")

	next2, called2 := identityHandler()
	_, err = dec.AnteHandle(ctx, mockTx{msgs: []sdk.Msg{
		&types.MsgFleetMove{Creator: "structs1alice2", FleetId: "2-3", DestinationLocationId: "7-1"},
	}}, false, next2)
	require.Error(t, err,
		"a second address of the same player must draw on the same admission quota")
	require.True(t, sante.ErrPlayerMsgCapExceeded.Is(err))
	require.False(t, *called2)
}

// The admission counter is per height, so a new block starts a fresh quota.
func TestStructsDecorator_AdmissionCapResetsWithHeight(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	mk.permissions[fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")] = types.PermPlay

	dec := sante.NewStructsDecorator(mk, 1)
	tx := mockTx{msgs: []sdk.Msg{
		&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"},
	}}

	next, _ := identityHandler()
	_, err := dec.AnteHandle(freeCtx().WithBlockHeight(100).WithIsCheckTx(true), tx, false, next)
	require.NoError(t, err)

	_, err = dec.AnteHandle(freeCtx().WithBlockHeight(100).WithIsCheckTx(true), tx, false, next)
	require.Error(t, err, "the quota is spent for this height")

	_, err = dec.AnteHandle(freeCtx().WithBlockHeight(101).WithIsCheckTx(true), tx, false, next)
	require.NoError(t, err, "a new height starts a fresh admission quota")
}

/* ReCheckTx and simulate must not spend the admission quota. Both run on
 * transactions that already passed fresh CheckTx, so counting them again would
 * evict transactions that were legitimately admitted.
 */
func TestStructsDecorator_AdmissionCapSkipsRecheckAndSimulate(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	mk.permissions[fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")] = types.PermPlay

	dec := sante.NewStructsDecorator(mk, 1)
	msgs := []sdk.Msg{
		&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"},
		&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-2", DestinationLocationId: "7-1"},
	}

	next, _ := identityHandler()
	_, err := dec.AnteHandle(freeCtx().WithBlockHeight(100).WithIsCheckTx(true).WithIsReCheckTx(true), mockTx{msgs: msgs}, false, next)
	require.NoError(t, err, "reCheckTx must not apply the admission cap")

	_, err = dec.AnteHandle(freeCtx().WithBlockHeight(100).WithIsCheckTx(true), mockTx{msgs: msgs}, true, next)
	require.NoError(t, err, "simulate is an estimate, not admission")
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
	require.True(t, sante.ErrPlayerMsgCapExceeded.Is(err))
}
