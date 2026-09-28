package keeper_test

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

/* Retained legacy charter arithmetic/wire regression tests and guild transfer
 * authorization. v0.22.0 paid creation replaces proof and entitlement admission;
 * msg_server_guild_create_test.go covers the current handler and rejects legacy
 * consent fields explicitly.
 */

// charterFixture is one dev-difficulty chain with a bonded reactor and a player
// holding rights on it.
type charterFixture struct {
	t   *testing.T
	k   keeperlib.Keeper
	ms  types.MsgServer
	ctx sdk.Context

	mock    *keepertest.MockStakingKeeper
	valAddr sdk.ValAddress
	reactor types.Reactor
	player  types.Player
}

/* charterTestDifficultyRange keeps the puzzle solvable inside a unit test.
 *
 * The curve is 64 - log10(age)/log10(range)*63, so at range 4 an age of 4 lands
 * on one leading zero: one hash in sixteen, which a loop finds immediately. The
 * production range of 2,500,000 would need ten zeros and about three weeks.
 */
const charterTestDifficultyRange = uint64(4)

// charterTestReactorAge is short enough to step past with a couple of blocks.
const charterTestReactorAge = uint64(10)

func newCharterFixture(t *testing.T) *charterFixture {
	t.Helper()

	k, ms, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	require.NoError(t, k.SetParams(ctx, types.Params{
		GuildCharterDifficultyRange: charterTestDifficultyRange,
		GuildCharterReactorAge:      charterTestReactorAge,
	}))

	// Height 1 rather than 0, so that "the anchor is behind us" and "the anchor
	// is now" are distinguishable states.
	ctx = ctx.WithBlockHeight(1)
	k.SetGuildCharterAnchor(ctx, uint64(ctx.BlockHeight()))

	playerAcc := sdk.AccAddress("charterfounder1234567890123456789012")
	player := testAppendPlayer(k, ctx, types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	})

	valAddr := sdk.ValAddress(playerAcc.Bytes())
	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
	mock.AddValidator(valAddr, math.NewInt(1000))

	reactor := k.AppendReactor(ctx, types.Reactor{
		Validator:  valAddr.String(),
		RawAddress: valAddr.Bytes(),
		Owner:      player.Id,
	})

	testPermissionAdd(k, ctx, keeperlib.GetObjectPermissionIDBytes(reactor.Id, player.Id), types.PermReactorAll)

	return &charterFixture{
		t: t, k: k, ms: ms, ctx: ctx,
		mock: mock, valAddr: valAddr, reactor: reactor, player: player,
	}
}

// advanceTo moves the fixture to a height, which is how the difficulty curve is
// stepped: age is the distance from the anchor.
func (f *charterFixture) advanceTo(height int64) {
	f.ctx = f.ctx.WithBlockHeight(height)
}

func (f *charterFixture) anchor() uint64 {
	anchor, found := f.k.GetGuildCharterAnchor(f.ctx)
	require.True(f.t, found, "fixture should always have stamped an anchor")
	return anchor
}

/* solve grinds a nonce that satisfies the charter puzzle for a (solver, founder)
 * pair at the current height.
 *
 * Deliberately grinds rather than hard-coding a nonce: the preimage format is
 * part of what these tests are checking, and a fixture that pre-computed hashes
 * would have to be rewritten to match any change to it rather than failing.
 */
func (f *charterFixture) solve(solverPlayerId string, founderPlayerId string) (proof string, nonce string) {
	f.t.Helper()

	anchor := f.anchor()
	difficulty := f.k.CharterDifficulty(f.ctx)
	require.LessOrEqual(f.t, difficulty, 4,
		"difficulty %d is too high to grind in a unit test; check the fixture's height and range", difficulty)

	for attempt := 0; attempt < 5000000; attempt++ {
		candidate := strconv.Itoa(attempt)
		hash := types.HashBuild(types.GuildCharterWorkInput(f.ctx.ChainID(), solverPlayerId, founderPlayerId, anchor, candidate))

		valid, _ := types.HashBuildAndCheckDifficulty(
			types.GuildCharterWorkInput(f.ctx.ChainID(), solverPlayerId, founderPlayerId, anchor, candidate),
			hash, f.k.CharterAge(f.ctx), charterTestDifficultyRange)
		if valid {
			return hash, candidate
		}
	}

	f.t.Fatalf("no nonce found at difficulty %d", difficulty)
	return "", ""
}

// TestCharterDifficultyCurve pins the shape of the decay, including both ends.
//
// The floor and the age-zero case are the two the params guard exists for: a
// range below 2 divides by zero or by a negative, and either one silently pins
// the requirement at 64 zeros forever.
func TestCharterDifficultyCurve(t *testing.T) {
	const production = types.DefaultGuildCharterDifficultyRange

	require.Equal(t, 64, types.CalculateDifficulty(0, production),
		"a fresh anchor must be unsolvable, or a guild could be founded in the block after the last one")
	require.Equal(t, 64, types.CalculateDifficulty(1, production))

	// The advertised calibration: about ten zeros three weeks in at six second
	// blocks, bottoming out at one zero well beyond that. Asserted as a band
	// rather than a point because the curve floors, so the exact age at which a
	// level begins is not a round number of days.
	threeWeeks := uint64(21 * 24 * 60 * 60 / 6)
	threeWeekDifficulty := types.CalculateDifficulty(threeWeeks, production)
	require.GreaterOrEqual(t, threeWeekDifficulty, 10)
	require.LessOrEqual(t, threeWeekDifficulty, 11)

	require.Equal(t, 1, types.CalculateDifficulty(production, production),
		"the range is by definition the age at which the requirement bottoms out")
	require.Equal(t, 1, types.CalculateDifficulty(production*10, production),
		"past the range the requirement must clamp, not go negative")

	// Monotonic, which is what makes waiting a strategy at all.
	previous := 64
	for age := uint64(2); age < production; age = age * 3 / 2 {
		current := types.CalculateDifficulty(age, production)
		require.LessOrEqual(t, current, previous, "difficulty must never rise with age")
		previous = current
	}
}

// TestParamsValidateRejectsUnsolvableRange covers the two values that break the
// curve rather than merely making it hard.
func TestParamsValidateRejectsUnsolvableRange(t *testing.T) {
	for _, invalid := range []uint64{0, 1} {
		params := types.DefaultParams()
		params.GuildCharterDifficultyRange = invalid
		require.Error(t, params.Validate(), "range %d must be rejected", invalid)
	}

	require.NoError(t, types.DefaultParams().Validate())

	// A record written before the fields existed decodes as zero, which Validate
	// cannot reach retroactively, so the readers substitute rather than trust.
	stale := types.Params{}
	require.Equal(t, types.DefaultGuildCharterDifficultyRange, stale.CharterDifficultyRange())
	require.Equal(t, types.DefaultGuildCharterReactorAge, stale.CharterReactorAge())
}

// TestCharterProofFoundsGuild is the happy path: solve the puzzle, get a guild,
// and be credited for it.

func TestCharterWorkInputSeparatesPlayers(t *testing.T) {
	require.NotEqual(t,
		types.GuildCharterWorkInput("test-chain", "1-4", "21-7", 100, "0"),
		types.GuildCharterWorkInput("test-chain", "1-42", "1-7", 100, "0"),
		"a pair boundary that concatenation can slide across would let one proof serve two founders")
}

// TestCharterInputsBindChainId pins that both preimages are chain-scoped, so a
// proof mined or a consent signed on one chain cannot be replayed on a fork that
// shares the founder, reactor, substation and anchor.
func TestCharterInputsBindChainId(t *testing.T) {
	require.NotEqual(t,
		types.GuildCharterWorkInput("chain-a", "1-4", "1-7", 100, "0"),
		types.GuildCharterWorkInput("chain-b", "1-4", "1-7", 100, "0"),
		"a work proof must not solve on a different chain")

	require.NotEqual(t,
		types.GuildCharterConsentInput("chain-a", "1-7", "3-1", "9-1", "endpoint", 100),
		types.GuildCharterConsentInput("chain-b", "1-7", "3-1", "9-1", "endpoint", 100),
		"a founder consent must not verify on a different chain")
}

/* TestCharterThirdPartyFoundingWithConsent is the pool case, and the reason
 * consent exists at all.
 *
 * The founder never signs a transaction. They sign a consent string before the
 * grind starts, where latency is free, and the winner broadcasts the instant they
 * find a nonce — which is forced rather than stylistic, because a charter is a
 * race and routing the winning nonce back to the founder would let the pool with
 * the most attentive leader beat the pool with the most hashpower.
 */

func TestGuildTransferRevokesOutgoingOwner(t *testing.T) {
	f := newCharterFixture(t)
	vaults, vaultErr := f.k.NewCurrentContext(f.ctx).EnsureVaults(f.player.Id)
	require.NoError(t, vaultErr)
	fundAlpha(t, f.k, f.ctx, vaults[0], 4000000)
	f.advanceTo(int64(charterTestReactorAge) + 2)

	resp, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
		Creator:   f.player.Creator,
		ReactorId: f.reactor.Id,
		Endpoint:  "for-sale",
	})
	require.NoError(t, err)

	buyer, _ := f.newConsentingPlayer("charterbuyer")

	sellerPermissionId := keeperlib.GetObjectPermissionIDBytes(resp.GuildId, f.player.Id)
	require.True(t, testPermissionHasAll(f.k, f.ctx, sellerPermissionId, types.PermGuildAll))

	_, err = f.ms.GuildUpdateOwnerId(f.ctx, &types.MsgGuildUpdateOwnerId{
		Creator: f.player.Creator,
		GuildId: resp.GuildId,
		Owner:   buyer.Id,
	})
	require.NoError(t, err)

	require.False(t, testPermissionHasAll(f.k, f.ctx, sellerPermissionId, types.PermGuildTokenMint),
		"a seller who can still mint the guild token has not sold the guild")
	require.False(t, testPermissionHasAll(f.k, f.ctx, sellerPermissionId, types.PermGuildTokenBurn),
		"a seller who can still confiscate holders' balances has not sold the guild")
	require.False(t, testPermissionHasAll(f.k, f.ctx, sellerPermissionId, types.PermAdmin),
		"a seller who keeps admin can grant the guild back to themselves")

	require.True(t,
		testPermissionHasAll(f.k, f.ctx, keeperlib.GetObjectPermissionIDBytes(resp.GuildId, buyer.Id), types.PermGuildAll),
		"the buyer must end up with full control")

	// The composition with the membership guard: having sold, the seller is a
	// plain member and may found again.
	f.advanceTo(5)
	_, err = f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
		Creator:   f.player.Creator,
		ReactorId: f.reactor.Id,
		Endpoint:  "next-one",
	})
	require.NoError(t, err, "an owner who has transferred their guild is free to found another")
}

/* newConsentingPlayer registers a player whose address is derived from a real
 * secp256k1 key, which is what the consent path needs: it verifies a signature
 * against a pubkey that has to bech32 back to the address named.
 *
 * The address comes from types.PubKeyToBech32 rather than AccAddress.String()
 * because that function hardcodes the "structs" prefix and only the CLI
 * configures it, so under a keeper test's default "cosmos" prefix the derivation
 * check could never pass. On a real chain the two agree.
 */
func (f *charterFixture) newConsentingPlayer(seed string) (types.Player, *secp256k1.PrivKey) {
	f.t.Helper()

	key := secp256k1.GenPrivKeyFromSecret([]byte(seed))
	address := types.PubKeyToBech32(key.PubKey().Bytes())

	player := testAppendPlayer(f.k, f.ctx, types.Player{
		Creator:        address,
		PrimaryAddress: address,
	})

	return player, key
}

// signConsent fills in a message's consent fields with the founder's signature
// over the message as it currently stands.
func (f *charterFixture) signConsent(msg *types.MsgGuildCreate, founder types.Player, key *secp256k1.PrivKey) {
	f.t.Helper()
	f.signConsentAs(msg, founder.Id, founder, key)
}

// signConsentAs signs a consent naming one player with another player's key,
// which is how the mismatch cases are built.
func (f *charterFixture) signConsentAs(msg *types.MsgGuildCreate, founderPlayerId string, signer types.Player, key *secp256k1.PrivKey) {
	f.t.Helper()

	input := types.GuildCharterConsentInput(f.ctx.ChainID(), founderPlayerId, msg.ReactorId, msg.EntrySubstationId, msg.Endpoint, f.anchor())
	signature, err := key.Sign([]byte(input))
	require.NoError(f.t, err)

	msg.Address = signer.PrimaryAddress
	msg.ProofPubKey = hex.EncodeToString(key.PubKey().Bytes())
	msg.ProofSignature = hex.EncodeToString(signature)
}

// transferGuildAway hands the founder's current guild to a throwaway player, so
// that a following creation is refused by whatever is under test rather than by
// the membership guard.
func (f *charterFixture) transferGuildAway(msg *types.MsgGuildCreate) {
	f.t.Helper()

	sink, _ := f.newConsentingPlayer(fmt.Sprintf("chartersink%d", f.ctx.BlockHeight()))
	founder, found := f.k.GetPlayer(f.ctx, f.player.Id)
	require.True(f.t, found)

	_, err := f.ms.GuildUpdateOwnerId(f.ctx, &types.MsgGuildUpdateOwnerId{
		Creator: msg.Creator,
		GuildId: founder.GuildId,
		Owner:   sink.Id,
	})
	require.NoError(f.t, err)
}
