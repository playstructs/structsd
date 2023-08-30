package types

import (
    "math"
    "crypto/sha256"
    "encoding/hex"
    "strconv"
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

func HashBuildAndCheckActionDifficulty(input string, proof string, age uint64) bool {
    hash := HashBuild(input)

    if (proof != hash) {
        return false
    }

    i, err := strconv.ParseUint(hash[:CalculateActionDifficulty(float64(age))], 10, 64)

    // Either the string isn't all 0's and can't
    // be converted, or it's a number greater than zero
    return (err != nil || i > 0)
}



func CalculateActionDifficulty(activationAge float64) int {
	if activationAge <= 1 {
		return 64
	}

	// Using logarithmic function to calculate difficulty.
	difficulty := 64 - int(math.Log10(activationAge)/math.Log10(DifficultyActionAgeRange)*63)

	if difficulty < 1 {
		return 1
	}

	return difficulty
}


func HashBuildAndCheckBuildDifficulty(input string, proof string, age uint64) bool {
    hash := HashBuild(input)

    if (proof != hash) {
        return false
    }

    i, err := strconv.ParseUint(hash[:CalculateActionDifficulty(float64(age))], 10, 64)

    // Either the string isn't all 0's and can't
    // be converted, or it's a number greater than zero
    return (err != nil || i > 0)
}


func CalculateBuildDifficulty(activationAge float64) int {
	if activationAge <= 1 {
		return 64
	}

	// Using logarithmic function to calculate difficulty.
	difficulty := 64 - int(math.Log10(activationAge)/math.Log10(DifficultyBuildAgeRange)*63)

	if difficulty < 1 {
		return 1
	}

	return difficulty
}