package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	// DefaultGuildCharterDifficultyRange puts a lone miner at roughly three
	// weeks and a twenty-strong pool at roughly seventeen days, so effort buys
	// a real but bounded edge over the global race.
	DefaultGuildCharterDifficultyRange = uint64(2500000)

	// DefaultGuildCharterReactorAge is about a month at six second blocks.
	DefaultGuildCharterReactorAge = uint64(432000)

	// MinGuildCharterDifficultyRange is the floor CalculateDifficulty can be
	// asked for. It divides by log10(range), so 1 divides by zero and 0 yields
	// negative infinity.
	MinGuildCharterDifficultyRange = uint64(2)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams() Params {
	return Params{
		GuildCharterDifficultyRange: DefaultGuildCharterDifficultyRange,
		GuildCharterReactorAge:      DefaultGuildCharterReactorAge,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams()
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

/* Validate validates the set of params.
 *
 * A zero difficulty range is deliberately rejected rather than normalised:
 * CalculateDifficulty divides by log10(range), so 1 divides by zero and 0 yields
 * negative infinity, and either one pins the requirement at 64 leading zeros
 * forever. Governance must not be able to store that.
 */
func (p Params) Validate() error {
	if p.GuildCharterDifficultyRange < MinGuildCharterDifficultyRange {
		return NewParameterValidationError("guildCharterDifficultyRange", p.GuildCharterDifficultyRange, "below_minimum").
			WithRange(MinGuildCharterDifficultyRange, 0)
	}

	return nil
}

/* CharterDifficultyRange reads the difficulty range with the default
 * substituted for a value the curve cannot take.
 *
 * A Params record written before these fields existed decodes them as zero, and
 * Validate cannot reach it retroactively. The v0.21.0 handler writes the
 * defaults explicitly; this is the belt to that braces, and it is also what
 * keeps a test keeper that never set params usable.
 */
func (p Params) CharterDifficultyRange() uint64 {
	if p.GuildCharterDifficultyRange < MinGuildCharterDifficultyRange {
		return DefaultGuildCharterDifficultyRange
	}
	return p.GuildCharterDifficultyRange
}

// CharterReactorAge reads the reactor eligibility delay, substituting the
// default when unset. Zero would make every new reactor eligible immediately.
func (p Params) CharterReactorAge() uint64 {
	if p.GuildCharterReactorAge == 0 {
		return DefaultGuildCharterReactorAge
	}
	return p.GuildCharterReactorAge
}
