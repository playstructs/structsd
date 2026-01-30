package types

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
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

	difficulty := CalculateDifficulty(float64(age), difficultyRange)

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

func CalculateDifficulty(activationAge float64, difficultyRange uint64) int {
	if activationAge <= 1 {
		return 64
	}

	// Using logarithmic function to calculate difficulty.
	difficulty := 64 - int(math.Log10(activationAge)/math.Log10(float64(difficultyRange))*63)

	if difficulty < 1 {
		return 1
	}

	return difficulty
}

func DifficultyPrefixString(plength int) (s string) {
	for i := len(s); i < plength; i++ {
		s = "1" + s
	}
	return s
}
