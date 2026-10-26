package v0_22_0_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/require"
)

// These SDK counterexamples constrain the retained-self-stake migration design:
// conserving original amounts and counters alone does not preserve spendability.
func TestVestingSplitCannotIndependentlyRoundContinuousSchedules(t *testing.T) {
	newSchedule := func(amount int64) *vestingtypes.ContinuousVestingAccount {
		account, err := vestingtypes.NewContinuousVestingAccount(
			authtypes.NewBaseAccountWithAddress(sdk.AccAddress(make([]byte, 20))),
			sdk.NewCoins(sdk.NewInt64Coin("ualpha", amount)), 0, 100)
		require.NoError(t, err)
		return account
	}
	at := time.Unix(51, 0)
	original := newSchedule(2)
	retained, moved := newSchedule(1), newSchedule(1)
	require.Equal(t, original.GetOriginalVesting(), retained.GetOriginalVesting().Add(moved.GetOriginalVesting()...))
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1)), original.GetVestedCoins(at))
	// SDK RoundInt rounds each 0.51 coin up. Rescaling the same schedule to two
	// addresses would release both coins while only one originally vested.
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 2)), retained.GetVestedCoins(at).Add(moved.GetVestedCoins(at)...))
}

func TestVestingSplitCannotPreserveOnlyAggregateDelegationCounters(t *testing.T) {
	newSchedule := func(amount int64) *vestingtypes.ContinuousVestingAccount {
		coins := sdk.NewCoins(sdk.NewInt64Coin("ualpha", amount))
		account, err := vestingtypes.NewContinuousVestingAccount(
			authtypes.NewBaseAccountWithAddress(sdk.AccAddress(make([]byte, 20))), coins, 0, 100)
		require.NoError(t, err)
		account.TrackDelegation(time.Unix(0, 0), coins, coins)
		return account
	}
	original, retained, moved := newSchedule(10), newSchedule(6), newSchedule(4)
	at := time.Unix(50, 0)
	// All schedules round exactly at this instant, isolating the counter issue.
	require.Equal(t, original.GetVestingCoins(at), retained.GetVestingCoins(at).Add(moved.GetVestingCoins(at)...))
	require.Equal(t, original.GetDelegatedVesting(), retained.GetDelegatedVesting().Add(moved.GetDelegatedVesting()...))
	refund := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 5))
	original.TrackUndelegation(refund)
	retained.TrackUndelegation(refund)
	require.Equal(t, original.GetDelegatedVesting(), retained.GetDelegatedVesting().Add(moved.GetDelegatedVesting()...))
	require.True(t, original.LockedCoins(at).IsZero())
	// The retained account now locks two refunded coins even though the old
	// single account would let the player spend the entire refund.
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 2)), retained.LockedCoins(at).Add(moved.LockedCoins(at)...))
}
