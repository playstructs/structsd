package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// fundAlpha mints ualpha into the module and sends it to an account.
func fundAlpha(t *testing.T, k keeperlib.Keeper, ctx sdk.Context, acc sdk.AccAddress, amount int64) {
	t.Helper()
	coins := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(amount)))
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, coins))
}

func collateralBalance(k keeperlib.Keeper, ctx sdk.Context, guildId string) math.Int {
	pool := authtypes.NewModuleAddress(types.GuildBankCollateralPool + guildId)
	return k.BankKeeper().SpendableCoin(ctx, pool, "ualpha").Amount
}

func tokenBalance(k keeperlib.Keeper, ctx sdk.Context, acc sdk.AccAddress, guildId string) math.Int {
	return k.BankKeeper().SpendableCoin(ctx, acc, "uguild."+guildId).Amount
}

// TestMsgGuildBankConvert covers convert-in ratio math, the convert-in fee
// (kept in collateral, rounded up), the min-output slippage guard, and the
// zero-supply / zero-amount rejections.
func TestMsgGuildBankConvert(t *testing.T) {
	k, ms, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	gs := testCreateGuild(k, ctx)
	ownerAcc, _ := sdk.AccAddressFromBech32(gs.GuildOwner.Creator)

	// Bootstrap the bank at a 2:1 collateral:token ratio (1000 alpha / 500 token).
	fundAlpha(t, k, ctx, ownerAcc, 5000)
	_, err := ms.GuildBankMint(goCtx, &types.MsgGuildBankMint{Creator: gs.GuildOwner.Creator, AmountAlpha: 1000, AmountToken: 500})
	require.NoError(t, err)
	require.Equal(t, int64(1000), collateralBalance(k, ctx, gs.Guild.Id).Int64())

	// --- convert-in, no fee: tokensOut = floor(100 * 500 / 1000) = 50 ---
	tokenBefore := tokenBalance(k, ctx, ownerAcc, gs.Guild.Id)
	_, err = ms.GuildBankConvert(goCtx, &types.MsgGuildBankConvert{Creator: gs.GuildOwner.Creator, GuildId: gs.Guild.Id, AmountAlpha: 100})
	require.NoError(t, err)
	require.Equal(t, int64(50), tokenBalance(k, ctx, ownerAcc, gs.Guild.Id).Sub(tokenBefore).Int64(), "50 tokens minted at 2:1")
	require.Equal(t, int64(1100), collateralBalance(k, ctx, gs.Guild.Id).Int64(), "full alpha deposited")

	// --- convert-in with a 10% in-fee ---
	// pool now 1100 alpha / 550 token. feeAlpha = ceil(0.1*100) = 10;
	// netAlpha = 90; tokensOut = floor(90 * 550 / 1100) = 45. Full 100 -> pool.
	require.NoError(t, setBankConvertInFee(t, ms, ctx, gs, "0.1"))
	tokenBefore = tokenBalance(k, ctx, ownerAcc, gs.Guild.Id)
	collBefore := collateralBalance(k, ctx, gs.Guild.Id)
	_, err = ms.GuildBankConvert(goCtx, &types.MsgGuildBankConvert{Creator: gs.GuildOwner.Creator, GuildId: gs.Guild.Id, AmountAlpha: 100})
	require.NoError(t, err)
	require.Equal(t, int64(45), tokenBalance(k, ctx, ownerAcc, gs.Guild.Id).Sub(tokenBefore).Int64(), "fee reduces minted tokens")
	require.Equal(t, int64(100), collateralBalance(k, ctx, gs.Guild.Id).Sub(collBefore).Int64(), "fee stays in collateral")

	// --- slippage guard: demand more tokens than the ratio yields ---
	_, err = ms.GuildBankConvert(goCtx, &types.MsgGuildBankConvert{Creator: gs.GuildOwner.Creator, GuildId: gs.Guild.Id, AmountAlpha: 100, MinAmountToken: 1000})
	require.Error(t, err)
	require.Contains(t, err.Error(), "slippage")

	// --- zero amount rejected ---
	_, err = ms.GuildBankConvert(goCtx, &types.MsgGuildBankConvert{Creator: gs.GuildOwner.Creator, GuildId: gs.Guild.Id, AmountAlpha: 0})
	require.Error(t, err)

	// --- zero-supply bank rejects convert (ratio undefined) ---
	gs2 := testCreateGuild(k, ctx)
	owner2, _ := sdk.AccAddressFromBech32(gs2.GuildOwner.Creator)
	fundAlpha(t, k, ctx, owner2, 500)
	_, err = ms.GuildBankConvert(goCtx, &types.MsgGuildBankConvert{Creator: gs2.GuildOwner.Creator, GuildId: gs2.Guild.Id, AmountAlpha: 100})
	require.Error(t, err)
	require.Contains(t, err.Error(), "undefined")
}

// TestMsgGuildBankRedeemWithFee verifies the convert-out fee is retained in
// collateral and the minAmountAlpha guard reverts on slippage.
func TestMsgGuildBankRedeemWithFee(t *testing.T) {
	k, ms, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	gs := testCreateGuild(k, ctx)
	ownerAcc, _ := sdk.AccAddressFromBech32(gs.GuildOwner.Creator)

	fundAlpha(t, k, ctx, ownerAcc, 2000)
	_, err := ms.GuildBankMint(goCtx, &types.MsgGuildBankMint{Creator: gs.GuildOwner.Creator, AmountAlpha: 1000, AmountToken: 500})
	require.NoError(t, err)

	require.NoError(t, setBankConvertOutFee(t, ms, ctx, gs, "0.1"))

	// Redeem 50 tokens: grossAlpha = floor(50 * 1000 / 500) = 100;
	// feeAlpha = ceil(0.1 * 100) = 10; net paid = 90; 10 stays in collateral.
	alphaBefore := k.BankKeeper().SpendableCoin(ctx, ownerAcc, "ualpha").Amount
	_, err = ms.GuildBankRedeem(goCtx, &types.MsgGuildBankRedeem{Creator: gs.GuildOwner.Creator, AmountToken: sdk.NewCoin("uguild."+gs.Guild.Id, math.NewInt(50))})
	require.NoError(t, err)
	require.Equal(t, int64(90), k.BankKeeper().SpendableCoin(ctx, ownerAcc, "ualpha").Amount.Sub(alphaBefore).Int64(), "net of 10% out-fee")
	require.Equal(t, int64(910), collateralBalance(k, ctx, gs.Guild.Id).Int64(), "fee retained: 1000 - 90")

	// Slippage guard on redeem.
	_, err = ms.GuildBankRedeem(goCtx, &types.MsgGuildBankRedeem{Creator: gs.GuildOwner.Creator, AmountToken: sdk.NewCoin("uguild."+gs.Guild.Id, math.NewInt(10)), MinAmountAlpha: 1000})
	require.Error(t, err)
	require.Contains(t, err.Error(), "slippage")
}

// TestMsgGuildBankConvertToken exercises the cross-guild convert: both guilds
// collect their fee, the final output is guarded, and a same-guild convert is
// rejected.
func TestMsgGuildBankConvertToken(t *testing.T) {
	k, ms, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	gsA := testCreateGuild(k, ctx)
	ownerA, _ := sdk.AccAddressFromBech32(gsA.GuildOwner.Creator)
	gsB := testCreateGuild(k, ctx)
	ownerB, _ := sdk.AccAddressFromBech32(gsB.GuildOwner.Creator)

	// Bootstrap both banks 2:1.
	fundAlpha(t, k, ctx, ownerA, 2000)
	_, err := ms.GuildBankMint(goCtx, &types.MsgGuildBankMint{Creator: gsA.GuildOwner.Creator, AmountAlpha: 1000, AmountToken: 500})
	require.NoError(t, err)
	fundAlpha(t, k, ctx, ownerB, 2000)
	_, err = ms.GuildBankMint(goCtx, &types.MsgGuildBankMint{Creator: gsB.GuildOwner.Creator, AmountAlpha: 1000, AmountToken: 500})
	require.NoError(t, err)

	require.NoError(t, setBankConvertOutFee(t, ms, ctx, gsA, "0.1"))
	require.NoError(t, setBankConvertInFee(t, ms, ctx, gsB, "0.1"))

	// Same-guild convert rejected.
	_, err = ms.GuildBankConvertToken(goCtx, &types.MsgGuildBankConvertToken{Creator: gsA.GuildOwner.Creator, AmountToken: sdk.NewCoin("uguild."+gsA.Guild.Id, math.NewInt(50)), GuildId: gsA.Guild.Id})
	require.Error(t, err)
	require.Contains(t, err.Error(), "same_guild")

	// ownerA converts 50 guildA token -> guildB token.
	// Leg 1 (redeem A): gross = floor(50*1000/500)=100; A out-fee 10 -> bridge = 90.
	// Leg 2 (convert B): B in-fee = ceil(0.1*90)=9; net=81; out = floor(81*500/1000)=40.
	collAbefore := collateralBalance(k, ctx, gsA.Guild.Id)
	collBbefore := collateralBalance(k, ctx, gsB.Guild.Id)
	bTokBefore := tokenBalance(k, ctx, ownerA, gsB.Guild.Id)
	_, err = ms.GuildBankConvertToken(goCtx, &types.MsgGuildBankConvertToken{Creator: gsA.GuildOwner.Creator, AmountToken: sdk.NewCoin("uguild."+gsA.Guild.Id, math.NewInt(50)), GuildId: gsB.Guild.Id})
	require.NoError(t, err)
	require.Equal(t, int64(40), tokenBalance(k, ctx, ownerA, gsB.Guild.Id).Sub(bTokBefore).Int64(), "final guildB token output")
	// Guild A collateral: -100 (gross out) +10 (fee retained) = -90.
	require.Equal(t, int64(-90), collateralBalance(k, ctx, gsA.Guild.Id).Sub(collAbefore).Int64(), "A pays gross less retained fee")
	// Guild B collateral: +90 bridge alpha (fee included).
	require.Equal(t, int64(90), collateralBalance(k, ctx, gsB.Guild.Id).Sub(collBbefore).Int64(), "B gains full bridge alpha")
	_ = ownerB
}

// TestGuildBankFeeValidation checks the fee range guard and PermAdmin gating.
func TestGuildBankFeeValidation(t *testing.T) {
	k, ms, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	gs := testCreateGuild(k, ctx)

	// Out of range (> 1.0) rejected.
	require.Error(t, setBankConvertInFee(t, ms, ctx, gs, "1.5"))
	require.Error(t, setBankConvertOutFee(t, ms, ctx, gs, "1.5"))

	// Boundaries accepted.
	require.NoError(t, setBankConvertInFee(t, ms, ctx, gs, "0"))
	require.NoError(t, setBankConvertInFee(t, ms, ctx, gs, "1"))

	// A player without PermAdmin on the guild cannot change fees.
	strangerAcc := sdk.AccAddress("stranger012345678901234567890123")
	stranger := testAppendPlayer(k, ctx, types.Player{Creator: strangerAcc.String(), PrimaryAddress: strangerAcc.String()})
	_, err := ms.GuildUpdateBankConvertInFee(ctx, &types.MsgGuildUpdateBankConvertInFee{Creator: stranger.Creator, GuildId: gs.Guild.Id, BankConvertInFee: math.LegacyMustNewDecFromStr("0.2")})
	require.Error(t, err)
}

// TestGuildBankRoundingInvariant asserts the pool-favored invariant against
// adversarial values: a round-trip convert-in then redeem never returns more
// alpha than was put in, even with a donation-inflated ratio.
func TestGuildBankRoundingInvariant(t *testing.T) {
	k, ms, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	gs := testCreateGuild(k, ctx)
	ownerAcc, _ := sdk.AccAddressFromBech32(gs.GuildOwner.Creator)

	// Tiny supply, then donate a large collateral to make one token expensive.
	fundAlpha(t, k, ctx, ownerAcc, 5_000_000)
	_, err := ms.GuildBankMint(goCtx, &types.MsgGuildBankMint{Creator: gs.GuildOwner.Creator, AmountAlpha: 3, AmountToken: 7})
	require.NoError(t, err)
	// Donate directly to the collateral pool (permissionless, the attack vector).
	donation := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(1_000_000)))
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, donation))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, authtypes.NewModuleAddress(types.GuildBankCollateralPool+gs.Guild.Id), donation))

	for _, amt := range []uint64{1, 2, 3, 7, 13, 100, 999} {
		alphaBefore := k.BankKeeper().SpendableCoin(ctx, ownerAcc, "ualpha").Amount
		tokBefore := tokenBalance(k, ctx, ownerAcc, gs.Guild.Id)

		_, cErr := ms.GuildBankConvert(goCtx, &types.MsgGuildBankConvert{Creator: gs.GuildOwner.Creator, GuildId: gs.Guild.Id, AmountAlpha: amt})
		if cErr != nil {
			// conversion_too_small is a legitimate rejection, never a loss.
			require.Contains(t, cErr.Error(), "conversion_too_small")
			continue
		}
		minted := tokenBalance(k, ctx, ownerAcc, gs.Guild.Id).Sub(tokBefore)

		if minted.IsPositive() {
			_, rErr := ms.GuildBankRedeem(goCtx, &types.MsgGuildBankRedeem{Creator: gs.GuildOwner.Creator, AmountToken: sdk.NewCoin("uguild."+gs.Guild.Id, minted)})
			require.NoError(t, rErr)
		}
		alphaAfter := k.BankKeeper().SpendableCoin(ctx, ownerAcc, "ualpha").Amount
		require.True(t, alphaAfter.LTE(alphaBefore), "round-trip must never profit (amt=%d): before=%s after=%s", amt, alphaBefore, alphaAfter)
	}
}

// TestGuildBankFeeNilNormalization verifies a guild imported with nil (pre-v0.21.0)
// fee decs is normalized to zero on commit rather than panicking on marshal.
func TestGuildBankFeeNilNormalization(t *testing.T) {
	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	cc := k.NewCurrentContext(ctx)
	// A Guild value with nil LegacyDec fee fields, as an old export would decode.
	require.True(t, math.LegacyDec{}.IsNil())
	cc.GenesisImportGuild(types.Guild{Id: "4-777", BankConvertInFee: math.LegacyDec{}, BankConvertOutFee: math.LegacyDec{}})
	require.NotPanics(t, func() { cc.CommitAll() })

	loaded, found := k.GetGuild(ctx, "4-777")
	require.True(t, found)
	require.False(t, loaded.BankConvertInFee.IsNil())
	require.False(t, loaded.BankConvertOutFee.IsNil())
	require.True(t, loaded.BankConvertInFee.IsZero())
}

func setBankConvertInFee(t *testing.T, ms types.MsgServer, ctx sdk.Context, gs testGuildSetup, fee string) error {
	t.Helper()
	_, err := ms.GuildUpdateBankConvertInFee(ctx, &types.MsgGuildUpdateBankConvertInFee{Creator: gs.GuildOwner.Creator, GuildId: gs.Guild.Id, BankConvertInFee: math.LegacyMustNewDecFromStr(fee)})
	return err
}

func setBankConvertOutFee(t *testing.T, ms types.MsgServer, ctx sdk.Context, gs testGuildSetup, fee string) error {
	t.Helper()
	_, err := ms.GuildUpdateBankConvertOutFee(ctx, &types.MsgGuildUpdateBankConvertOutFee{Creator: gs.GuildOwner.Creator, GuildId: gs.Guild.Id, BankConvertOutFee: math.LegacyMustNewDecFromStr(fee)})
	return err
}
