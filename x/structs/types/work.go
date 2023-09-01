package types

import (
    "math"
    "crypto/sha256"
    "encoding/hex"

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

    difficulty := CalculateActionDifficulty(float64(age))


    position := 1
    for position <= difficulty {
        if (hash[position - 1 : position] != "0") {
            return false
        }
        position++
    }

    return true

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

    difficulty := CalculateBuildDifficulty(float64(age))

    position := 1
    for position <= difficulty {
        if (hash[position - 1 : position] != "0") {
            return false
        }
        position++
    }

    return true
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

func DifficultyPrefixString(plength int) (s string){
    for i:=len(s);i<plength;i++{
        s="1"+s
    }
    return s
}


func HashBuildAndCheckActionDifficultySabotage(input string, proof string, age uint64,difficultRange float64 ) bool {
    hash := HashBuild(input)

    if (proof != hash) {
        return false
    }

    difficulty := CalculateActionDifficultySabotage(float64(age), difficultRange)


    position := 1
    for position <= difficulty {
        if (hash[position - 1 : position] != "0") {
            return false
        }
        position++
    }

    return true

}


func CalculateActionDifficultySabotage(activationAge float64, difficultRange float64) int {
	if activationAge <= 1 {
		return 64
	}

	// Using logarithmic function to calculate difficulty.
	difficulty := 64 - int(math.Log10(activationAge)/math.Log10(difficultRange)*63)

	if difficulty < 1 {
		return 1
	}

	return difficulty
}
