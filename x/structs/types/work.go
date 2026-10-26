package types

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
)

func HashCheck(input string, hash string) bool {
	newHash := sha256.New()
	newHash.Write([]byte(input))
	newHashOutput := hex.EncodeToString(newHash.Sum(nil))

	return (newHashOutput == hash)
}

func HashBuild(input string) string {
	newHash := sha256.New()
	newHash.Write([]byte(input))
	newHashOutput := hex.EncodeToString(newHash.Sum(nil))

	return newHashOutput
}

func HashBuildAndCheckDifficulty(input string, proof string, age uint64, difficultyRange uint64) (bool, uint64) {
	hash := HashBuild(input)

	if proof != hash {
		return false, 0
	}

	difficulty := CalculateDifficulty(age, difficultyRange)

	// Count leading zeros in the hash, up to the required difficulty
	achievedDifficulty := uint64(0)
	for position := 1; position <= difficulty; position++ {
		if hash[position-1:position] != "0" {
			return false, achievedDifficulty
		}
		achievedDifficulty++
	}

	return true, achievedDifficulty
}

/* CalculateDifficulty returns the number of leading hexadecimal zeroes a proof
 * must carry: 64 at the start, falling towards 1 as the work ages, and reaching
 * 1 once the age passes the difficulty range.
 *
 * The curve is logarithmic, but it is computed in integers, because this decides
 * whether a transaction is accepted and every validator has to reach the same
 * answer. It used to be
 *
 *     64 - int(math.Log10(age)/math.Log10(range)*63)
 *
 * and math.Log10 is not the same function everywhere: Go ships an assembly
 * implementation on some architectures and the portable one elsewhere, and they
 * disagree in the last bit. int() truncation turns a last-bit disagreement into
 * a whole leading zero, so two validators could require different proofs for the
 * same block and the same planet - one accepting a hash the other rejects, which
 * is an app hash split rather than a wrong answer. The current agreement between
 * amd64 and arm64 is a coincidence of Go's implementation, not a guarantee: a Go
 * release or a new architecture backend could break it with no change here.
 *
 * The integer form drops out of the logarithm identity. Writing e for the
 * exponent being truncated,
 *
 *     e = floor(63 * log(age) / log(range))
 *
 * and for range > 1 that is exactly
 *
 *     e = max{ k : range^k <= age^63 }
 *
 * so the answer is found by multiplying rather than by taking a logarithm at
 * all. k never needs to exceed 63, because difficulty is clamped at 1, which
 * also bounds the work and the size of the numbers involved.
 */
func CalculateDifficulty(activationAge uint64, difficultyRange uint64) int {
	if activationAge <= 1 {
		return 64
	}

	// log(1) is zero and log(0) is negative infinity, so neither has a
	// meaningful curve; both pin the requirement at its maximum. The charter
	// range is kept above this by Params.Validate, but struct-type difficulties
	// and planetary shields are not governed by that, so it is answered here
	// rather than assumed.
	if difficultyRange <= 1 {
		return 64
	}

	ageToThe63 := new(big.Int).Exp(new(big.Int).SetUint64(activationAge), big.NewInt(63), nil)
	rangeBig := new(big.Int).SetUint64(difficultyRange)

	rangeToTheK := big.NewInt(1)
	exponent := 0

	for k := 1; k <= 63; k++ {
		rangeToTheK.Mul(rangeToTheK, rangeBig)
		if rangeToTheK.Cmp(ageToThe63) > 0 {
			break
		}
		exponent = k
	}

	// exponent is capped at 63, so this cannot fall below 1.
	return 64 - exponent
}

func DifficultyPrefixString(plength int) (s string) {
	for i := len(s); i < plength; i++ {
		s = "1" + s
	}
	return s
}
