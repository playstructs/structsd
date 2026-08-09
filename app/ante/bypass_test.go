package ante_test

import (
	"fmt"
	"testing"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	sante "structs/app/ante"
	"structs/x/structs/types"
)

// The Structs ante checks used to run only on free txs, and IsFreeTransaction
// requires every message in the tx to be a Structs message. Pairing one gameplay
// message with any non-Structs message therefore bought a way past the player
// registration, permission, charge and throttle gating for the price of a normal
// fee. Gating follows the message now, not the fee.

func mixedTx(structsMsg sdk.Msg) mockTx {
	return mockTx{msgs: []sdk.Msg{
		structsMsg,
		&banktypes.MsgSend{FromAddress: "structs1payer", ToAddress: "structs1payee"},
	}}
}

func TestStructsDecorator_MixedPaidTxStillPermissionChecked(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1weak"] = 7
	// Registered, but without PermTokenTransfer.
	mk.permissions[fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1weak")] = types.PermPlay

	dec := sante.NewStructsDecorator(mk, 40)
	next, called := identityHandler()

	tx := mixedTx(&types.MsgPlayerSend{
		Creator:     "structs1weak",
		FromAddress: "structs1primary",
		ToAddress:   "structs1attacker",
	})

	// Deliberately a plain context: no free-gas flag, i.e. a fee-paying tx.
	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.Error(t, err, "a fee-paying mixed tx must not skip the permission check")
	require.True(t, sante.ErrMissingPermission.Is(err))
	require.False(t, *called)
}

func TestStructsDecorator_MixedPaidTxChecksRegistration(t *testing.T) {
	dec := sante.NewStructsDecorator(newMockAnteKeeper(), 40)
	next, called := identityHandler()

	tx := mixedTx(&types.MsgFleetMove{Creator: "structs1unknown", FleetId: "2-1", DestinationLocationId: "7-1"})

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.Error(t, err)
	require.True(t, sante.ErrUnregisteredAddress.Is(err))
	require.False(t, *called)
}

// Non-Structs messages are gated by their own modules and have no player-owned
// creator, so they must pass through rather than being rejected.
func TestStructsDecorator_NonStructsMessagesPassThrough(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1
	mk.permissions[fmt.Sprintf("%d-%s@0", types.ObjectType_address, "structs1alice")] = types.PermPlay

	dec := sante.NewStructsDecorator(mk, 40)
	next, called := identityHandler()

	tx := mixedTx(&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"})

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestStructsDecorator_TxWithoutStructsMessagesIsIgnored(t *testing.T) {
	dec := sante.NewStructsDecorator(newMockAnteKeeper(), 40)
	next, called := identityHandler()

	tx := mockTx{msgs: []sdk.Msg{&banktypes.MsgSend{FromAddress: "structs1a", ToAddress: "structs1b"}}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestThrottleDecorator_MixedPaidTxStillThrottled(t *testing.T) {
	mk := newMockAnteKeeper()
	mk.playerIndexes["structs1alice"] = 1

	dec := sante.NewThrottleDecorator(mk)
	next, called := identityHandler()

	// Two moves of the same fleet in one tx must collide on the throttle key
	// regardless of the tx paying a fee.
	tx := mockTx{msgs: []sdk.Msg{
		&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"},
		&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-2"},
		&banktypes.MsgSend{FromAddress: "structs1a", ToAddress: "structs1b"},
	}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.Error(t, err)
	require.True(t, sante.ErrDuplicateThrottleKeyInTx.Is(err))
	require.False(t, *called)
}

// A grantee could otherwise execute gameplay through authz with none of the
// Structs ante gating applied, because nested messages never appear in
// tx.GetMsgs().
func TestNestedStructsMsgDecorator_RejectsStructsInsideMsgExec(t *testing.T) {
	inner, err := codectypes.NewAnyWithValue(&types.MsgPlayerSend{
		Creator:     "structs1granter",
		FromAddress: "structs1granter",
		ToAddress:   "structs1grantee",
	})
	require.NoError(t, err)

	dec := sante.NewNestedStructsMsgDecorator()
	next, called := identityHandler()

	tx := mockTx{msgs: []sdk.Msg{&authz.MsgExec{Grantee: "structs1grantee", Msgs: []*codectypes.Any{inner}}}}

	_, err = dec.AnteHandle(newTestCtx(), tx, false, next)
	require.Error(t, err)
	require.True(t, sante.ErrNestedStructsMessage.Is(err))
	require.False(t, *called)
}

func TestNestedStructsMsgDecorator_AllowsNonStructsInsideMsgExec(t *testing.T) {
	inner, err := codectypes.NewAnyWithValue(&banktypes.MsgSend{
		FromAddress: "structs1granter",
		ToAddress:   "structs1grantee",
	})
	require.NoError(t, err)

	dec := sante.NewNestedStructsMsgDecorator()
	next, called := identityHandler()

	tx := mockTx{msgs: []sdk.Msg{&authz.MsgExec{Grantee: "structs1grantee", Msgs: []*codectypes.Any{inner}}}}

	_, err = dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestNestedStructsMsgDecorator_IgnoresOrdinaryTxs(t *testing.T) {
	dec := sante.NewNestedStructsMsgDecorator()
	next, called := identityHandler()

	tx := mockTx{msgs: []sdk.Msg{&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1"}}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestContainsGatedStructsMessage(t *testing.T) {
	bankMsg := &banktypes.MsgSend{FromAddress: "structs1a", ToAddress: "structs1b"}
	structsMsg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1"}

	require.False(t, sante.ContainsGatedStructsMessage(nil))
	require.False(t, sante.ContainsGatedStructsMessage([]sdk.Msg{bankMsg}))
	require.True(t, sante.ContainsGatedStructsMessage([]sdk.Msg{structsMsg}))
	require.True(t, sante.ContainsGatedStructsMessage([]sdk.Msg{bankMsg, structsMsg}))

	// Governance params are authority-signed, so there is no player to gate.
	require.False(t, sante.ContainsGatedStructsMessage([]sdk.Msg{&types.MsgUpdateParams{}}))
}
