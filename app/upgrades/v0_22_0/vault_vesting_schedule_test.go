package v0_22_0_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/require"
)

// A vault migration must retain independent schedule/counter pairs. A single
// aggregated BaseVestingAccount cannot preserve the original locked balances,
// even if its GetVestingCoins computes the sum of both schedules exactly.
func TestVaultVestingMigrationMustNotNetDelegationsAcrossSchedules(t *testing.T) {
	coins := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100))
	first, err := vestingtypes.NewDelayedVestingAccount(authtypes.NewBaseAccountWithAddress(sdk.AccAddress("vesting-first-address")), coins, 100)
	require.NoError(t, err)
	second, err := vestingtypes.NewDelayedVestingAccount(authtypes.NewBaseAccountWithAddress(sdk.AccAddress("vesting-other-address")), coins, 200)
	require.NoError(t, err)
	first.TrackDelegation(time.Unix(50, 0), coins, coins)
	require.Equal(t, coins, first.DelegatedVesting)
	require.Empty(t, second.DelegatedVesting)
	naive := vestingtypes.BaseVestingAccount{
		OriginalVesting:  coins.Add(coins...),
		DelegatedVesting: first.DelegatedVesting.Add(second.DelegatedVesting...),
	}
	for _, timestamp := range []int64{99, 100, 150, 199, 200} {
		at := time.Unix(timestamp, 0)
		originalLocked := first.LockedCoins(at).Add(second.LockedCoins(at)...)
		summedVesting := first.GetVestingCoins(at).Add(second.GetVestingCoins(at)...)
		naiveLocked := naive.LockedCoinsFromVesting(summedVesting)
		if timestamp >= 100 && timestamp < 200 {
			require.Equal(t, coins, originalLocked, "second account's coins remain locked")
			require.True(t, naiveLocked.IsZero(), "naive merged counters incorrectly unlock the second schedule")
		} else {
			require.True(t, originalLocked.Equal(naiveLocked), "checking only initial/final balances misses the intervening unlock")
		}
	}
}

func TestVaultVestingMigrationMustPreservePerScheduleRounding(t *testing.T) {
	coins := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100))
	first, err := vestingtypes.NewContinuousVestingAccount(authtypes.NewBaseAccountWithAddress(sdk.AccAddress("vesting-first-address")), coins, 100, 103)
	require.NoError(t, err)
	second, err := vestingtypes.NewContinuousVestingAccount(authtypes.NewBaseAccountWithAddress(sdk.AccAddress("vesting-other-address")), coins, 100, 103)
	require.NoError(t, err)
	merged, err := vestingtypes.NewContinuousVestingAccount(authtypes.NewBaseAccountWithAddress(sdk.AccAddress("vesting-merged-address")), coins.Add(coins...), 100, 103)
	require.NoError(t, err)
	for _, tc := range []struct{ at, original, naive int64 }{{101, 66, 67}, {102, 134, 133}} {
		at := time.Unix(tc.at, 0)
		original := first.GetVestedCoins(at).Add(second.GetVestedCoins(at)...)
		require.Equal(t, tc.original, original.AmountOf("ualpha").Int64())
		require.Equal(t, tc.naive, merged.GetVestedCoins(at).AmountOf("ualpha").Int64())
	}
}
