package types_test

import (
	"github.com/stretchr/testify/require"
	"math/big"
	"structs/x/structs/types"
	"testing"
)

func TestReactorStakeCapExactBoundaryAndHeadroom(t *testing.T) {
	for _, multiplier := range []*big.Int{big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), 255)} {
		tokens := new(big.Int).Mul(big.NewInt(3), multiplier)
		total := new(big.Int).Mul(big.NewInt(10), multiplier)
		require.True(t, types.ReactorStakeWithinCap(tokens, total, 3000))
		require.False(t, types.ReactorStakeWithinCap(new(big.Int).Add(tokens, big.NewInt(1)), total, 3000))
	}
	require.Equal(t, "285714", types.ReactorStakeHeadroom(big.NewInt(1000000), big.NewInt(4000000), 3000, true))
	require.Equal(t, "200000", types.ReactorStakeHeadroom(big.NewInt(1000000), big.NewInt(4000000), 3000, false))
	require.Equal(t, "0", types.ReactorStakeHeadroom(big.NewInt(4000000), big.NewInt(4000000), 3000, true))
	require.Equal(t, "", types.ReactorStakeHeadroom(big.NewInt(4000000), big.NewInt(4000000), 10000, true))
	require.True(t, types.ReactorStakeWithinCap(big.NewInt(0), big.NewInt(0), 3000))
	require.False(t, types.ReactorStakeWithinCap(big.NewInt(1), big.NewInt(0), 3000))
}

func TestReactorParameterValidation(t *testing.T) {
	defaults := types.DefaultParams()
	require.NoError(t, defaults.Validate())
	legacy := types.Params{GuildCharterDifficultyRange: defaults.GuildCharterDifficultyRange, GuildCharterReactorAge: defaults.GuildCharterReactorAge}
	require.Equal(t, defaults, legacy.WithReactorDefaults())
	for _, edit := range []func(*types.Params){
		func(p *types.Params) { p.ReactorStakeCapBps = 0 },
		func(p *types.Params) { p.ReactorStakeCapBps = 10001 },
		func(p *types.Params) { p.ReactorStakeCapMinValidators = 0 },
		func(p *types.Params) { p.DiversityMaxBonusBps = 10001 },
		func(p *types.Params) { p.DiversityTargetExtraReactors = 0 },
	} {
		p := defaults
		edit(&p)
		require.Error(t, p.Validate())
	}
	defaults.DiversityMaxBonusBps = 0
	require.NoError(t, defaults.Validate())
	require.Zero(t, defaults.WithReactorDefaults().DiversityMaxBonusBps)
}
