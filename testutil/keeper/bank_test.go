package keeper

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMockBankDebitRequiresEveryRequestedDenom(t *testing.T) {
	for _, moduleDestination := range []bool{false, true} {
		for _, tc := range []struct {
			name    string
			balance sdk.Coins
			amount  sdk.Coins
			allowed bool
		}{
			{"empty", sdk.Coins{}, sdk.NewCoins(sdk.NewInt64Coin("utoken", 1)), false},
			{"missing_denom", sdk.NewCoins(sdk.NewInt64Coin("ualpha", 333)), sdk.NewCoins(sdk.NewInt64Coin("utoken", 1)), false},
			{"short_with_other_asset", sdk.NewCoins(sdk.NewInt64Coin("ualpha", 333), sdk.NewInt64Coin("utoken", 7)), sdk.NewCoins(sdk.NewInt64Coin("utoken", 8)), false},
			{"one_of_two_short", sdk.NewCoins(sdk.NewInt64Coin("ualpha", 333), sdk.NewInt64Coin("utoken", 7)), sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1), sdk.NewInt64Coin("utoken", 8)), false},
			{"exact_with_other_asset", sdk.NewCoins(sdk.NewInt64Coin("ualpha", 333), sdk.NewInt64Coin("utoken", 7)), sdk.NewCoins(sdk.NewInt64Coin("utoken", 7)), true},
		} {
			destination := "account/"
			if moduleDestination {
				destination = "module/"
			}
			t.Run(destination+tc.name, func(t *testing.T) {
				bank := NewMockBankKeeper()
				from, to := sdk.AccAddress("mock-bank-source-001"), sdk.AccAddress("mock-bank-target-001")
				bank.balances[from.String()] = tc.balance
				var err error
				require.NotPanics(t, func() {
					if moduleDestination {
						err = bank.SendCoinsFromAccountToModule(context.Background(), from, "structs", tc.amount)
					} else {
						err = bank.SendCoins(context.Background(), from, to, tc.amount)
					}
				})
				if tc.allowed {
					require.NoError(t, err)
					require.Equal(t, tc.balance.Sub(tc.amount...), bank.balances[from.String()])
					if !moduleDestination {
						require.Equal(t, tc.amount, bank.balances[to.String()])
					}
				} else {
					require.ErrorContains(t, err, "insufficient funds")
					require.Equal(t, tc.balance, bank.balances[from.String()])
					require.Empty(t, bank.balances[to.String()])
				}
			})
		}
	}
}
