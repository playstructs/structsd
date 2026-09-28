package types

import (
	"fmt"
	"math/big"
)

// CalculateReactorDiversityBonus prices an owner's current portfolio. sum is
// the sum of post-commission reactor power; squares is the sum of the square
// of each reactor's aggregate power, after combining the owner's addresses.
// Generator and received allocation power do not belong in either argument.
//
// Intermediate values and the result remain arbitrary precision. The caller
// must check representability before installing power in a uint64 grid field.
// Neither argument is mutated. There is one division, at the final step.
func CalculateReactorDiversityBonus(sum, squares *big.Int, maxBps, targetExtra uint64) (*big.Int, error) {
	if maxBps > 10000 || targetExtra == 0 {
		return nil, fmt.Errorf("invalid reactor diversity parameters")
	}
	if sum == nil || squares == nil || sum.Sign() < 0 || squares.Sign() < 0 {
		return nil, fmt.Errorf("invalid reactor diversity totals")
	}
	if sum.Sign() == 0 {
		if squares.Sign() != 0 {
			return nil, fmt.Errorf("nonzero squared power for empty reactor portfolio")
		}
		return new(big.Int), nil
	}
	sumSquared := new(big.Int).Mul(sum, sum)
	// Powers are nonnegative integers, hence sum <= squares <= sum*sum.
	// This is a consistency bound, not a substitute for the state invariant
	// that reconstructs both totals from canonical infusion rows.
	if squares.Cmp(sum) < 0 || squares.Cmp(sumSquared) > 0 {
		return nil, fmt.Errorf("inconsistent reactor diversity totals")
	}
	if maxBps == 0 {
		return new(big.Int), nil
	}
	numerator := new(big.Int).Sub(sumSquared, squares)
	ceiling := new(big.Int).Mul(squares, new(big.Int).SetUint64(targetExtra))
	if numerator.Cmp(ceiling) > 0 {
		numerator.Set(ceiling)
	}
	numerator.Mul(numerator, sum)
	numerator.Mul(numerator, new(big.Int).SetUint64(maxBps))
	denominator := new(big.Int).Mul(ceiling, big.NewInt(10000))
	return numerator.Quo(numerator, denominator), nil
}
