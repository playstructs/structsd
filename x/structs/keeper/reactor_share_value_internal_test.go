package keeper

import (
	"testing"

	"cosmossdk.io/math"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

/* Internal because the arithmetic is the thing under test and it is unexported.
 * Everything else in this package tests through the keeper; this one has to
 * reach the conversion itself, because the property that matters is about the
 * *sum* across delegations and no single handler call can show it.
 */

/* TestDelegationShareValueCannotOutgrowTheStake is the regression on a slash
 * being undone by sharding the stake.
 *
 * Fuel is written per delegation, and it used to be rounded per delegation.
 * LegacyDec.RoundInt is banker's rounding, so a shard worth exactly x.5 rounds
 * to x+1 - and after a 1% slash a 50-share delegation is worth exactly 49.5.
 * Split the stake finely enough and every shard rounds back to its pre-slash
 * value, so the total Fuel, and the grid capacity standing on it, survives a
 * slash the stake did not. Nothing sums Fuel against validator.Tokens, so
 * nothing would have noticed.
 *
 * These are the report's own numbers.
 */
func TestDelegationShareValueCannotOutgrowTheStake(t *testing.T) {
	const shards = 20_000
	const sharesEach = 50

	totalShares := math.LegacyNewDec(shards * sharesEach)
	tokensAfterSlash := math.LegacyNewDec(990_000) // a 1% slash of 1,000,000

	perShard := delegationShareValueAgainst(
		math.LegacyNewDec(sharesEach), totalShares, tokensAfterSlash)

	require.Equal(t, int64(49), perShard.Int64(),
		"a shard worth exactly 49.5 must not round back to its pre-slash 50")

	total := perShard.MulRaw(shards)
	require.True(t, total.LTE(tokensAfterSlash.TruncateInt()),
		"the sum of every shard must never exceed the stake behind it: got %s against %s",
		total, tokensAfterSlash.TruncateInt())
}

/* TestDelegationShareValueMatchesTheStakingKeeper pins the conversion to the
 * SDK's own, rather than to a second hand-written copy of it.
 *
 * Validator.TokensFromShares multiplies by the pool before dividing by the share
 * total; dividing first rounds the ratio to LegacyDec's 18 places before it
 * meets the pool, so the error scales with the pool. What a delegation is worth
 * is staking's answer to give, so this asserts agreement rather than restating
 * the formula.
 */
func TestDelegationShareValueMatchesTheStakingKeeper(t *testing.T) {
	pools := []int64{1, 7, 1_000, 999_983, 1_000_000, 4_100_000_000}
	splits := []int64{1, 2, 3, 7, 1_000, 20_000}

	for _, tokens := range pools {
		for _, split := range splits {
			if split > tokens {
				continue
			}

			validator := stakingtypes.Validator{
				Tokens:          math.NewInt(tokens),
				DelegatorShares: math.LegacyNewDec(tokens),
			}
			shares := math.LegacyNewDec(tokens / split)

			require.Equal(t,
				validator.TokensFromShares(shares).TruncateInt(),
				delegationShareValue(shares, validator),
				"tokens=%d split=%d", tokens, split)
		}
	}
}

// A slash reduces what every delegation is worth. Sharding must not change that,
// which is the property the per-shard rounding broke.
func TestDelegationShareValueFallsWithASlash(t *testing.T) {
	totalShares := math.LegacyNewDec(1_000_000)
	shares := math.LegacyNewDec(50)

	before := delegationShareValueAgainst(shares, totalShares, math.LegacyNewDec(1_000_000))
	after := delegationShareValueAgainst(shares, totalShares, math.LegacyNewDec(990_000))

	require.True(t, after.LT(before),
		"a slashed delegation must be worth less than an unslashed one: %s vs %s", after, before)
}

// The guards, which run inside staking hooks that cannot refuse.
func TestDelegationShareValueGuards(t *testing.T) {
	dec := math.LegacyNewDec(10)

	require.True(t, delegationShareValueAgainst(math.LegacyDec{}, dec, dec).IsZero(), "nil shares")
	require.True(t, delegationShareValueAgainst(dec, math.LegacyDec{}, dec).IsZero(), "nil share total")
	require.True(t, delegationShareValueAgainst(dec, dec, math.LegacyDec{}).IsZero(), "nil pool")
	require.True(t, delegationShareValueAgainst(dec, math.LegacyZeroDec(), dec).IsZero(),
		"a share-less validator backs no value, and Quo would panic")

	// Every caller writes this into a uint64, and math.Int.Uint64 panics rather
	// than wrapping.
	huge := math.LegacyNewDecFromInt(math.NewIntFromUint64(1 << 63).MulRaw(4))
	require.True(t, delegationShareValueAgainst(dec, dec, huge).IsZero(),
		"a value that cannot be a uint64 must answer with the one that mints nothing")
}
