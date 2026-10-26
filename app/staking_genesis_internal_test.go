package app

import (
	"math"
	"testing"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

func TestStakingGenesisLastID(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		validator, unbonding, redelegation []uint64
		exported, want                     uint64
		invalid                            bool
	}{
		{name: "fresh"},
		{name: "completed only", exported: 47, want: 47},
		{name: "legacy live maximum", validator: []uint64{3}, unbonding: []uint64{8}, redelegation: []uint64{5}, want: 8},
		{name: "completed above live", validator: []uint64{3}, unbonding: []uint64{8}, redelegation: []uint64{5}, exported: 47, want: 47},
		{name: "counter behind live", unbonding: []uint64{8}, exported: 7, invalid: true},
		{name: "cross type collision", validator: []uint64{3}, redelegation: []uint64{3}, invalid: true},
		{name: "same type collision", unbonding: []uint64{8, 8}, invalid: true},
		{name: "zero validator", validator: []uint64{0}, invalid: true},
		{name: "zero unbonding", unbonding: []uint64{0}, invalid: true},
		{name: "zero redelegation", redelegation: []uint64{0}, invalid: true},
		{name: "exhausted export", exported: math.MaxUint64, invalid: true},
		{name: "exhausted legacy", redelegation: []uint64{math.MaxUint64}, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			genesis := stakingtypes.GenesisState{Validators: []stakingtypes.Validator{{UnbondingIds: tc.validator}}}
			for _, id := range tc.unbonding {
				genesis.UnbondingDelegations = append(genesis.UnbondingDelegations, stakingtypes.UnbondingDelegation{Entries: []stakingtypes.UnbondingDelegationEntry{{UnbondingId: id}}})
			}
			for _, id := range tc.redelegation {
				genesis.Redelegations = append(genesis.Redelegations, stakingtypes.Redelegation{Entries: []stakingtypes.RedelegationEntry{{UnbondingId: id}}})
			}
			got, err := stakingGenesisLastID(genesis, tc.exported)
			if tc.invalid {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}
