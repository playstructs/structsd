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

// Opening an agreement debits the player's primary address for the collateral, so
// it is a token spend and needs the same bit PlayerSend does. The access policy
// only decides who may contract with a provider: open-market used to check merely
// that the signer mapped to some player, and PermProviderOpen gates guild-market
// access without saying anything about spending.

// registerConsumerAddress adds another address to the teardown fixture's consumer
// with exactly the given address-level permissions.
func registerConsumerAddress(t *testing.T, f *teardownFixture, seed string, permissions types.Permission) string {
	t.Helper()

	acc := sdk.AccAddress(seed + "_padding_address_12345678901234567890")
	address := acc.String()
	f.k.SetPlayerIndexForAddress(f.ctx, address, f.consumer.Index)
	f.k.SetPermissionsByBytes(f.ctx, keeperlib.GetAddressPermissionIDBytes(address), permissions)

	return address
}

// requireDeniedOnSignerBit asserts the rejection came from the signing key lacking
// a specific bit, rather than from any other permission failure. The message text
// is identical for every Layer 1 denial, so match on the structured bit instead.
func requireDeniedOnSignerBit(t *testing.T, err error, signer string, permission types.Permission) {
	t.Helper()

	require.Error(t, err)

	var permErr *types.PermissionError
	require.ErrorAs(t, err, &permErr, "expected a structured PermissionError, got %v", err)
	require.Equal(t, uint64(permission), permErr.Permission,
		"rejected for the wrong permission bit")
	require.Equal(t, "address", permErr.CallerType,
		"the denial must be against the signing key, not the player's standing on an object")
	require.Equal(t, signer, permErr.CallerId)
}

func TestSignerIdentity_AgreementOpenRejectsWeakKey(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	weak := registerConsumerAddress(t, f, "agreeweak", types.PermPlay)

	const capacity, duration = 100, 50
	collateral := int64(capacity * duration * 10)
	f.fund(t, f.consumerAcc, collateral)

	_, err := f.ms.AgreementOpen(f.ctx, &types.MsgAgreementOpen{
		Creator:    weak,
		ProviderId: f.provider.Id,
		Capacity:   capacity,
		Duration:   duration,
	})
	requireDeniedOnSignerBit(t, err, weak, types.PermTokenTransfer)

	require.Equal(t, math.NewInt(collateral), f.balance(f.consumerAcc),
		"the rejected open must leave the primary balance alone")
	require.True(t, f.balance(f.collateralAcc).IsZero(),
		"no collateral should have reached the provider pool")
	require.Empty(t, f.k.GetAllAgreementIdByProviderIndex(f.ctx, f.provider.Id))
}

// The capability the gate has to preserve: a scoped key that does hold the spend
// bit may still open an agreement against its player's primary balance.
func TestSignerIdentity_AgreementOpenWithScopedKeyStillWorks(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	spender := registerConsumerAddress(t, f, "agreespend", types.PermPlay|types.PermAssetsAll)

	const capacity, duration = 100, 50
	collateral := int64(capacity * duration * 10)
	f.fund(t, f.consumerAcc, collateral)

	_, err := f.ms.AgreementOpen(f.ctx, &types.MsgAgreementOpen{
		Creator:    spender,
		ProviderId: f.provider.Id,
		Capacity:   capacity,
		Duration:   duration,
	})
	require.NoError(t, err, "a key holding PermTokenTransfer may open an agreement")

	require.True(t, f.balance(f.consumerAcc).IsZero())
	require.Equal(t, math.NewInt(collateral), f.balance(f.collateralAcc))
	require.Len(t, f.k.GetAllAgreementIdByProviderIndex(f.ctx, f.provider.Id), 1)
}

// PermProviderOpen is an access grant, not a spend one. A key that satisfies the
// guild-market gate still may not reach the primary address's balance without the
// token bit.
func TestSignerIdentity_AgreementOpenGuildMarketRequiresSpendBit(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	provider, found := f.k.GetProvider(f.ctx, f.provider.Id)
	require.True(t, found)
	provider.AccessPolicy = types.ProviderAccessPolicy_guildMarket
	_, err := f.k.SetProvider(f.ctx, provider)
	require.NoError(t, err)

	// Enough to pass the access gate at both layers: the bit on the signing key,
	// and standing for the player on the provider object.
	opener := registerConsumerAddress(t, f, "agreeguild", types.PermPlay|types.PermProviderOpen)
	f.k.SetPermissionsByBytes(f.ctx,
		keeperlib.GetObjectPermissionIDBytes(f.provider.Id, f.consumer.Id), types.PermProviderOpen)

	const capacity, duration = 100, 50
	collateral := int64(capacity * duration * 10)
	f.fund(t, f.consumerAcc, collateral)

	_, err = f.ms.AgreementOpen(f.ctx, &types.MsgAgreementOpen{
		Creator:    opener,
		ProviderId: f.provider.Id,
		Capacity:   capacity,
		Duration:   duration,
	})
	// PermTokenTransfer, not PermProviderOpen: the access gate was satisfied and
	// the spend gate is what stopped it.
	requireDeniedOnSignerBit(t, err, opener, types.PermTokenTransfer)

	require.Equal(t, math.NewInt(collateral), f.balance(f.consumerAcc))
	require.True(t, f.balance(f.collateralAcc).IsZero())
}

// Extending an agreement buys the extra blocks out of the primary address, so it
// is the same spend as opening one. PermUpdate decides who may modify the
// agreement and is not a substitute: the key below holds it, which is what makes
// this test isolate the spend gate rather than the update gate.
func TestSignerIdentity_AgreementDurationIncreaseRequiresSpendBit(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	agreement, _ := f.openAgreement(t, 100, 50)

	updater := registerConsumerAddress(t, f, "agreedur", types.PermPlay|types.PermUpdate)

	const increase = 10
	topUp := int64(100 * increase * 10)
	f.fund(t, f.consumerAcc, topUp)

	collateralBefore := f.balance(f.collateralAcc)

	_, err := f.ms.AgreementDurationIncrease(f.ctx, &types.MsgAgreementDurationIncrease{
		Creator:          updater,
		AgreementId:      agreement.Id,
		DurationIncrease: increase,
	})
	requireDeniedOnSignerBit(t, err, updater, types.PermTokenTransfer)

	require.Equal(t, math.NewInt(topUp), f.balance(f.consumerAcc),
		"the rejected top-up must leave the primary balance alone")
	require.Equal(t, collateralBefore, f.balance(f.collateralAcc))

	unchanged, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, found)
	require.Equal(t, agreement.EndBlock, unchanged.EndBlock,
		"the agreement must not have been extended")
}

func TestSignerIdentity_AgreementDurationIncreaseWithSpendBitSucceeds(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	agreement, _ := f.openAgreement(t, 100, 50)

	updater := registerConsumerAddress(t, f, "agreedurok",
		types.PermPlay|types.PermUpdate|types.PermTokenTransfer)

	const increase = 10
	topUp := int64(100 * increase * 10)
	f.fund(t, f.consumerAcc, topUp)

	collateralBefore := f.balance(f.collateralAcc)

	_, err := f.ms.AgreementDurationIncrease(f.ctx, &types.MsgAgreementDurationIncrease{
		Creator:          updater,
		AgreementId:      agreement.Id,
		DurationIncrease: increase,
	})
	require.NoError(t, err)

	require.True(t, f.balance(f.consumerAcc).IsZero())
	require.Equal(t, collateralBefore.AddRaw(topUp), f.balance(f.collateralAcc))

	extended, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, found)
	require.Equal(t, agreement.EndBlock+increase, extended.EndBlock)
}
