package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// These tests cover the shared-cache authorization bug: CurrentContext caches
// PlayerCache by player id, so two addresses of the same player resolve to the
// same object. ActiveAddress used to live on that object, so loading a
// caller-supplied address second overwrote the signer and every Layer 1
// permission check gated on the wrong key.
//
// The invariant under test: a signing key may only ever exercise permissions it
// holds itself, no matter which of its player's addresses the message names.

type multiKeyPlayer struct {
	player types.Player
	// primary holds PermAll.
	primary string
	// weak holds PermPlay only, and stands in for a compromised or
	// deliberately narrow hot key.
	weak string
	// spender holds PermPlay plus every asset permission, and is the positive
	// control: a scoped key that does hold the bit must keep working.
	spender string
}

func setupMultiKeyPlayer(t *testing.T, k keeperlib.Keeper, ctx context.Context, seed string) multiKeyPlayer {
	t.Helper()

	primaryAcc := sdk.AccAddress(seed + "_primary_padding_address_1234567890")
	player := types.Player{
		Creator:        primaryAcc.String(),
		PrimaryAddress: primaryAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	weakAcc := sdk.AccAddress(seed + "_weak_padding_address_1234567890")
	weak := weakAcc.String()
	k.SetPlayerIndexForAddress(ctx, weak, player.Index)
	k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(weak), types.PermPlay)

	spenderAcc := sdk.AccAddress(seed + "_spender_padding_address_1234567890")
	spender := spenderAcc.String()
	k.SetPlayerIndexForAddress(ctx, spender, player.Index)
	k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(spender),
		types.PermPlay|types.PermAssetsAll)

	return multiKeyPlayer{
		player:  player,
		primary: player.PrimaryAddress,
		weak:    weak,
		spender: spender,
	}
}

func TestSignerIdentity_PlayerSendFromStrongerAddress(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	mk := setupMultiKeyPlayer(t, k, ctx, "send")
	primaryAcc, err := sdk.AccAddressFromBech32(mk.primary)
	require.NoError(t, err)

	coins := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(1000)))
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, primaryAcc, coins))

	attackerAcc := sdk.AccAddress("attacker_padding_address_1234567890")

	_, err = ms.PlayerSend(wctx, &types.MsgPlayerSend{
		Creator:     mk.weak,
		FromAddress: mk.primary,
		ToAddress:   attackerAcc.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(500))),
	})
	require.Error(t, err, "a PermPlay-only key must not spend from the primary address")
	require.Contains(t, err.Error(), "permission")

	require.Equal(t, math.NewInt(1000), k.BankKeeper().SpendableCoin(ctx, primaryAcc, "ualpha").Amount,
		"the rejected send must not move funds")
	require.True(t, k.BankKeeper().SpendableCoin(ctx, attackerAcc, "ualpha").IsZero())
}

// The scoped key that actually holds PermTokenTransfer must still be able to
// spend from another address of its own player. This is the capability the fix
// has to preserve.
func TestSignerIdentity_PlayerSendWithScopedKeyStillWorks(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	mk := setupMultiKeyPlayer(t, k, ctx, "scoped")
	primaryAcc, err := sdk.AccAddressFromBech32(mk.primary)
	require.NoError(t, err)

	coins := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(1000)))
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, primaryAcc, coins))

	destAcc := sdk.AccAddress("scopeddest_padding_address_1234567890")

	_, err = ms.PlayerSend(wctx, &types.MsgPlayerSend{
		Creator:     mk.spender,
		FromAddress: mk.primary,
		ToAddress:   destAcc.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(400))),
	})
	require.NoError(t, err, "a key holding PermTokenTransfer may spend from its player's primary")

	require.Equal(t, math.NewInt(600), k.BankKeeper().SpendableCoin(ctx, primaryAcc, "ualpha").Amount)
	require.Equal(t, math.NewInt(400), k.BankKeeper().SpendableCoin(ctx, destAcc, "ualpha").Amount)
}

func TestSignerIdentity_ReactorHandlersRejectWeakKey(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	mk := setupMultiKeyPlayer(t, k, ctx, "reactor")
	primaryAcc, err := sdk.AccAddressFromBech32(mk.primary)
	require.NoError(t, err)

	validatorAddress := sdk.ValAddress(primaryAcc.Bytes())
	testAddValidator(k, validatorAddress, math.NewInt(1000000))
	reactor := k.AppendReactor(ctx, types.Reactor{
		Validator:  validatorAddress.String(),
		RawAddress: validatorAddress.Bytes(),
	})

	bondDenom := "stake"
	coins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1000)))
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, primaryAcc, coins))

	amount := sdk.NewCoin(bondDenom, math.NewInt(100))

	t.Run("infuse", func(t *testing.T) {
		_, err := ms.ReactorInfuse(wctx, &types.MsgReactorInfuse{
			Creator:          mk.weak,
			DelegatorAddress: mk.primary,
			ValidatorAddress: reactor.Validator,
			Amount:           amount,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "permission")
	})

	t.Run("defuse", func(t *testing.T) {
		_, err := ms.ReactorDefuse(wctx, &types.MsgReactorDefuse{
			Creator:          mk.weak,
			DelegatorAddress: mk.primary,
			ValidatorAddress: reactor.Validator,
			Amount:           amount,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "permission")
	})

	t.Run("begin migration", func(t *testing.T) {
		_, err := ms.ReactorBeginMigration(wctx, &types.MsgReactorBeginMigration{
			Creator:             mk.weak,
			DelegatorAddress:    mk.primary,
			ValidatorSrcAddress: reactor.Validator,
			ValidatorDstAddress: reactor.Validator,
			Amount:              amount,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "permission")
	})

	t.Run("cancel defusion", func(t *testing.T) {
		_, err := ms.ReactorCancelDefusion(wctx, &types.MsgReactorCancelDefusion{
			Creator:          mk.weak,
			DelegatorAddress: mk.primary,
			ValidatorAddress: reactor.Validator,
			Amount:           amount,
			CreationHeight:   1,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "permission")
	})

	require.Equal(t, math.NewInt(1000), k.BankKeeper().SpendableCoin(ctx, primaryAcc, bondDenom).Amount,
		"no rejected reactor action may move the primary address's tokens")
}

// A weak key rewriting the primary address's permission bits is the lockout
// vector: PermissionSetOnAddress and PermissionRevokeOnAddress have no
// ante-level permission gate (they are DynamicPermissionMessages), so the
// handler check is the only thing standing in the way.
func TestSignerIdentity_PermissionOnAddressRejectsWeakKey(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	mk := setupMultiKeyPlayer(t, k, ctx, "perms")
	primaryPermId := keeperlib.GetAddressPermissionIDBytes(mk.primary)

	t.Run("set", func(t *testing.T) {
		_, err := ms.PermissionSetOnAddress(wctx, &types.MsgPermissionSetOnAddress{
			Creator:     mk.weak,
			Address:     mk.primary,
			Permissions: uint64(types.PermPlay),
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "permission")
		require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, primaryPermId),
			"the primary address must keep its permissions")
	})

	t.Run("revoke", func(t *testing.T) {
		_, err := ms.PermissionRevokeOnAddress(wctx, &types.MsgPermissionRevokeOnAddress{
			Creator:     mk.weak,
			Address:     mk.primary,
			Permissions: uint64(types.PermAll),
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "permission")
		require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, primaryPermId),
			"a PermPlay-only key must not strip the primary address")
	})

	t.Run("grant", func(t *testing.T) {
		_, err := ms.PermissionGrantOnAddress(wctx, &types.MsgPermissionGrantOnAddress{
			Creator:     mk.weak,
			Address:     mk.primary,
			Permissions: uint64(types.PermAll),
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "permission")
	})

	// The primary holds PermAll, so it holds both the bits being written and the
	// bits being overwritten, and must still be able to rescope its own keys.
	t.Run("primary may rescope a weaker key", func(t *testing.T) {
		_, err := ms.PermissionSetOnAddress(wctx, &types.MsgPermissionSetOnAddress{
			Creator:     mk.primary,
			Address:     mk.weak,
			Permissions: uint64(types.PermPlay | types.PermUpdate),
		})
		require.NoError(t, err)
		require.Equal(t, types.PermPlay|types.PermUpdate,
			k.GetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(mk.weak)))
	})
}

func TestSignerIdentity_AddressRevokeRejectsWeakKey(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	mk := setupMultiKeyPlayer(t, k, ctx, "revoke")

	// The spender address holds PermDelete via PermAssetsAll only if that bundle
	// includes it; either way the weak key does not, so revoking it must fail.
	_, err := ms.AddressRevoke(wctx, &types.MsgAddressRevoke{
		Creator: mk.weak,
		Address: mk.spender,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission")

	require.NotZero(t, k.GetPlayerIndexFromAddress(ctx, mk.spender),
		"the address must still be registered after a rejected revoke")
}

func TestSignerIdentity_UpdatePrimaryAddressRejectsWeakKey(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	mk := setupMultiKeyPlayer(t, k, ctx, "primary")

	_, err := ms.PlayerUpdatePrimaryAddress(wctx, &types.MsgPlayerUpdatePrimaryAddress{
		Creator:        mk.weak,
		PrimaryAddress: mk.spender,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission")

	unchanged, found := k.GetPlayer(ctx, mk.player.Id)
	require.True(t, found)
	require.Equal(t, mk.primary, unchanged.PrimaryAddress)
}

// The signer is per-operation state, so a handler must not be able to end up
// with two acting identities. GetSigningPlayer is write-once and rejects the
// second, different address rather than silently letting the last write win.
func TestSignerIdentity_SecondSignerRejected(t *testing.T) {
	k, _, ctx := setupMsgServer(t)

	mk := setupMultiKeyPlayer(t, k, ctx, "twosigners")

	cc := k.NewCurrentContext(ctx)

	_, err := cc.GetSigningPlayer(mk.primary)
	require.NoError(t, err)
	require.Equal(t, mk.primary, cc.SignerAddress())

	// Same address again is a no-op, not an error.
	_, err = cc.GetSigningPlayer(mk.primary)
	require.NoError(t, err)
	require.Equal(t, mk.primary, cc.SignerAddress())

	_, err = cc.GetSigningPlayer(mk.weak)
	require.Error(t, err)
	require.Equal(t, mk.primary, cc.SignerAddress(),
		"the established signer must survive a second attempt")

	// A plain lookup of another address of the same player must not disturb it.
	_, err = cc.GetPlayerByAddress(mk.weak)
	require.NoError(t, err)
	require.Equal(t, mk.primary, cc.SignerAddress())
}
