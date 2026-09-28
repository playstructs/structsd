package types_test

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func diversityTotals(powers ...uint64) (*big.Int, *big.Int) {
	sum, squares := new(big.Int), new(big.Int)
	for _, power := range powers {
		p := new(big.Int).SetUint64(power)
		sum.Add(sum, p)
		squares.Add(squares, new(big.Int).Mul(p, p))
	}
	return sum, squares
}

func TestReactorDiversityCurve(t *testing.T) {
	for _, count := range []int{0, 1, 2, 3, 5, 10, 11} {
		powers := make([]uint64, count)
		for i := range powers {
			powers[i] = 9000
		}
		sum, squares := diversityTotals(powers...)
		bonus, err := types.CalculateReactorDiversityBonus(sum, squares, 5000, 9)
		require.NoError(t, err)
		expected := max(0, min(9, count-1)) * count * 500
		require.Equal(t, int64(expected), bonus.Int64())
	}
	// One final division: two one-power reactors have a fractional effective
	// bonus that must not become one unit through intermediate rounding.
	sum, squares := diversityTotals(1, 1)
	bonus, err := types.CalculateReactorDiversityBonus(sum, squares, 5000, 9)
	require.NoError(t, err)
	require.Zero(t, bonus.Sign())
	// The cap and slope are configurable, including disabling the bonus.
	sum, squares = diversityTotals(100, 100, 100)
	for _, sample := range []struct{ bps, extra, expected uint64 }{{3000, 2, 90}, {5000, 4, 75}, {0, 9, 0}} {
		bonus, err := types.CalculateReactorDiversityBonus(sum, squares, sample.bps, sample.extra)
		require.NoError(t, err)
		require.Equal(t, sample.expected, bonus.Uint64())
	}
}

func TestReactorDiversityAcceptedEconomicProperties(t *testing.T) {
	price := func(powers ...uint64) uint64 {
		t.Helper()
		sum, squares := diversityTotals(powers...)
		bonus, err := types.CalculateReactorDiversityBonus(sum, squares, 5000, 9)
		require.NoError(t, err)
		return bonus.Uint64()
	}
	// The identity limitation is intentional: separate owners can obtain a
	// larger combined bonus. The chain does not infer common ownership.
	require.Equal(t, uint64(4), price(1000, 10, 10, 10, 10))
	require.Equal(t, uint64(6), price(1000)+price(10, 10, 10, 10))
	// Concentrating more stake can lower total capacity; callers must apply
	// the resulting decrease and schedule GridCascade when necessary.
	powers := append([]uint64{600}, make([]uint64, 20)...)
	for i := 1; i < len(powers); i++ {
		powers[i] = 70
	}
	require.Equal(t, uint64(2859), uint64(2000)+price(powers...))
	powers[0] = 650
	require.Equal(t, uint64(2855), uint64(2050)+price(powers...))
}

func TestReactorDiversityExactArithmeticAgainstRationalReference(t *testing.T) {
	rng := rand.New(rand.NewSource(923))
	for sample := 0; sample < 400; sample++ {
		powers := make([]uint64, 1+rng.Intn(25))
		for i := range powers {
			powers[i] = rng.Uint64()
		}
		sum, squares := diversityTotals(powers...)
		originalSum, originalSquares := sum.String(), squares.String()
		bps, target := uint64(rng.Intn(10001)), uint64(1+rng.Intn(30))
		bonus, err := types.CalculateReactorDiversityBonus(sum, squares, bps, target)
		require.NoError(t, err)
		// Independently evaluate the effective-reactor form using exact
		// fractions, then floor once. S2 routinely exceeds uint64 here.
		effective := new(big.Rat).SetFrac(new(big.Int).Mul(sum, sum), squares)
		effective.Sub(effective, big.NewRat(1, 1))
		effective.Quo(effective, new(big.Rat).SetInt(new(big.Int).SetUint64(target)))
		if effective.Cmp(big.NewRat(1, 1)) > 0 {
			effective.SetInt64(1)
		}
		effective.Mul(effective, new(big.Rat).SetFrac(new(big.Int).SetUint64(bps), big.NewInt(10000)))
		effective.Mul(effective, new(big.Rat).SetInt(sum))
		expected := new(big.Int).Quo(effective.Num(), effective.Denom())
		require.Equal(t, expected.String(), bonus.String())
		require.Equal(t, originalSum, sum.String())
		require.Equal(t, originalSquares, squares.String())
	}
}

func TestReactorDiversityRejectsInvalidTotalsAndParameters(t *testing.T) {
	for _, sample := range []struct {
		sum, squares *big.Int
		bps, extra   uint64
	}{
		{nil, big.NewInt(1), 5000, 9},
		{big.NewInt(1), nil, 5000, 9},
		{big.NewInt(-1), big.NewInt(1), 5000, 9},
		{big.NewInt(0), big.NewInt(1), 5000, 9},
		{big.NewInt(2), big.NewInt(1), 5000, 9},
		{big.NewInt(2), big.NewInt(5), 5000, 9},
		{big.NewInt(1), big.NewInt(1), 10001, 9},
		{big.NewInt(1), big.NewInt(1), 5000, 0},
	} {
		_, err := types.CalculateReactorDiversityBonus(sample.sum, sample.squares, sample.bps, sample.extra)
		require.Error(t, err)
	}
}
