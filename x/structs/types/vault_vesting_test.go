package types_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func vestingCoins(n int64) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin("ualpha", n)) }

func vestingSource(seed byte) *authtypes.BaseAccount {
	address := make([]byte, 20)
	address[19] = seed
	return authtypes.NewBaseAccountWithAddress(address)
}

func vestingVault(t *testing.T, free sdk.Coins, schedules ...vestingexported.VestingAccount) *types.VaultVestingAccount {
	t.Helper()
	address, err := types.VaultAddress("1-1")
	require.NoError(t, err)
	base := authtypes.NewBaseAccountWithAddress(address)
	base.AccountNumber = 42
	vault, err := types.NewVaultVestingAccount("1-1", base, free, schedules...)
	require.NoError(t, err)
	return vault
}

// Exercise both wire forms through AccountI, as auth storage and genesis do.
// Fresh registries ensure cached Any values cannot hide stale serialized counters.
func restartVesting(t *testing.T, before *types.VaultVestingAccount) *types.VaultVestingAccount {
	t.Helper()
	newCodec := func() *codec.ProtoCodec {
		r := cdctypes.NewInterfaceRegistry()
		authtypes.RegisterInterfaces(r)
		vestingtypes.RegisterInterfaces(r)
		types.RegisterInterfaces(r)
		return codec.NewProtoCodec(r)
	}
	wire, err := newCodec().MarshalInterface(before)
	require.NoError(t, err)
	var account sdk.AccountI
	require.NoError(t, newCodec().UnmarshalInterface(wire, &account))
	json, err := newCodec().MarshalInterfaceJSON(account)
	require.NoError(t, err)
	var genesis authtypes.GenesisAccount
	require.NoError(t, newCodec().UnmarshalInterfaceJSON(json, &genesis))
	require.NoError(t, genesis.Validate())
	after := genesis.(*types.VaultVestingAccount)
	require.Equal(t, uint64(42), after.GetAccountNumber())
	require.Nil(t, after.GetPubKey())
	require.Equal(t, before.GetDelegatedFree(), after.GetDelegatedFree())
	require.Equal(t, before.GetDelegatedVesting(), after.GetDelegatedVesting())
	return after
}

func TestVaultVestingPreservesIndependentSchedulesThroughCodec(t *testing.T) {
	first, err := vestingtypes.NewDelayedVestingAccount(vestingSource(1), vestingCoins(100), 100)
	require.NoError(t, err)
	first.TrackDelegation(time.Unix(50, 0), vestingCoins(100), vestingCoins(100))
	second, err := vestingtypes.NewDelayedVestingAccount(vestingSource(2), vestingCoins(100), 200)
	require.NoError(t, err)
	continuous, err := vestingtypes.NewContinuousVestingAccount(vestingSource(3), vestingCoins(100), 100, 103)
	require.NoError(t, err)
	continuous2, err := vestingtypes.NewContinuousVestingAccount(vestingSource(4), vestingCoins(100), 100, 103)
	require.NoError(t, err)
	periodic, err := vestingtypes.NewPeriodicVestingAccount(vestingSource(5), vestingCoins(100), 100, vestingtypes.Periods{
		{Length: 2, Amount: vestingCoins(40)}, {Length: 5, Amount: vestingCoins(60)},
	})
	require.NoError(t, err)
	permanent, err := vestingtypes.NewPermanentLockedAccount(vestingSource(6), vestingCoins(100))
	require.NoError(t, err)
	sources := []vestingexported.VestingAccount{permanent, continuous2, second, periodic, first, continuous}
	vault := restartVesting(t, vestingVault(t, nil, sources...))
	for _, timestamp := range []int64{50, 99, 100, 101, 102, 103, 107, 150, 199, 200, 1000} {
		at := time.Unix(timestamp, 0)
		var locked, vested sdk.Coins
		for _, source := range sources {
			locked = locked.Add(source.LockedCoins(at)...)
			vested = vested.Add(source.GetVestedCoins(at)...)
		}
		require.True(t, locked.Equal(vault.LockedCoins(at)), "locked at %d", timestamp)
		require.True(t, vested.Equal(vault.GetVestedCoins(at)), "vested at %d", timestamp)
	}
	require.Zero(t, vault.GetEndTime(), "permanent component never unlocks")
}

func TestVaultVestingTracksAcrossSchedulesWithoutMutatingSources(t *testing.T) {
	first, err := vestingtypes.NewDelayedVestingAccount(vestingSource(1), vestingCoins(100), 100)
	require.NoError(t, err)
	second, err := vestingtypes.NewDelayedVestingAccount(vestingSource(2), vestingCoins(100), 200)
	require.NoError(t, err)
	first.TrackDelegation(time.Unix(50, 0), vestingCoins(110), vestingCoins(110))
	vault := vestingVault(t, vestingCoins(20), second, first)
	at := time.Unix(150, 0)
	require.Equal(t, vestingCoins(100), vault.LockedCoins(at))
	vault.TrackDelegation(at, vestingCoins(150), vestingCoins(120))
	vault = restartVesting(t, vault)
	require.Equal(t, vestingCoins(200), vault.GetDelegatedVesting())
	require.Equal(t, vestingCoins(50), vault.GetDelegatedFree())
	require.True(t, vault.LockedCoins(at).IsZero())
	// Refund liquid stake first, then vesting counters in source-address order.
	// The first schedule has already vested; its refund must not unlock the second.
	vault.TrackUndelegation(vestingCoins(175))
	vault = restartVesting(t, vault)
	require.True(t, vault.GetDelegatedFree().IsZero())
	require.Equal(t, vestingCoins(75), vault.GetDelegatedVesting())
	require.Equal(t, vestingCoins(25), vault.LockedCoins(at))
	require.Equal(t, vestingCoins(100), first.DelegatedVesting)
	require.Equal(t, vestingCoins(10), first.DelegatedFree)
	require.True(t, second.DelegatedVesting.IsZero())
	// SDK refunds can exceed tracked counters because of rounding.
	vault.TrackUndelegation(vestingCoins(76))
	vault = restartVesting(t, vault)
	require.True(t, vault.GetDelegatedVesting().IsZero())
	require.Equal(t, vestingCoins(100), vault.LockedCoins(at))
}

func TestVaultVestingRejectsAmbiguousAndRecursiveSchedules(t *testing.T) {
	source, err := vestingtypes.NewDelayedVestingAccount(vestingSource(1), vestingCoins(100), 200)
	require.NoError(t, err)
	vault := vestingVault(t, nil, source)
	_, err = types.NewVaultVestingAccount("1-1", vault.BaseAccount, nil, source, source)
	require.ErrorContains(t, err, "unique sorted")
	_, err = types.NewVaultVestingAccount("1-1", vault.BaseAccount, nil, vault)
	require.ErrorContains(t, err, "unsupported vesting schedule")
	_, err = types.NewVaultVestingAccount("1-2", vault.BaseAccount, nil, source)
	require.ErrorContains(t, err, "derived address")
	_, err = types.NewVaultVestingAccount("1-1", vault.BaseAccount, nil)
	require.ErrorContains(t, err, "empty schedules")
	before, err := vault.Marshal()
	require.NoError(t, err)
	require.Panics(t, func() { vault.TrackDelegation(time.Unix(100, 0), vestingCoins(5), vestingCoins(6)) })
	after, err := vault.Marshal()
	require.NoError(t, err)
	require.Equal(t, before, after, "rejected tracking must leave the account untouched")
}

// A schedule contributes no interface-unpack calls, so source count does not
// consume the SDK's 100-call budget during auth or genesis decoding.
func TestVaultVestingManySchedulesSurviveSDKUnpackBudget(t *testing.T) {
	sources := make([]vestingexported.VestingAccount, 200)
	for i := range sources {
		source, err := vestingtypes.NewDelayedVestingAccount(vestingSource(byte(i+1)), vestingCoins(100), 200)
		require.NoError(t, err)
		sources[i] = source
	}
	vault := restartVesting(t, vestingVault(t, nil, sources...))
	require.Len(t, vault.Schedules, 200)
	require.Equal(t, vestingCoins(20000), vault.LockedCoins(time.Unix(100, 0)))
	vault.TrackDelegation(time.Unix(100, 0), vestingCoins(20000), vestingCoins(15000))
	vault = restartVesting(t, vault)
	require.Equal(t, vestingCoins(5000), vault.LockedCoins(time.Unix(100, 0)))
	vault.TrackUndelegation(vestingCoins(10000))
	vault = restartVesting(t, vault)
	require.Equal(t, vestingCoins(15000), vault.LockedCoins(time.Unix(100, 0)))
}

func TestVaultVestingConcreteScheduleValidationDeniesUnknownKinds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*types.VaultVestingSchedule)
	}{
		{"unknown-kind", func(s *types.VaultVestingSchedule) { s.Kind = 500 }},
		{"zero-kind", func(s *types.VaultVestingSchedule) { s.Kind = 0 }},
		{"unexpected-periods", func(s *types.VaultVestingSchedule) {
			s.Periods = vestingtypes.Periods{{Length: 1, Amount: vestingCoins(100)}}
		}},
		{"delayed-start", func(s *types.VaultVestingSchedule) { s.StartTime = 1 }},
		{"negative-counters", func(s *types.VaultVestingSchedule) {
			s.DelegatedFree = sdk.Coins{{Denom: "ualpha", Amount: sdk.NewInt64Coin("ualpha", 1).Amount.Neg()}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, err := vestingtypes.NewDelayedVestingAccount(vestingSource(1), vestingCoins(100), 200)
			require.NoError(t, err)
			vault := vestingVault(t, nil, source)
			tc.mutate(vault.Schedules[0])
			require.NotPanics(t, func() { require.Error(t, vault.Validate()) })
		})
	}
}
