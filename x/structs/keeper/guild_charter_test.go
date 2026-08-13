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

/* Guild charter regression suite.
 *
 * Guild creation used to be a reactor permission with no membership check at
 * all, which let one permissioned player accumulate guilds while their Player
 * record pointed only at the newest. It is now a chain-global proof-of-work with
 * a reactor entitlement beside it, and most of what follows is about the seams
 * that arrangement introduces: the anchor that makes a solution single-use, the
 * split between the signer who solved and the founder who owns, and the
 * consent signature that binds the two together.
 *
 * The mock staking keeper fires no hooks, so nothing here proves anything about
 * when the SDK calls us. What it can prove is bonded and jailed status, which the
 * free path turns on; app/guild_charter_test.go covers the same path against real
 * staking.
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
		hash := types.HashBuild(types.GuildCharterWorkInput(solverPlayerId, founderPlayerId, anchor, candidate))

		valid, _ := types.HashBuildAndCheckDifficulty(
			types.GuildCharterWorkInput(solverPlayerId, founderPlayerId, anchor, candidate),
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
	threeWeeks := float64(21 * 24 * 60 * 60 / 6)
	threeWeekDifficulty := types.CalculateDifficulty(threeWeeks, production)
	require.GreaterOrEqual(t, threeWeekDifficulty, 10)
	require.LessOrEqual(t, threeWeekDifficulty, 11)

	require.Equal(t, 1, types.CalculateDifficulty(float64(production), production),
		"the range is by definition the age at which the requirement bottoms out")
	require.Equal(t, 1, types.CalculateDifficulty(float64(production)*10, production),
		"past the range the requirement must clamp, not go negative")

	// Monotonic, which is what makes waiting a strategy at all.
	previous := 64
	for age := float64(2); age < float64(production); age *= 1.5 {
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
func TestCharterProofFoundsGuild(t *testing.T) {
	f := newCharterFixture(t)
	f.advanceTo(5)

	proof, nonce := f.solve(f.player.Id, f.player.Id)

	resp, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
		Creator:   f.player.Creator,
		ReactorId: f.reactor.Id,
		Endpoint:  "charter-endpoint",
		Proof:     proof,
		Nonce:     nonce,
	})
	require.NoError(t, err)

	guild, found := f.k.GetGuild(f.ctx, resp.GuildId)
	require.True(t, found)
	require.Equal(t, f.player.Id, guild.Owner)
	require.Equal(t, f.player.Id, guild.CharterSolverId,
		"a proof-founded guild must record who solved it, or credit is lost the moment it is sold")

	player, playerFound := f.k.GetPlayer(f.ctx, f.player.Id)
	require.True(t, playerFound)
	require.Equal(t, resp.GuildId, player.GuildId)
	require.Equal(t, uint64(1), player.GuildRank)
}

/* TestCharterProofResetsAnchor is the replay defence, which is the anchor and
 * nothing else.
 *
 * There is no spent-proof set and no nonce counter. Founding a guild moves the
 * anchor to the current height, which invalidates the proof just spent, every
 * other nonce anyone was grinding, and any consent signed against the old value
 * — all at once, and permanently, because heights never repeat.
 */
func TestCharterProofResetsAnchor(t *testing.T) {
	f := newCharterFixture(t)
	f.advanceTo(5)

	proof, nonce := f.solve(f.player.Id, f.player.Id)
	msg := &types.MsgGuildCreate{
		Creator:   f.player.Creator,
		ReactorId: f.reactor.Id,
		Endpoint:  "first",
		Proof:     proof,
		Nonce:     nonce,
	}

	_, err := f.ms.GuildCreate(f.ctx, msg)
	require.NoError(t, err)
	require.Equal(t, uint64(5), f.anchor(), "founding must stamp the anchor at the founding height")

	// The same proof again, in the next block. The difficulty is back at 64 and
	// the preimage no longer even names the current anchor.
	f.advanceTo(6)
	_, err = f.ms.GuildCreate(f.ctx, msg)
	require.Error(t, err, "a spent proof must not found a second guild")

	// And nothing in between resurrects it: the age has to climb all the way
	// back up before anyone can win again.
	require.Equal(t, 64, f.k.CharterDifficulty(f.ctx.WithBlockHeight(6)))
}

/* TestCharterProofBindsSolverAndFounder is why the preimage names both players.
 *
 * The four pre-existing proofs bind no player, because their reward is
 * object-bound: a build proof lifted from the mempool merely finishes somebody
 * else's struct. A charter's reward accrues to a named owner, so a proof that
 * bound neither end would let a mempool observer re-address the guild to
 * themselves, and one that bound only the founder would let a pool member's
 * solution be submitted under another solver's name.
 */
func TestCharterProofBindsSolverAndFounder(t *testing.T) {
	f := newCharterFixture(t)
	f.advanceTo(5)

	otherAcc := sdk.AccAddress("chartersolvertwo12345678901234567890")
	other := testAppendPlayer(f.k, f.ctx, types.Player{
		Creator:        otherAcc.String(),
		PrimaryAddress: otherAcc.String(),
	})

	// A proof mined for a different founder, replayed by the solver for
	// themselves: the front-run case.
	proof, nonce := f.solve(f.player.Id, other.Id)
	_, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
		Creator:   f.player.Creator,
		ReactorId: f.reactor.Id,
		Endpoint:  "stolen-founder",
		Proof:     proof,
		Nonce:     nonce,
	})
	require.Error(t, err, "a proof mined for another founder must not found the solver's own guild")

	// A proof mined by a different solver, submitted by someone else: the
	// credit-theft case.
	proof, nonce = f.solve(other.Id, f.player.Id)
	_, err = f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
		Creator:   f.player.Creator,
		ReactorId: f.reactor.Id,
		Endpoint:  "stolen-credit",
		Proof:     proof,
		Nonce:     nonce,
	})
	require.Error(t, err, "a proof mined by another solver must not be submittable as one's own")
}

// TestCharterWorkInputSeparatesPlayers covers the separator in the preimage,
// which is load-bearing rather than decorative: both sides are player ids with
// the same type prefix, so bare concatenation would let two different pairs share
// one solution.
func TestCharterWorkInputSeparatesPlayers(t *testing.T) {
	require.NotEqual(t,
		types.GuildCharterWorkInput("1-4", "21-7", 100, "0"),
		types.GuildCharterWorkInput("1-42", "1-7", 100, "0"),
		"a pair boundary that concatenation can slide across would let one proof serve two founders")
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
func TestCharterThirdPartyFoundingWithConsent(t *testing.T) {
	f := newCharterFixture(t)
	f.advanceTo(5)

	solver, _ := f.newConsentingPlayer("chartersolver")
	founder, founderKey := f.newConsentingPlayer("charterfounderb")

	proof, nonce := f.solve(solver.Id, founder.Id)
	msg := &types.MsgGuildCreate{
		Creator:         solver.Creator,
		ReactorId:       f.reactor.Id,
		Endpoint:        "pool-endpoint",
		FounderPlayerId: founder.Id,
		Proof:           proof,
		Nonce:           nonce,
	}
	f.signConsent(msg, founder, founderKey)

	resp, err := f.ms.GuildCreate(f.ctx, msg)
	require.NoError(t, err)

	guild, found := f.k.GetGuild(f.ctx, resp.GuildId)
	require.True(t, found)
	require.Equal(t, founder.Id, guild.Owner, "the guild belongs to the founder")
	require.Equal(t, founder.Creator, guild.Creator)
	require.Equal(t, solver.Id, guild.CharterSolverId, "the solver keeps the credit")

	storedFounder, _ := f.k.GetPlayer(f.ctx, founder.Id)
	require.Equal(t, resp.GuildId, storedFounder.GuildId)
	require.Equal(t, uint64(1), storedFounder.GuildRank)

	// The solver's own standing is untouched, which is what lets a pool helper
	// grind for somebody else without giving up their own guild.
	storedSolver, _ := f.k.GetPlayer(f.ctx, solver.Id)
	require.Empty(t, storedSolver.GuildId, "solving for somebody else must not move the solver")
	require.Zero(t, storedSolver.GuildRank)
	require.False(t,
		testPermissionHasAll(f.k, f.ctx, keeperlib.GetObjectPermissionIDBytes(resp.GuildId, solver.Id), types.PermGuildAll),
		"the solver must not gain rights over the guild they were paid to found")
}

// TestCharterConsentNegatives walks one case per binding in the consent
// preimage. The substation case is the important one: consent is a bearer token
// shared with a whole pool, and an unbound one would let any holder name their
// own substation as the guild's entry point, which is every future member's
// power supply.
func TestCharterConsentNegatives(t *testing.T) {
	newCase := func(t *testing.T) (*charterFixture, types.Player, *types.MsgGuildCreate, types.Player, *secp256k1.PrivKey) {
		f := newCharterFixture(t)
		f.advanceTo(5)

		solver, _ := f.newConsentingPlayer("chartersolver")
		founder, founderKey := f.newConsentingPlayer("charterfounderb")

		proof, nonce := f.solve(solver.Id, founder.Id)
		msg := &types.MsgGuildCreate{
			Creator:         solver.Creator,
			ReactorId:       f.reactor.Id,
			Endpoint:        "pool-endpoint",
			FounderPlayerId: founder.Id,
			Proof:           proof,
			Nonce:           nonce,
		}
		return f, solver, msg, founder, founderKey
	}

	t.Run("no consent at all", func(t *testing.T) {
		f, _, msg, _, _ := newCase(t)
		_, err := f.ms.GuildCreate(f.ctx, msg)
		require.Error(t, err, "founding for another player must require their consent")
	})

	t.Run("consent from a different player", func(t *testing.T) {
		f, _, msg, founder, _ := newCase(t)
		stranger, strangerKey := f.newConsentingPlayer("charterstranger")

		f.signConsentAs(msg, founder.Id, stranger, strangerKey)
		_, err := f.ms.GuildCreate(f.ctx, msg)
		require.Error(t, err, "a signature from somebody else's address must not stand for the founder")
	})

	t.Run("founder address lacking PermPlay", func(t *testing.T) {
		f, _, msg, founder, founderKey := newCase(t)
		f.signConsent(msg, founder, founderKey)

		testPermissionRemove(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(founder.PrimaryAddress), types.PermPlay)

		_, err := f.ms.GuildCreate(f.ctx, msg)
		require.Error(t, err, "a narrowly scoped key must not be able to sign a guild away")
	})

	t.Run("pubkey not deriving to the named address", func(t *testing.T) {
		f, _, msg, founder, founderKey := newCase(t)
		f.signConsent(msg, founder, founderKey)

		_, otherKey := f.newConsentingPlayer("charterotherkey")
		msg.ProofPubKey = hex.EncodeToString(otherKey.PubKey().Bytes())

		_, err := f.ms.GuildCreate(f.ctx, msg)
		require.Error(t, err)
	})

	t.Run("consent bound to a different substation", func(t *testing.T) {
		f, _, msg, founder, founderKey := newCase(t)

		// Signed for no substation, submitted naming one: the retargeting
		// attack, where the holder of a bearer consent points the new guild's
		// entry substation at themselves.
		f.signConsent(msg, founder, founderKey)

		substation, _, err := testAppendSubstation(f.k, f.ctx, types.Allocation{}, founder)
		require.NoError(t, err)
		msg.EntrySubstationId = substation.Id

		_, err = f.ms.GuildCreate(f.ctx, msg)
		require.Error(t, err, "consent must bind the entry substation, or a pool member can redirect it")
	})

	t.Run("consent bound to a different endpoint", func(t *testing.T) {
		f, _, msg, founder, founderKey := newCase(t)
		f.signConsent(msg, founder, founderKey)
		msg.Endpoint = "somewhere-else"

		_, err := f.ms.GuildCreate(f.ctx, msg)
		require.Error(t, err)
	})

	t.Run("consent bound to a different reactor", func(t *testing.T) {
		f, _, msg, founder, founderKey := newCase(t)
		f.signConsent(msg, founder, founderKey)

		otherReactor := f.k.AppendReactor(f.ctx, types.Reactor{
			Validator:  sdk.ValAddress("othervalidator123456").String(),
			RawAddress: sdk.ValAddress("othervalidator123456").Bytes(),
		})
		msg.ReactorId = otherReactor.Id

		_, err := f.ms.GuildCreate(f.ctx, msg)
		require.Error(t, err)
	})

	t.Run("consent bound to a stale anchor", func(t *testing.T) {
		f, _, msg, founder, founderKey := newCase(t)
		f.signConsent(msg, founder, founderKey)

		// Somebody else founded a guild, so the anchor moved. This is what makes
		// a consent single-use without any nonce of its own.
		f.k.SetGuildCharterAnchor(f.ctx, 4)

		_, err := f.ms.GuildCreate(f.ctx, msg)
		require.Error(t, err, "a consent signed against a spent anchor must not be replayable")
	})

	t.Run("unregistered founder", func(t *testing.T) {
		f, _, msg, _, _ := newCase(t)
		msg.FounderPlayerId = "1-9999"

		_, err := f.ms.GuildCreate(f.ctx, msg)
		require.Error(t, err, "a message-supplied player id must be resolved, not allocated")
	})
}

/* TestCharterConsentRequiresProof is the condition under which a consent is
 * single-use.
 *
 * Nothing about a consent is consumed on use. What retires it is the anchor
 * moving, and only the proof path moves the anchor — so accepting one on the
 * entitlement path would leave the signature live until somebody unrelated
 * proof-founded, which is long enough for the solver to mine a fresh proof and
 * replay it. The payload is griefing rather than theft: it drags the founder out
 * of whatever guild they had since joined and into a second one at rank 1.
 *
 * The refusal costs nothing, which is the point: the entitlement path already
 * demands reactor permission from the founder, so a founder who can reach it can
 * sign the transaction themselves, and unlike the proof path there is no race to
 * sign ahead of.
 */
func TestCharterConsentRequiresProof(t *testing.T) {
	newCase := func(t *testing.T) (*charterFixture, *types.MsgGuildCreate, types.Player, types.Player, *secp256k1.PrivKey) {
		f := newCharterFixture(t)
		f.advanceTo(int64(charterTestReactorAge) + 2)

		solver, _ := f.newConsentingPlayer("chartersolver")
		founder, founderKey := f.newConsentingPlayer("charterfounderb")

		// Reactor permission for the founder, so the entitlement path is
		// genuinely open to them and the consent rule is what refuses.
		testPermissionAdd(f.k, f.ctx, keeperlib.GetObjectPermissionIDBytes(f.reactor.Id, founder.Id), types.PermReactorAll)

		msg := &types.MsgGuildCreate{
			Creator:         solver.Creator,
			ReactorId:       f.reactor.Id,
			Endpoint:        "pool-endpoint",
			FounderPlayerId: founder.Id,
		}
		f.signConsent(msg, founder, founderKey)

		return f, msg, solver, founder, founderKey
	}

	t.Run("refused on the entitlement path", func(t *testing.T) {
		f, msg, _, _, _ := newCase(t)

		_, err := f.ms.GuildCreate(f.ctx, msg)
		require.Error(t, err, "a path that does not move the anchor must not spend a consent")
		require.Contains(t, err.Error(), "consent_needs_proof")

		reactor, _ := f.k.GetReactor(f.ctx, f.reactor.Id)
		require.Empty(t, reactor.GuildId, "a refused founding must not spend the entitlement either")
	})

	t.Run("the same consent still works with a proof", func(t *testing.T) {
		f, msg, solver, _, _ := newCase(t)

		msg.Proof, msg.Nonce = f.solve(solver.Id, msg.FounderPlayerId)

		_, err := f.ms.GuildCreate(f.ctx, msg)
		require.NoError(t, err, "refusing consent without a proof must not break the pool case it exists for")
	})
}

/* TestCharterProofPathRequiresLiveValidator is parity with
 * GuildUpdatePrimaryReactor, which is the handler that repairs exactly this
 * mistake after the fact.
 *
 * AppendGuild writes PrimaryReactorId unconditionally and GuildMembershipJoin
 * redelegates every joiner's infusion to whatever it names, so a guild founded on
 * a tombstoned validator is a trap for its members from the first block. The
 * entitlement path already demanded bonded and unjailed; the proof path demanded
 * nothing about the reactor at all.
 *
 * Refusing here cannot strand a mined solution, which is what makes the check
 * safe rather than merely correct: the work preimage binds solver, founder and
 * anchor and says nothing about the reactor, so a solver refused here names a
 * different reactor and re-submits the same nonce.
 */
func TestCharterProofPathRequiresLiveValidator(t *testing.T) {
	t.Run("refused when the validator is jailed", func(t *testing.T) {
		f := newCharterFixture(t)
		f.advanceTo(5)
		f.mock.JailValidator(f.valAddr)

		proof, nonce := f.solve(f.player.Id, f.player.Id)
		_, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
			Creator:   f.player.Creator,
			ReactorId: f.reactor.Id,
			Endpoint:  "jailed-primary",
			Proof:     proof,
			Nonce:     nonce,
		})
		require.Error(t, err, "a guild must not be born pointing at a jailed validator")
		require.Contains(t, err.Error(), "validator_jailed")

		require.Equal(t, uint64(1), f.anchor(),
			"a refused founding must leave everyone else's in-flight mining alone")
	})

	t.Run("refused when the validator is gone", func(t *testing.T) {
		f := newCharterFixture(t)
		f.advanceTo(5)

		proof, nonce := f.solve(f.player.Id, f.player.Id)
		f.mock.RemoveValidator(f.valAddr)

		_, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
			Creator:   f.player.Creator,
			ReactorId: f.reactor.Id,
			Endpoint:  "tombstoned-primary",
			Proof:     proof,
			Nonce:     nonce,
		})
		require.Error(t, err)
	})

	t.Run("an unbonded but unjailed validator is accepted", func(t *testing.T) {
		f := newCharterFixture(t)
		f.advanceTo(5)

		// Deliberately not a bonded check, matching GuildUpdatePrimaryReactor's
		// rationale: a guild may legitimately form around a validator still
		// working its way into the active set. Bonded is only required on the
		// entitlement path, where reactor health is what is being rewarded.
		f.mock.JailValidator(f.valAddr)
		f.mock.UnjailValidator(f.valAddr)

		proof, nonce := f.solve(f.player.Id, f.player.Id)
		_, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
			Creator:   f.player.Creator,
			ReactorId: f.reactor.Id,
			Endpoint:  "waiting-for-the-set",
			Proof:     proof,
			Nonce:     nonce,
		})
		require.NoError(t, err, "the proof path must not require a bond the entitlement path is paid for")
	})
}

/* TestCharterOwnedGuildNeedNotBeJoined pins a product rule that reads like a bug.
 *
 * Guilds are property. SetOwner moves guild.Owner and the PermGuildAll row and
 * touches neither player's GuildId, so a transfer inherently produces an owner
 * who is not a member — and a buyer who was guildless can then found a second
 * guild and hold both. That is not the unbounded loop the original report
 * described: each extra guild costs a real incoming transfer, and creation is
 * still gated on the puzzle.
 *
 * This exists to fail loudly if somebody "fixes" it, because the cure is worse
 * than the disease: forcing a buyer into the guild they bought would either evict
 * them from the one they are in or make a purchase impossible while they are in
 * one.
 */
func TestCharterOwnedGuildNeedNotBeJoined(t *testing.T) {
	f := newCharterFixture(t)
	f.advanceTo(int64(charterTestReactorAge) + 2)

	seller, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
		Creator:   f.player.Creator,
		ReactorId: f.reactor.Id,
		Endpoint:  "for-sale",
	})
	require.NoError(t, err)

	buyer, _ := f.newConsentingPlayer("charterbuyer")
	require.Empty(t, buyer.GuildId, "the buyer starts guildless, which is the case in question")

	_, err = f.ms.GuildUpdateOwnerId(f.ctx, &types.MsgGuildUpdateOwnerId{
		Creator: f.player.Creator,
		GuildId: seller.GuildId,
		Owner:   buyer.Id,
	})
	require.NoError(t, err)

	stored, _ := f.k.GetPlayer(f.ctx, buyer.Id)
	require.Empty(t, stored.GuildId, "buying a guild does not join it")

	// And founding is not blocked by owning one they are not in, because the
	// membership guard asks what they would be leaving.
	f.advanceTo(5)
	proof, nonce := f.solve(buyer.Id, buyer.Id)
	second, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
		Creator:   buyer.Creator,
		ReactorId: f.reactor.Id,
		Endpoint:  "mine-too",
		Proof:     proof,
		Nonce:     nonce,
	})
	require.NoError(t, err)

	purchased, _ := f.k.GetGuild(f.ctx, seller.GuildId)
	founded, _ := f.k.GetGuild(f.ctx, second.GuildId)
	require.Equal(t, buyer.Id, purchased.Owner)
	require.Equal(t, buyer.Id, founded.Owner)

	stored, _ = f.k.GetPlayer(f.ctx, buyer.Id)
	require.Equal(t, second.GuildId, stored.GuildId,
		"membership is singular even though ownership is not")
}

/* TestCharterSubstationRightsFollowTheFounder proves the subject of the one
 * pre-existing check that had to move.
 *
 * Every authorization in this handler belongs to the founder now, and the entry
 * substation check is the one that used to belong to the signer. It has to move:
 * a third-party solver holds no rights on the founder's substation and never
 * will, and what vouches for the substation named is the founder's consent, which
 * covers it.
 */
func TestCharterSubstationRightsFollowTheFounder(t *testing.T) {
	f := newCharterFixture(t)
	f.advanceTo(5)

	solver, _ := f.newConsentingPlayer("chartersolver")
	founder, founderKey := f.newConsentingPlayer("charterfounderb")

	// A substation the solver controls and the founder does not.
	solverSubstation, _, err := testAppendSubstation(f.k, f.ctx, types.Allocation{}, solver)
	require.NoError(t, err)

	proof, nonce := f.solve(solver.Id, founder.Id)
	msg := &types.MsgGuildCreate{
		Creator:           solver.Creator,
		ReactorId:         f.reactor.Id,
		Endpoint:          "pool-endpoint",
		EntrySubstationId: solverSubstation.Id,
		FounderPlayerId:   founder.Id,
		Proof:             proof,
		Nonce:             nonce,
	}
	f.signConsent(msg, founder, founderKey)

	_, err = f.ms.GuildCreate(f.ctx, msg)
	require.Error(t, err,
		"the solver's own rights on a substation must not satisfy a check that belongs to the founder")

	// The same message with a substation the founder controls succeeds, which is
	// what shows the check moved rather than merely tightened.
	founderSubstation, _, err := testAppendSubstation(f.k, f.ctx, types.Allocation{}, founder)
	require.NoError(t, err)
	msg.EntrySubstationId = founderSubstation.Id
	f.signConsent(msg, founder, founderKey)

	_, err = f.ms.GuildCreate(f.ctx, msg)
	require.NoError(t, err)
}

/* TestCharterReactorPath covers the free path and each of its four conditions.
 *
 * The bonded requirement is the load-bearing one. ReactorInitialize runs from
 * AfterValidatorCreated, so a reactor exists for every validator ever created,
 * and MsgCreateValidator chooses its own min_self_delegation — without a bonded
 * check a hundred throwaway validators would be a hundred free guilds.
 */
func TestCharterReactorPath(t *testing.T) {
	t.Run("succeeds without a proof and leaves the anchor alone", func(t *testing.T) {
		f := newCharterFixture(t)
		f.advanceTo(int64(charterTestReactorAge) + 2)
		anchorBefore := f.anchor()

		resp, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
			Creator:   f.player.Creator,
			ReactorId: f.reactor.Id,
			Endpoint:  "reactor-endpoint",
		})
		require.NoError(t, err)

		guild, found := f.k.GetGuild(f.ctx, resp.GuildId)
		require.True(t, found)
		require.Empty(t, guild.CharterSolverId,
			"nobody solved anything, so the two provenances stay distinguishable")

		require.Equal(t, anchorBefore, f.anchor(),
			"a validator collecting a perk must not wipe every pool's in-flight mining")

		reactor, _ := f.k.GetReactor(f.ctx, f.reactor.Id)
		require.Equal(t, resp.GuildId, reactor.GuildId, "success must spend the entitlement")
	})

	t.Run("is single-use per reactor", func(t *testing.T) {
		f := newCharterFixture(t)
		f.advanceTo(int64(charterTestReactorAge) + 2)

		msg := &types.MsgGuildCreate{
			Creator:   f.player.Creator,
			ReactorId: f.reactor.Id,
			Endpoint:  "reactor-endpoint",
		}
		_, err := f.ms.GuildCreate(f.ctx, msg)
		require.NoError(t, err)

		// Hand the first guild away so the membership guard is not what refuses
		// the second attempt, and the entitlement is.
		f.transferGuildAway(msg)

		_, err = f.ms.GuildCreate(f.ctx, msg)
		require.Error(t, err, "a reactor gets one free guild, not one per attempt")
	})

	t.Run("refused before the eligibility height", func(t *testing.T) {
		f := newCharterFixture(t)
		f.advanceTo(2)

		_, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
			Creator:   f.player.Creator,
			ReactorId: f.reactor.Id,
			Endpoint:  "too-soon",
		})
		require.Error(t, err)
	})

	t.Run("refused when the validator is jailed", func(t *testing.T) {
		f := newCharterFixture(t)
		f.advanceTo(int64(charterTestReactorAge) + 2)
		f.mock.JailValidator(f.valAddr)

		_, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
			Creator:   f.player.Creator,
			ReactorId: f.reactor.Id,
			Endpoint:  "jailed",
		})
		require.Error(t, err)
	})

	t.Run("refused when the validator is unbonded", func(t *testing.T) {
		f := newCharterFixture(t)
		f.advanceTo(int64(charterTestReactorAge) + 2)

		// Unjailed but out of the active set, which is the cheap-identity case:
		// a validator that never bonds costs almost nothing to create.
		f.mock.JailValidator(f.valAddr)
		f.mock.UnjailValidator(f.valAddr)

		_, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
			Creator:   f.player.Creator,
			ReactorId: f.reactor.Id,
			Endpoint:  "unbonded",
		})
		require.Error(t, err)
	})

	t.Run("refused without reactor permission", func(t *testing.T) {
		f := newCharterFixture(t)
		f.advanceTo(int64(charterTestReactorAge) + 2)

		outsiderAcc := sdk.AccAddress("charteroutsider12345678901234567890")
		outsider := testAppendPlayer(f.k, f.ctx, types.Player{
			Creator:        outsiderAcc.String(),
			PrimaryAddress: outsiderAcc.String(),
		})

		_, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
			Creator:   outsider.Creator,
			ReactorId: f.reactor.Id,
			Endpoint:  "not-mine",
		})
		require.Error(t, err, "the free path is a reward for running the reactor, so it stays permissioned")
	})
}

/* TestCharterProofDoesNotStealAReactorEntitlement is the hole that opened when
 * PermReactorGuildCreate stopped being the creation gate.
 *
 * Any player may now found a guild and may name any reactor as its primary. If
 * that also bound the reactor's GuildId, an outsider could burn a validator's
 * free-guild entitlement — and make their guild that reactor's official one — for
 * the price of one proof. The permission survives as the gate on the binding for
 * exactly this reason.
 */
func TestCharterProofDoesNotStealAReactorEntitlement(t *testing.T) {
	f := newCharterFixture(t)
	f.advanceTo(5)

	outsiderAcc := sdk.AccAddress("charteroutsider12345678901234567890")
	outsider := testAppendPlayer(f.k, f.ctx, types.Player{
		Creator:        outsiderAcc.String(),
		PrimaryAddress: outsiderAcc.String(),
	})

	proof, nonce := f.solve(outsider.Id, outsider.Id)
	_, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
		Creator:   outsider.Creator,
		ReactorId: f.reactor.Id,
		Endpoint:  "squatter",
		Proof:     proof,
		Nonce:     nonce,
	})
	require.NoError(t, err, "anyone who solves the puzzle may found a guild on any reactor")

	reactor, _ := f.k.GetReactor(f.ctx, f.reactor.Id)
	require.Empty(t, reactor.GuildId,
		"an outsider's guild must not consume the reactor owner's free-guild entitlement")

	// And the owner's entitlement is still there to spend.
	f.advanceTo(int64(charterTestReactorAge) + 2)
	_, err = f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
		Creator:   f.player.Creator,
		ReactorId: f.reactor.Id,
		Endpoint:  "still-mine",
	})
	require.NoError(t, err)
}

/* TestCharterMembershipGuard is the original report: GuildCreate checked nothing
 * about the caller's existing guild, so one permissioned player could accumulate
 * guilds while their Player record pointed only at the newest.
 *
 * An owner is refused rather than migrated, because leaving a guild you own would
 * strand it with an owner who is not a member — the exact state the old handler
 * produced on every repeat call.
 */
func TestCharterMembershipGuard(t *testing.T) {
	t.Run("an owner is refused", func(t *testing.T) {
		f := newCharterFixture(t)
		f.advanceTo(int64(charterTestReactorAge) + 2)

		msg := &types.MsgGuildCreate{
			Creator:   f.player.Creator,
			ReactorId: f.reactor.Id,
			Endpoint:  "first",
		}
		first, err := f.ms.GuildCreate(f.ctx, msg)
		require.NoError(t, err)

		// The two-messages-in-one-transaction case from the report, which as far
		// as this handler is concerned is just a second call.
		f.advanceTo(int64(charterTestReactorAge) + 3)
		proof, nonce := f.solve(f.player.Id, f.player.Id)
		_, err = f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
			Creator:   f.player.Creator,
			ReactorId: f.reactor.Id,
			Endpoint:  "second",
			Proof:     proof,
			Nonce:     nonce,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "is the owner of guild")

		player, _ := f.k.GetPlayer(f.ctx, f.player.Id)
		require.Equal(t, first.GuildId, player.GuildId,
			"a refused creation must leave the founder in the guild they own")
	})

	t.Run("a plain member leaves", func(t *testing.T) {
		f := newCharterFixture(t)
		f.advanceTo(int64(charterTestReactorAge) + 2)

		// A guild owned by somebody else, whose entry substation the joiner is
		// connected to.
		ownerAcc := sdk.AccAddress("charterotherowner1234567890123456789")
		owner := testAppendPlayer(f.k, f.ctx, types.Player{
			Creator:        ownerAcc.String(),
			PrimaryAddress: ownerAcc.String(),
		})
		entrySubstation, _, err := testAppendSubstation(f.k, f.ctx, types.Allocation{}, owner)
		require.NoError(t, err)
		oldGuild := f.k.AppendGuild(f.ctx, "old", entrySubstation.Id, f.reactor, owner, "")

		joiner, joinerKey := f.newConsentingPlayer("charterjoiner")
		joiner.GuildId = oldGuild.Id
		joiner.GuildRank = types.DefaultEntryRank
		joiner.SubstationId = entrySubstation.Id
		f.k.SetPlayer(f.ctx, joiner)
		_ = joinerKey

		f.advanceTo(5)
		proof, nonce := f.solve(joiner.Id, joiner.Id)
		resp, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
			Creator:   joiner.Creator,
			ReactorId: f.reactor.Id,
			Endpoint:  "mine-now",
			Proof:     proof,
			Nonce:     nonce,
		})
		require.NoError(t, err)

		stored, _ := f.k.GetPlayer(f.ctx, joiner.Id)
		require.Equal(t, resp.GuildId, stored.GuildId, "founding a guild leaves the one you were in")
		require.Equal(t, uint64(1), stored.GuildRank)
		require.Empty(t, stored.SubstationId,
			"the old guild's entry substation connection must drop, since it was that guild's")
	})

	t.Run("a personal substation is left alone", func(t *testing.T) {
		f := newCharterFixture(t)
		f.advanceTo(5)

		ownerAcc := sdk.AccAddress("charterotherowner1234567890123456789")
		owner := testAppendPlayer(f.k, f.ctx, types.Player{
			Creator:        ownerAcc.String(),
			PrimaryAddress: ownerAcc.String(),
		})
		entrySubstation, _, err := testAppendSubstation(f.k, f.ctx, types.Allocation{}, owner)
		require.NoError(t, err)
		oldGuild := f.k.AppendGuild(f.ctx, "old", entrySubstation.Id, f.reactor, owner, "")

		joiner, _ := f.newConsentingPlayer("charterjoiner")
		ownSubstation, _, err := testAppendSubstation(f.k, f.ctx, types.Allocation{}, joiner)
		require.NoError(t, err)

		joiner.GuildId = oldGuild.Id
		joiner.SubstationId = ownSubstation.Id
		f.k.SetPlayer(f.ctx, joiner)

		proof, nonce := f.solve(joiner.Id, joiner.Id)
		_, err = f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
			Creator:   joiner.Creator,
			ReactorId: f.reactor.Id,
			Endpoint:  "mine-now",
			Proof:     proof,
			Nonce:     nonce,
		})
		require.NoError(t, err)

		stored, _ := f.k.GetPlayer(f.ctx, joiner.Id)
		require.Equal(t, ownSubstation.Id, stored.SubstationId,
			"a substation that is not the old guild's entry point is likely the player's own")
	})
}

/* TestGuildTransferRevokesOutgoingOwner is the other half of making guilds
 * priceable.
 *
 * Transferability is now the pricing mechanism — there is no alpha fee, so the
 * secondary market is what says what a charter is worth — and a transfer that
 * left the seller's permission row in place transferred nothing: PermGuildAll
 * carries token mint, token burn and admin, so a seller could mint the currency,
 * confiscate holders' balances and grant the guild back to themselves.
 */
func TestGuildTransferRevokesOutgoingOwner(t *testing.T) {
	f := newCharterFixture(t)
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
	proof, nonce := f.solve(f.player.Id, f.player.Id)
	_, err = f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
		Creator:   f.player.Creator,
		ReactorId: f.reactor.Id,
		Endpoint:  "next-one",
		Proof:     proof,
		Nonce:     nonce,
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

	input := types.GuildCharterConsentInput(founderPlayerId, msg.ReactorId, msg.EntrySubstationId, msg.Endpoint, f.anchor())
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
