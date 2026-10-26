package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestProviderRateValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rate  sdk.Coin
		valid bool
	}{
		{"paid", sdk.NewInt64Coin("ualpha", 3), true},
		{"free", sdk.NewInt64Coin("ualpha", 0), true},
		{"guild token", sdk.NewInt64Coin("uguild.0-1", 3), true},
		{"nil amount", sdk.Coin{Denom: "ualpha"}, false},
		{"negative", sdk.Coin{Denom: "ualpha", Amount: math.NewInt(-1)}, false},
		{"invalid denom", sdk.Coin{Denom: "!", Amount: math.NewInt(3)}, false},
		{"empty denom", sdk.Coin{Amount: math.NewInt(3)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := types.Provider{Id: "10-1", DurationMinimum: 1, DurationMaximum: 100, Rate: sdk.NewInt64Coin("ualpha", 7)}
			before := provider
			err := provider.SetRate(tc.rate)
			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, tc.rate, provider.Rate)
			} else {
				require.ErrorContains(t, err, "invalid_coin")
				require.ErrorIs(t, err, types.ErrParameterValidation)
				require.Equal(t, before, provider, "rejected rates cannot replace the published rate")
			}
			genesis := types.DefaultGenesis()
			provider.Rate = tc.rate // Whole-record imports do not call SetRate.
			genesis.ProviderList = []types.Provider{provider}
			if tc.valid {
				require.NoError(t, genesis.Validate())
			} else {
				require.ErrorContains(t, genesis.Validate(), "invalid_coin")
			}
		})
	}
}
