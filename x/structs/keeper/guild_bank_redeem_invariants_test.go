package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Guild bank redemption pays floor(amountToken * collateral / supply) in pure
// math.Int. Before v0.21.0 it divided first in LegacyDec, which rounds the ratio
// to 18 places before multiplying by the collateral balance: at supply 1.5e18
// and collateral 1.5e19 one token paid 15 ualpha where its pro-rata share is 10.
// A holder could split a redemption into favourably sized pieces and take more
// than they owned, out of everybody else's collateral.
//
// The Int form is exact, but exact per call is not the same as unsplittable
// across calls, which is what these tests measure. Every redemption floors,
// discarding a remainder, yet it also leaves the pool backing each remaining
// token with more alpha than before — so each later chunk prices better than the
// first, and the two effects run against each other by under a unit per chunk.
//
// The invariant that holds unconditionally is the one that matters: a holder
// never extracts more than their pro-rata share of the pool as it stood when
// they started, however they slice the redemption. See
// TestInvariant_RedemptionNeverExceedsProRataShare, which derives from the
// per-call ratio check in redeemSequence.
//
// "Splitting never pays more than one call" is a strictly weaker claim and is
// only true at a zero convert-out fee. The fee stays in the collateral pool, so
// a holder who splits recaptures the share of their own fee that backs the
// tokens they have not redeemed yet. That is a property of where the fee goes,
// not a rounding flaw, and it is bounded by the pro-rata invariant above.
// TestGuildBankSplitRecapturesFeeButNotPrincipal pins both halves.

// guildBankRedeemDenom is the token minted for the guild under test.
func guildBankRedeemDenom(guildId string) string { return "uguild." + guildId }

// snapshotBankState returns a function that rewinds the bank to this moment.
//
// It is the bank half of branching, and it is needed because sdk.Context's
// CacheContext only branches KV state: the mock bank keeper holds balances and
// supply in Go maps and ignores the ctx it is handed, so a discarded cache
// branch still leaves its mints behind. Without this rewind the second scenario
// in a test reads the first one's pool, which is exactly how the numbers below
// were first wrong.
func snapshotBankState(t require.TestingT, k keeperlib.Keeper) (rewind func()) {
	mock, ok := k.BankKeeper().(*keepertest.MockBankKeeper)
	require.True(t, ok, "these tests need the map-backed mock bank keeper to snapshot")

	snapshot := mock.Snapshot()
	return func() { mock.Restore(snapshot) }
}

// seedGuildBank puts an exact (supply, collateral, out-fee) state into ctx.
//
// It mints through the bank keeper rather than going via BankMint so the ratio
// can be set to any pair the property wants, including ones no sequence of mints
// would reach. The whole supply goes to one player, which is what lets the
// redeemed amount range over all of it.
func seedGuildBank(t require.TestingT, k keeperlib.Keeper, ctx sdk.Context, guildId string, player sdk.AccAddress, supply uint64, collateral uint64, outFee string) {
	tokens := sdk.NewCoins(sdk.NewCoin(guildBankRedeemDenom(guildId), math.NewIntFromUint64(supply)))
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, tokens))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, player, tokens))

	alpha := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewIntFromUint64(collateral)))
	pool := authtypes.NewModuleAddress(types.GuildBankCollateralPool + guildId)
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, alpha))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, pool, alpha))

	cc := k.NewCurrentContext(ctx)
	guild := cc.GetGuild(guildId)
	require.True(t, guild.LoadGuild())
	require.NoError(t, guild.SetBankConvertOutFee(math.LegacyMustNewDecFromStr(outFee)))
	cc.CommitAll()
}

// redeemSequence redeems each amount in order and returns the total net alpha
// paid out, checking the per-redemption invariants as it goes.
//
// minAmountAlpha is zero, the value BankRedeem reserves for an internally
// composed leg. The public message demands a positive floor, which would turn a
// chunk small enough to floor to nothing into an error rather than a zero
// payout; that guard is covered by TestGuildBankSafetyBoundaries and would only
// obscure the arithmetic here.
//
// A refusal ends the sequence instead of failing the test. Splitting can drain
// collateral to zero, and BankRedeem then rejects the next chunk with
// bank_ratio — a shorter sequence only lowers the total, so it cannot hide a
// violation of the properties being measured.
func redeemSequence(t require.TestingT, k keeperlib.Keeper, ctx sdk.Context, guildId string, creator string, amounts []math.Int) math.Int {
	denom := guildBankRedeemDenom(guildId)
	paid := math.ZeroInt()

	for i, amount := range amounts {
		collateralBefore := collateralBalance(k, ctx, guildId)
		supplyBefore := k.BankKeeper().GetSupply(ctx, denom).Amount

		cc := k.NewCurrentContext(ctx)
		player, err := cc.GetSigningPlayer(creator)
		require.NoError(t, err)

		guild := cc.GetGuild(guildId)
		require.True(t, guild.LoadGuild())

		netAlpha, redeemErr := guild.BankRedeem(amount, math.ZeroInt(), player)
		if redeemErr != nil {
			break
		}
		cc.CommitAll()

		collateralAfter := collateralBalance(k, ctx, guildId)
		supplyAfter := k.BankKeeper().GetSupply(ctx, denom).Amount

		require.True(t, netAlpha.LTE(collateralBefore),
			"chunk %d paid %s out of a pool holding %s", i, netAlpha, collateralBefore)

		// The pool must never end up backing each remaining token with less
		// alpha than before. This is the load-bearing check: the pro-rata bound
		// on the whole sequence follows from it, because collateral paid out is
		// exactly what the ratio holds back. Cross-multiplied so the check does
		// not reintroduce the decimal division the fix removed.
		if supplyAfter.IsPositive() {
			require.True(t, collateralAfter.Mul(supplyBefore).GTE(collateralBefore.Mul(supplyAfter)),
				"chunk %d diluted the remaining holders: %s/%s became %s/%s",
				i, collateralBefore, supplyBefore, collateralAfter, supplyAfter)
		}

		paid = paid.Add(netAlpha)
	}

	return paid
}

// drawPartition splits total into positive chunks summing to exactly total.
func drawPartition(rt *rapid.T, total uint64) []math.Int {
	count := rapid.Uint64Range(2, 6).Draw(rt, "chunkCount")
	if count > total {
		count = total
	}

	amounts := make([]math.Int, 0, count)
	remaining := total

	for i := uint64(0); i < count-1; i++ {
		// Leave at least one unit for each chunk still to come.
		chunk := rapid.Uint64Range(1, remaining-(count-1-i)).Draw(rt, fmt.Sprintf("chunk%d", i))
		amounts = append(amounts, math.NewIntFromUint64(chunk))
		remaining -= chunk
	}

	return append(amounts, math.NewIntFromUint64(remaining))
}

// bankRedeemFixture builds the guild once. Every case branches off ctx with
// CacheContext and discards the write function, so ctx stays pristine and cases
// cannot see each other.
func bankRedeemFixture(t *testing.T) (keeperlib.Keeper, sdk.Context, testGuildSetup, sdk.AccAddress) {
	t.Helper()

	k, _, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	gs := testCreateGuild(k, ctx)
	playerAcc, err := sdk.AccAddressFromBech32(gs.GuildOwner.Creator)
	require.NoError(t, err)

	return k, ctx, gs, playerAcc
}

// guildBankMaxBalance bounds both pool sides. BankRedeem refuses a supply or
// collateral it cannot represent in uint64, and the bound covers the magnitude
// the report used (1.5e18 supply against 1.5e19 collateral).
const guildBankMaxBalance = uint64(10_000_000_000_000_000_000)

// drawMagnitude draws a uint64 in [1, max] spread across orders of magnitude
// instead of uniformly.
//
// This is what makes the properties sensitive. The rounding the old formula got
// wrong only appears when the collateral-to-supply ratio needs more than 18
// significant digits, which means one side being small while the other is
// large. A uniform uint64 draw lands within a bit or two of the maximum almost
// every time and explores none of those pairs — with a flat generator the
// pro-rata property missed the reintroduced bug for the whole default budget.
func drawMagnitude(rt *rapid.T, label string, max uint64) uint64 {
	bits := rapid.IntRange(1, 64).Draw(rt, label+"Bits")

	upper := max
	if bits < 64 {
		if bound := uint64(1)<<bits - 1; bound < upper {
			upper = bound
		}
	}

	return rapid.Uint64Range(1, upper).Draw(rt, label)
}

// drawBankState draws a redeemable (supply, collateral, amount) triple.
//
// The three are drawn differently on purpose, to aim at the region where an
// inexact ratio actually costs a whole unit of alpha. An 18-place ratio is off
// by up to 5e-19, so the error only reaches one ualpha once collateral passes
// ~2e18: collateral is drawn flat, which puts it above that most of the time.
// Supply and the redeemed amount instead want to span magnitudes, since what
// makes the ratio inexact is the amount being small against the supply.
func drawBankState(rt *rapid.T) (supply uint64, collateral uint64, total uint64) {
	supply = drawMagnitude(rt, "supply", guildBankMaxBalance)
	collateral = rapid.Uint64Range(1, guildBankMaxBalance).Draw(rt, "collateral")
	total = drawMagnitude(rt, "amountToken", supply)
	return
}

// TestInvariant_RedemptionNeverExceedsProRataShare is the anti-extraction
// property the report asked for, and it holds at any fee: whatever chunk sizes a
// holder picks, the alpha they walk away with never exceeds
// floor(amountToken * collateral / supply) measured against the pool as it stood
// before they began. That is the entitlement the old LegacyDec formula could
// exceed by half again.
//
// Run with -rapid.checks for deeper exploration.
func TestInvariant_RedemptionNeverExceedsProRataShare(t *testing.T) {
	k, ctx, gs, playerAcc := bankRedeemFixture(t)

	rapid.Check(t, func(rt *rapid.T) {
		supply, collateral, total := drawBankState(rt)
		outFee := rapid.SampledFrom([]string{"0", "0.001", "0.05", "0.25"}).Draw(rt, "outFee")
		amounts := drawPartition(rt, total)

		defer snapshotBankState(rt, k)()
		branch, _ := ctx.CacheContext()
		seedGuildBank(rt, k, branch, gs.Guild.Id, playerAcc, supply, collateral, outFee)

		entitlement := math.NewIntFromUint64(total).
			Mul(math.NewIntFromUint64(collateral)).
			Quo(math.NewIntFromUint64(supply))

		paid := redeemSequence(rt, k, branch, gs.Guild.Id, gs.GuildOwner.Creator, amounts)

		require.True(rt, paid.LTE(entitlement),
			"redeeming %d tokens in %d chunks paid %s, above the pro-rata share %s (supply=%d collateral=%d fee=%s)",
			total, len(amounts), paid, entitlement, supply, collateral, outFee)
	})
}

// TestInvariant_SplitRedemptionNeverBeatsBulk is the stronger claim, and it
// needs the zero-fee precondition. Both legs run against byte-identical state,
// branched twice off the same seeded cache context, so the only difference
// between them is the slicing.
//
// With a fee this is false by design rather than by accident — see
// TestGuildBankSplitRecapturesFeeButNotPrincipal.
func TestInvariant_SplitRedemptionNeverBeatsBulk(t *testing.T) {
	k, ctx, gs, playerAcc := bankRedeemFixture(t)

	rapid.Check(t, func(rt *rapid.T) {
		supply, collateral, total := drawBankState(rt)
		amounts := drawPartition(rt, total)

		defer snapshotBankState(rt, k)()
		base, _ := ctx.CacheContext()
		seedGuildBank(rt, k, base, gs.Guild.Id, playerAcc, supply, collateral, "0")

		// Both legs must start from the same state, which for the bank means
		// rewinding it between them rather than branching it twice.
		rewindToSeeded := snapshotBankState(rt, k)

		bulkCtx, _ := base.CacheContext()
		bulk := redeemSequence(rt, k, bulkCtx, gs.Guild.Id, gs.GuildOwner.Creator,
			[]math.Int{math.NewIntFromUint64(total)})

		rewindToSeeded()
		splitCtx, _ := base.CacheContext()
		split := redeemSequence(rt, k, splitCtx, gs.Guild.Id, gs.GuildOwner.Creator, amounts)

		require.True(rt, split.LTE(bulk),
			"splitting %d tokens into %d chunks paid %s where one call paid %s (supply=%d collateral=%d)",
			total, len(amounts), split, bulk, supply, collateral)
	})
}

// TestGuildBankSplitRecapturesFeeButNotPrincipal documents the one way splitting
// pays better, so that nobody later reads it as the rounding bug returning.
//
// The convert-out fee is retained in the collateral pool, which raises the alpha
// backing every token still outstanding — including the redeemer's own. Slicing
// a redemption therefore refunds part of the fee on the earlier slices. It is a
// consequence of where the fee goes, and it stays under the fee-free pro-rata
// share, so it cannot reach another holder's principal.
//
// These are the numbers rapid produced when TestInvariant_SplitRedemptionNeverBeatsBulk
// was first written without the zero-fee precondition.
func TestGuildBankSplitRecapturesFeeButNotPrincipal(t *testing.T) {
	k, ctx, gs, playerAcc := bankRedeemFixture(t)

	const (
		supply     = uint64(61)
		collateral = uint64(18_949)
		total      = uint64(11)
	)

	redeem := func(outFee string, amounts ...uint64) math.Int {
		defer snapshotBankState(t, k)()
		branch, _ := ctx.CacheContext()
		seedGuildBank(t, k, branch, gs.Guild.Id, playerAcc, supply, collateral, outFee)

		chunks := make([]math.Int, 0, len(amounts))
		for _, amount := range amounts {
			chunks = append(chunks, math.NewIntFromUint64(amount))
		}
		return redeemSequence(t, k, branch, gs.Guild.Id, gs.GuildOwner.Creator, chunks)
	}

	bulkWithFee := redeem("0.05", total)
	splitWithFee := redeem("0.05", 8, 1, 1, 1)
	bulkNoFee := redeem("0", total)
	splitNoFee := redeem("0", 8, 1, 1, 1)

	require.Equal(t, "3246", bulkWithFee.String())
	require.Equal(t, "3251", splitWithFee.String(), "splitting recaptures part of the redeemer's own fee")
	require.True(t, splitWithFee.GT(bulkWithFee), "the recapture is what makes the zero-fee precondition necessary")

	// The recapture is bounded: it can only claw back fee, never principal.
	entitlement := math.NewIntFromUint64(total).
		Mul(math.NewIntFromUint64(collateral)).
		Quo(math.NewIntFromUint64(supply))
	require.Equal(t, "3417", entitlement.String())
	require.True(t, splitWithFee.LT(entitlement), "a splitter must still not reach the fee-free pro-rata share")

	// Remove the fee and the ordinary direction is restored.
	require.Equal(t, "3417", bulkNoFee.String(), "with no fee a single call pays exactly the pro-rata share")
	require.True(t, splitNoFee.LTE(bulkNoFee), "without a fee to recapture, splitting only loses to truncation")
}

// TestGuildBankRedeemUsesExactFloorRatio pins the reported numbers directly.
//
// supply 1.5e18 against collateral 1.5e19 makes one token worth exactly 10
// ualpha. The divide-first LegacyDec form rounded 6.66e-19 up to one 10^-18 unit
// before multiplying and paid 15 — half again the holder's actual share.
func TestGuildBankRedeemUsesExactFloorRatio(t *testing.T) {
	k, ctx, gs, playerAcc := bankRedeemFixture(t)

	const (
		supply     = uint64(1_500_000_000_000_000_000)
		collateral = uint64(15_000_000_000_000_000_000)
	)

	defer snapshotBankState(t, k)()
	branch, _ := ctx.CacheContext()
	seedGuildBank(t, k, branch, gs.Guild.Id, playerAcc, supply, collateral, "0")

	paid := redeemSequence(t, k, branch, gs.Guild.Id, gs.GuildOwner.Creator, []math.Int{math.OneInt()})
	require.Equal(t, "10", paid.String(),
		"one token against %d supply / %d collateral is worth floor(10), not the 15 the LegacyDec form paid", supply, collateral)
}
