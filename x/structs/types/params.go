package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	DefaultReactorStakeCapBps           = uint64(3000)
	DefaultReactorStakeCapMinValidators = uint64(10)
	DefaultDiversityMaxBonusBps         = uint64(5000)
	DefaultDiversityTargetExtraReactors = uint64(9)
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
		ReactorStakeCapBps:           DefaultReactorStakeCapBps,
		ReactorStakeCapMinValidators: DefaultReactorStakeCapMinValidators,
		DiversityMaxBonusBps:         DefaultDiversityMaxBonusBps,
		DiversityTargetExtraReactors: DefaultDiversityTargetExtraReactors,
		GuildCharterDifficultyRange:  DefaultGuildCharterDifficultyRange,
		GuildCharterReactorAge:       DefaultGuildCharterReactorAge,
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

	return p.WithReactorDefaults().ValidateReactorParams()
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

// WithReactorDefaults supports source states written before the new fields
// existed. Only the completely absent tuple receives defaults. Partial/invalid
// configuration is rejected, and an explicitly configured zero bonus is valid.
func (p Params) WithReactorDefaults() Params {
	if p.ReactorStakeCapBps == 0 && p.ReactorStakeCapMinValidators == 0 && p.DiversityMaxBonusBps == 0 && p.DiversityTargetExtraReactors == 0 {
		p.ReactorStakeCapBps = DefaultReactorStakeCapBps
		p.ReactorStakeCapMinValidators = DefaultReactorStakeCapMinValidators
		p.DiversityMaxBonusBps = DefaultDiversityMaxBonusBps
		p.DiversityTargetExtraReactors = DefaultDiversityTargetExtraReactors
	}
	return p
}

func (p Params) ValidateReactorParams() error {
	if p.ReactorStakeCapBps == 0 || p.ReactorStakeCapBps > 10000 {
		return NewParameterValidationError("reactorStakeCapBps", p.ReactorStakeCapBps, "outside_range").WithRange(1, 10000)
	}
	if p.ReactorStakeCapMinValidators == 0 {
		return NewParameterValidationError("reactorStakeCapMinValidators", 0, "below_minimum").WithRange(1, 0)
	}
	if p.DiversityMaxBonusBps > 10000 {
		return NewParameterValidationError("diversityMaxBonusBps", p.DiversityMaxBonusBps, "above_maximum").WithRange(0, 10000)
	}
	if p.DiversityTargetExtraReactors == 0 {
		return NewParameterValidationError("diversityTargetExtraReactors", 0, "below_minimum").WithRange(1, 0)
	}
	return nil
}
