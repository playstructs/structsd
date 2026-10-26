package types_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

/* TestCalculateDifficultyGoldenVectors is the consensus vector table.
 *
 * Difficulty decides whether a proof-of-work transaction is accepted, so every
 * validator has to reach the same number for the same inputs. It used to be
 * derived from math.Log10, which Go implements in assembly on some
 * architectures and portable code on others; those disagree in the last bit,
 * and the truncation to an integer turned that into a whole leading zero.
 *
 * Fixed values rather than a re-derivation: a test that recomputed the formula
 * would agree with any future rewrite of it, including a wrong one. These are
 * the numbers the chain is committed to.
 */
func TestCalculateDifficultyGoldenVectors(t *testing.T) {
	cases := []struct {
		age    uint64
		irange uint64
		want   int
	}{
		// Degenerate inputs pin the requirement at its maximum.
		{age: 0, irange: 25, want: 64},
		{age: 1, irange: 25, want: 64},
		{age: 5, irange: 0, want: 64},
		{age: 5, irange: 1, want: 64},

		// A planet at its starting shield, across the curve.
		{age: 2, irange: 25, want: 51},
		{age: 5, irange: 25, want: 33},
		{age: 12, irange: 25, want: 16},
		{age: 24, irange: 25, want: 2},
		{age: 25, irange: 25, want: 1},

		// Past the range the requirement floors at one, and stays there.
		{age: 26, irange: 25, want: 1},
		{age: 100000, irange: 25, want: 1},

		// The exact powers, which is where the float version was wrong: these
		// are the points at which 63*log_range(age) is an exact integer, and
		// truncating a value a hair below it cost an extra leading zero.
		{age: 3, irange: 27, want: 43},
		{age: 5, irange: 125, want: 43},
		{age: 25, irange: 125, want: 22},
		{age: 3, irange: 2187, want: 55},
		{age: 243, irange: 2187, want: 19},
		{age: 25, irange: 15625, want: 43},
		{age: 625, irange: 15625, want: 22},
		{age: 5, irange: 1953125, want: 57},
		{age: 78125, irange: 1953125, want: 15},

		// A large shield, the kind an Orbital Shield Generator stack produces.
		{age: 2, irange: 1953125, want: 61},
		{age: 1000000, irange: 1953125, want: 4},
	}

	for _, c := range cases {
		require.Equal(t, c.want, types.CalculateDifficulty(c.age, c.irange),
			"age=%d range=%d", c.age, c.irange)
	}
}

/* TestCalculateDifficultyMatchesTheLogarithmIdentity checks the implementation
 * against the definition it is derived from, over the whole reachable space.
 *
 * The golden table above pins specific answers; this pins the rule that
 * produced them. Stated as the inequality rather than as a logarithm, because a
 * logarithm is the thing that could not be trusted here in the first place:
 * difficulty is 64 - e, where e is the largest exponent with range^e <= age^63.
 */
func TestCalculateDifficultyMatchesTheLogarithmIdentity(t *testing.T) {
	for _, r := range []uint64{2, 25, 27, 125, 2187, 3125, 15625, 1953125} {
		for age := uint64(2); age <= r+2 && age <= 30000; age++ {
			got := types.CalculateDifficulty(age, r)

			ageToThe63 := new(big.Int).Exp(new(big.Int).SetUint64(age), big.NewInt(63), nil)
			rangeBig := new(big.Int).SetUint64(r)

			expectedExponent := 0
			pow := big.NewInt(1)
			for k := 1; k <= 63; k++ {
				pow.Mul(pow, rangeBig)
				if pow.Cmp(ageToThe63) > 0 {
					break
				}
				expectedExponent = k
			}

			require.Equal(t, 64-expectedExponent, got, "age=%d range=%d", age, r)
			require.GreaterOrEqual(t, got, 1, "difficulty must never fall below one")
			require.LessOrEqual(t, got, 64, "difficulty must never exceed the hash length")
		}
	}
}

// The curve has to fall as work ages, or a miner's proof could expire while they
// are computing it.
func TestCalculateDifficultyIsMonotonicInAge(t *testing.T) {
	for _, r := range []uint64{2, 25, 125, 2187, 1953125} {
		previous := 65
		for age := uint64(1); age <= r*2 && age <= 20000; age++ {
			got := types.CalculateDifficulty(age, r)
			require.LessOrEqual(t, got, previous,
				"difficulty rose from %d to %d at age=%d range=%d", previous, got, age, r)
			previous = got
		}
	}
}
