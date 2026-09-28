package types

import (
	"fmt"
	"math/big"
)

// Cross-products use arbitrary precision: SDK token amounts already occupy up
// to 256 bits, so multiplying them inside math.Int can overflow its own bound.
func ReactorStakeWithinCap(tokens, denominator *big.Int, capBps uint64) bool {
	if tokens == nil || denominator == nil || tokens.Sign() < 0 || denominator.Sign() < 0 || capBps == 0 || capBps > 10000 {
		return false
	}
	left := new(big.Int).Mul(tokens, big.NewInt(10000))
	right := new(big.Int).Mul(denominator, new(big.Int).SetUint64(capBps))
	return left.Cmp(right) <= 0
}

// ReactorStakeHeadroom prices new inflows. When growsDenominator is false, the
// incoming tokens were already bonded elsewhere. An empty string means no cap
// ceiling. Zero means this operation has no headroom, including an over-cap row.
func ReactorStakeHeadroom(tokens, denominator *big.Int, capBps uint64, growsDenominator bool) string {
	if capBps == 10000 {
		return ""
	}
	numerator := new(big.Int).Mul(denominator, new(big.Int).SetUint64(capBps))
	numerator.Sub(numerator, new(big.Int).Mul(tokens, big.NewInt(10000)))
	if numerator.Sign() <= 0 {
		return "0"
	}
	divisor := uint64(10000)
	if growsDenominator {
		divisor -= capBps
	}
	return numerator.Quo(numerator, new(big.Int).SetUint64(divisor)).String()
}

// ReactorStakeCapError preserves the post-operation values for actionable
// transaction diagnostics without narrowing SDK token amounts to uint64.
type ReactorStakeCapError struct {
	ValidatorAddress string
	Tokens           string
	Denominator      string
	CapBps           uint64
}

func (e *ReactorStakeCapError) Error() string {
	return fmt.Sprintf("reactor stake cap exceeded: validator %s has %s of %s tokens (cap %d bps)", e.ValidatorAddress, e.Tokens, e.Denominator, e.CapBps)
}
func (e *ReactorStakeCapError) Unwrap() error { return ErrReactorStakeCapExceeded }
func (e *ReactorStakeCapError) Code() uint32  { return ErrReactorStakeCapExceeded.ABCICode() }

// ABCI reporting uses these interfaces rather than the standard Unwrap chain.
// Preserve the actionable cap code when BaseApp wraps a failed message.
func (e *ReactorStakeCapError) ABCICode() uint32  { return e.Code() }
func (e *ReactorStakeCapError) Codespace() string { return ErrReactorStakeCapExceeded.Codespace() }
func (e *ReactorStakeCapError) LogFields() []interface{} {
	return []interface{}{"error_type", "reactor_stake_cap", "validator", e.ValidatorAddress, "tokens", e.Tokens, "denominator", e.Denominator, "cap_bps", e.CapBps}
}
