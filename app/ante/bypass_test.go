package ante_test

import (
	"fmt"
	"testing"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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

func TestNestedStructsMsgDecorator_RejectsStructsInsideNestedMsgExec(t *testing.T) {
	structsMsg, err := codectypes.NewAnyWithValue(&types.MsgPlayerSend{
		Creator:     "structs1granter",
		FromAddress: "structs1granter",
		ToAddress:   "structs1grantee",
	})
	require.NoError(t, err)
	innerExec, err := codectypes.NewAnyWithValue(&authz.MsgExec{
		Grantee: "structs1middle",
		Msgs:    []*codectypes.Any{structsMsg},
	})
	require.NoError(t, err)

	dec := sante.NewNestedStructsMsgDecorator()
	next, called := identityHandler()
	tx := mockTx{msgs: []sdk.Msg{&authz.MsgExec{
		Grantee: "structs1grantee",
		Msgs:    []*codectypes.Any{innerExec},
	}}}

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

// The two decorators below kept their fee-based gate after StructsDecorator and
// ThrottleDecorator moved to a message-based one, so a mixed tx still walked
// past them. chain.json sets fixed_min_gas_price to 0 and the mempool fee check
// returns early on a zero min price, so "paid" was free in the default config
// and neither rate limit cost anything to skip.

func TestCheckTxThrottleDecorator_MixedPaidTxStillCounted(t *testing.T) {
	dec := sante.NewCheckTxThrottleDecorator(2)
	ctx := newTestCtx().WithIsCheckTx(true)

	// Two admissions are the whole quota, mixed or not.
	for attempt := 1; attempt <= 2; attempt++ {
		next, called := identityHandler()
		tx := mixedTx(&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"})

		_, err := dec.AnteHandle(ctx, tx, false, next)
		require.NoError(t, err, "attempt %d is inside the cap", attempt)
		require.True(t, *called)
	}

	next, called := identityHandler()
	tx := mixedTx(&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-2"})

	_, err := dec.AnteHandle(ctx, tx, false, next)
	require.Error(t, err, "pairing a bank send with gameplay must not buy extra admissions")
	require.True(t, sante.ErrCheckTxAddrCapExceeded.Is(err))
	require.False(t, *called)
}

// Widening the gate must not turn a Structs rate limiter into a chain-wide one:
// a tx holding no Structs and no staking message is another module's business.
func TestCheckTxThrottleDecorator_PureNonStructsTxUncounted(t *testing.T) {
	dec := sante.NewCheckTxThrottleDecorator(1)
	ctx := newTestCtx().WithIsCheckTx(true)

	for attempt := 1; attempt <= 5; attempt++ {
		next, called := identityHandler()
		tx := mockTx{msgs: []sdk.Msg{&banktypes.MsgSend{FromAddress: "structs1alice", ToAddress: "structs1bob"}}}

		_, err := dec.AnteHandle(ctx, tx, false, next)
		require.NoError(t, err, "attempt %d: a bank-only tx consumes no Structs quota", attempt)
		require.True(t, *called)
	}
}

func TestStakingThrottleDecorator_MixedPaidTxStillThrottled(t *testing.T) {
	mk := newMockAnteKeeper()
	dec := sante.NewStakingThrottleDecorator(mk)
	ctx := newTestCtx()

	mixedDelegate := func() mockTx {
		return mockTx{msgs: []sdk.Msg{
			&stakingtypes.MsgDelegate{DelegatorAddress: "structs1alice", ValidatorAddress: "structsvaloper1x"},
			&banktypes.MsgSend{FromAddress: "structs1alice", ToAddress: "structs1bob"},
		}}
	}

	next, called := identityHandler()
	_, err := dec.AnteHandle(ctx, mixedDelegate(), false, next)
	require.NoError(t, err, "the first staking operation of the block is allowed")
	require.True(t, *called)

	next, called = identityHandler()
	_, err = dec.AnteHandle(ctx, mixedDelegate(), false, next)
	require.Error(t, err, "one staking operation per address per block, however the tx is composed")
	require.True(t, sante.ErrStakingAlreadySubmittedThisBlock.Is(err))
	require.False(t, *called)
}

// The loop used to reject anything absent from StakingSignerExtractors, which
// was only safe while IsFreeStakingTransaction had already guaranteed every
// message was a staking one. Under a message-based gate that would reject the
// non-staking half of the very txs this is meant to throttle.
func TestStakingThrottleDecorator_SkipsNonStakingMessages(t *testing.T) {
	mk := newMockAnteKeeper()
	dec := sante.NewStakingThrottleDecorator(mk)
	next, called := identityHandler()

	tx := mockTx{msgs: []sdk.Msg{
		&stakingtypes.MsgDelegate{DelegatorAddress: "structs1alice", ValidatorAddress: "structsvaloper1x"},
		&banktypes.MsgSend{FromAddress: "structs1alice", ToAddress: "structs1bob"},
		&types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"},
	}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err, "messages from other modules are gated by their own modules, not rejected here")
	require.True(t, *called)
	require.True(t, mk.throttleKeys["staking/structs1alice"], "the staking message still reserves its key")
}

func TestContainsStakingMessage(t *testing.T) {
	bankMsg := &banktypes.MsgSend{FromAddress: "structs1a", ToAddress: "structs1b"}
	structsMsg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1"}
	stakingMsg := &stakingtypes.MsgDelegate{DelegatorAddress: "structs1alice"}

	require.False(t, sante.ContainsStakingMessage(nil))
	require.False(t, sante.ContainsStakingMessage([]sdk.Msg{bankMsg, structsMsg}))
	require.True(t, sante.ContainsStakingMessage([]sdk.Msg{stakingMsg}))
	require.True(t, sante.ContainsStakingMessage([]sdk.Msg{bankMsg, stakingMsg}))
}

/* TestStructsDecorator_RecoveryMessagesSurviveAnExhaustedQuota is the regression
 * on a delegated key blocking its own eviction.
 *
 * The per-player message quota is keyed by player, so every address of a player
 * draws on one budget, and it is charged here in the ante — before any handler
 * runs, and therefore whether or not the message succeeds. The SDK commits the
 * ante cache and discards only the message cache, so forty messages that all
 * fail still spend the whole block's budget.
 *
 * That is griefing everywhere except one place, where it is a trap: the lockout
 * covers AddressRevoke, so the key doing the griefing blocks the one message
 * that would remove it. The rule this breaks is already written down for
 * delegation transfers — a revoke is the response to a compromised key, so
 * anything that key can sustain must not be able to block it.
 */
func TestStructsDecorator_RecoveryMessagesSurviveAnExhaustedQuota(t *testing.T) {
	const playerId = "1-1"
	const primary = "structs1primary"
	const delegate = "structs1delegate"

	newDecorator := func() (sante.StructsDecorator, *mockAnteKeeper) {
		mk := newMockAnteKeeper()
		mk.playerIndexes[primary] = 1
		mk.playerIndexes[delegate] = 1
		mk.setPrimaryAddress(playerId, primary)
		mk.permissions[fmt.Sprintf("%d-%s@0", types.ObjectType_address, primary)] = types.PermAll
		mk.permissions[fmt.Sprintf("%d-%s@0", types.ObjectType_address, delegate)] = types.PermAll
		return sante.NewStructsDecorator(mk, 2), mk
	}

	// The quota is only charged in DeliverTx.
	deliverCtx := func() sdk.Context { return newTestCtx() }

	exhaust := func(t *testing.T, dec sante.StructsDecorator) {
		t.Helper()
		next, _ := identityHandler()
		tx := mockTx{msgs: []sdk.Msg{
			&types.MsgPlayerUpdateName{Creator: delegate},
			&types.MsgPlayerUpdateName{Creator: delegate},
		}}
		_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
		require.NoError(t, err, "the delegate's own messages are within the cap")
	}

	t.Run("an ordinary message is capped once the delegate has spent the quota", func(t *testing.T) {
		dec, _ := newDecorator()
		exhaust(t, dec)

		next, called := identityHandler()
		tx := mockTx{msgs: []sdk.Msg{&types.MsgPlayerUpdateName{Creator: primary}}}

		_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
		require.Error(t, err, "the quota is shared, which is the behaviour being preserved")
		require.True(t, sante.ErrPlayerMsgCapExceeded.Is(err))
		require.False(t, *called)
	})

	t.Run("a revoke from the primary address still gets through", func(t *testing.T) {
		dec, _ := newDecorator()
		exhaust(t, dec)

		next, called := identityHandler()
		tx := mockTx{msgs: []sdk.Msg{&types.MsgAddressRevoke{Creator: primary, Address: delegate}}}

		_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
		require.NoError(t, err, "the key being revoked must not be able to block the revoke")
		require.True(t, *called)
	})

	t.Run("so does a primary-address rotation", func(t *testing.T) {
		dec, _ := newDecorator()
		exhaust(t, dec)

		next, called := identityHandler()
		tx := mockTx{msgs: []sdk.Msg{
			&types.MsgPlayerUpdatePrimaryAddress{Creator: primary, PrimaryAddress: "structs1replacement"},
		}}

		_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
		require.NoError(t, err)
		require.True(t, *called)
	})

	// The exemption is the primary address, not the message type: a delegate
	// cannot mint itself unlimited free messages by naming a recovery one.
	t.Run("a delegate gets no exemption from a recovery message", func(t *testing.T) {
		dec, _ := newDecorator()
		exhaust(t, dec)

		next, called := identityHandler()
		tx := mockTx{msgs: []sdk.Msg{&types.MsgAddressRevoke{Creator: delegate, Address: primary}}}

		_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
		require.Error(t, err, "only the recovery identity is exempt")
		require.True(t, sante.ErrPlayerMsgCapExceeded.Is(err))
		require.False(t, *called)
	})

	// And the exemption does not itself become a free channel: recovery messages
	// from the primary spend nothing, so they cannot exhaust the quota either.
	t.Run("recovery messages spend nothing", func(t *testing.T) {
		dec, mk := newDecorator()

		for i := 0; i < 10; i++ {
			next, _ := identityHandler()
			tx := mockTx{msgs: []sdk.Msg{&types.MsgAddressRevoke{Creator: primary, Address: delegate}}}
			_, err := dec.AnteHandle(deliverCtx(), tx, false, next)
			require.NoError(t, err)
		}

		require.Zero(t, mk.msgCounts[playerId], "no recovery message may touch the counter")
	})
}
