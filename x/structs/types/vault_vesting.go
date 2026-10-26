package types

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"cosmossdk.io/math"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/gogoproto/proto"
)

var _ vestingexported.VestingAccount = (*VaultVestingAccount)(nil)
var _ authtypes.GenesisAccount = (*VaultVestingAccount)(nil)

// NewVaultVestingAccount clones source schedules; subsequent tracking must never
// mutate the source auth records. LiquidDelegated represents stake brought from
// ordinary accounts, whose counters were not stored in a vesting account.
func NewVaultVestingAccount(ownerID string, base *authtypes.BaseAccount, liquidDelegated sdk.Coins, sources ...vestingexported.VestingAccount) (*VaultVestingAccount, error) {
	if base == nil {
		return nil, fmt.Errorf("vault vesting requires a base account")
	}
	ordered := append([]vestingexported.VestingAccount(nil), sources...)
	for _, source := range ordered {
		if source == nil {
			return nil, fmt.Errorf("nil vesting source")
		}
	}
	sort.Slice(ordered, func(i, j int) bool { return bytes.Compare(ordered[i].GetAddress(), ordered[j].GetAddress()) < 0 })
	account := &VaultVestingAccount{
		BaseAccount: proto.Clone(base).(*authtypes.BaseAccount), OwnerId: ownerID,
		LiquidDelegated: append(sdk.Coins(nil), liquidDelegated...),
	}
	for _, source := range ordered {
		schedule, err := encodeVestingSchedule(source)
		if err != nil {
			return nil, err
		}
		account.Schedules = append(account.Schedules, schedule)
	}
	return account, account.Validate()
}

func (a *VaultVestingAccount) UnpackInterfaces(unpacker cdctypes.AnyUnpacker) error {
	if a.BaseAccount == nil {
		return fmt.Errorf("vault vesting requires a base account")
	}
	if err := a.BaseAccount.UnpackInterfaces(unpacker); err != nil {
		return err
	}
	return nil
}

func (a *VaultVestingAccount) Validate() error {
	if a.BaseAccount == nil {
		return fmt.Errorf("vault vesting requires a base account")
	}
	if err := a.BaseAccount.Validate(); err != nil {
		return err
	}
	address, err := VaultAddress(a.OwnerId)
	if err != nil || !address.Equals(a.GetAddress()) || a.GetPubKey() != nil {
		return fmt.Errorf("vesting vault must have its owner's derived address and no signing key")
	}
	if !a.LiquidDelegated.IsValid() || len(a.Schedules) == 0 {
		return fmt.Errorf("invalid vesting vault counters or empty schedules")
	}
	var previous sdk.AccAddress
	for _, encoded := range a.Schedules {
		schedule, err := decodeVestingSchedule(encoded)
		if err != nil {
			return err
		}
		if err := schedule.(authtypes.GenesisAccount).Validate(); err != nil {
			return err
		}
		if len(previous) != 0 && bytes.Compare(previous, schedule.GetAddress()) >= 0 {
			return fmt.Errorf("vesting schedules must have unique sorted source addresses")
		}
		previous = schedule.GetAddress()
	}
	return nil
}

func (a *VaultVestingAccount) schedules() []vestingexported.VestingAccount {
	result := make([]vestingexported.VestingAccount, len(a.Schedules))
	for i, encoded := range a.Schedules {
		schedule, err := decodeVestingSchedule(encoded)
		if err != nil {
			panic(err)
		}
		result[i] = schedule
	}
	return result
}

func (a *VaultVestingAccount) sum(fn func(vestingexported.VestingAccount) sdk.Coins) sdk.Coins {
	var coins sdk.Coins
	for _, schedule := range a.schedules() {
		coins = coins.Add(fn(schedule)...)
	}
	return coins
}

func (a *VaultVestingAccount) LockedCoins(at time.Time) sdk.Coins {
	return a.sum(func(s vestingexported.VestingAccount) sdk.Coins { return s.LockedCoins(at) })
}
func (a *VaultVestingAccount) GetVestedCoins(at time.Time) sdk.Coins {
	return a.sum(func(s vestingexported.VestingAccount) sdk.Coins { return s.GetVestedCoins(at) })
}
func (a *VaultVestingAccount) GetVestingCoins(at time.Time) sdk.Coins {
	return a.sum(func(s vestingexported.VestingAccount) sdk.Coins { return s.GetVestingCoins(at) })
}
func (a *VaultVestingAccount) GetOriginalVesting() sdk.Coins {
	return a.sum(func(s vestingexported.VestingAccount) sdk.Coins { return s.GetOriginalVesting() })
}
func (a *VaultVestingAccount) GetDelegatedFree() sdk.Coins {
	return a.sum(func(s vestingexported.VestingAccount) sdk.Coins { return s.GetDelegatedFree() }).Add(a.LiquidDelegated...)
}
func (a *VaultVestingAccount) GetDelegatedVesting() sdk.Coins {
	return a.sum(func(s vestingexported.VestingAccount) sdk.Coins { return s.GetDelegatedVesting() })
}
func (a *VaultVestingAccount) GetStartTime() int64 {
	start := int64(0)
	for i, s := range a.schedules() {
		if i == 0 || s.GetStartTime() < start {
			start = s.GetStartTime()
		}
	}
	return start
}
func (a *VaultVestingAccount) GetEndTime() int64 {
	end := int64(0)
	for _, s := range a.schedules() {
		if _, permanent := s.(*vestingtypes.PermanentLockedAccount); permanent {
			return 0
		}
		if s.GetEndTime() > end {
			end = s.GetEndTime()
		}
	}
	return end
}

// Tracking follows the SDK's vesting-first delegation and free-first refund
// priorities. Within each class, source-address order is deterministic. Rebuild
// each changed concrete schedule before replacing the persisted counters.
func (a *VaultVestingAccount) TrackDelegation(at time.Time, balance, amount sdk.Coins) {
	if !amount.IsValid() || !amount.IsAllPositive() || !balance.IsAllGTE(amount) {
		panic("invalid vault vesting delegation")
	}
	copy := a.trackingCopy()
	schedules := copy.schedules()
	for _, coin := range amount {
		remaining := coin.Amount
		for _, s := range schedules {
			locked := s.LockedCoins(at).AmountOf(coin.Denom)
			take := math.MinInt(remaining, locked)
			if take.IsPositive() {
				coins := sdk.NewCoins(sdk.NewCoin(coin.Denom, take))
				s.TrackDelegation(at, coins, coins)
				remaining = remaining.Sub(take)
			}
		}
		if remaining.IsPositive() {
			copy.LiquidDelegated = copy.LiquidDelegated.Add(sdk.NewCoin(coin.Denom, remaining))
		}
	}
	a.commitTracking(copy, schedules)
}

func (a *VaultVestingAccount) TrackUndelegation(amount sdk.Coins) {
	if !amount.IsValid() || !amount.IsAllPositive() {
		panic("invalid vault vesting undelegation")
	}
	copy := a.trackingCopy()
	schedules := copy.schedules()
	for _, coin := range amount {
		free := math.MinInt(coin.Amount, copy.LiquidDelegated.AmountOf(coin.Denom))
		if free.IsPositive() {
			copy.LiquidDelegated = copy.LiquidDelegated.Sub(sdk.NewCoin(coin.Denom, free))
		}
		remaining := coin.Amount.Sub(free)
		for _, vesting := range []bool{false, true} {
			for _, s := range schedules {
				available := s.GetDelegatedFree().AmountOf(coin.Denom)
				if vesting {
					available = s.GetDelegatedVesting().AmountOf(coin.Denom)
				}
				take := math.MinInt(remaining, available)
				if take.IsPositive() {
					s.TrackUndelegation(sdk.NewCoins(sdk.NewCoin(coin.Denom, take)))
					remaining = remaining.Sub(take)
				}
			}
		}
	}
	a.commitTracking(copy, schedules)
}

func (a *VaultVestingAccount) trackingCopy() *VaultVestingAccount {
	// schedules() reconstructs independent SDK values; concrete stored schedules
	// remain immutable until every new counter has been encoded successfully.
	return &VaultVestingAccount{LiquidDelegated: cloneVestingCoins(a.LiquidDelegated), Schedules: a.Schedules}
}

func (a *VaultVestingAccount) commitTracking(copy *VaultVestingAccount, schedules []vestingexported.VestingAccount) {
	encoded := make([]*VaultVestingSchedule, len(schedules))
	for i, schedule := range schedules {
		var err error
		encoded[i], err = encodeVestingSchedule(schedule)
		if err != nil {
			panic(err)
		}
	}
	a.Schedules, a.LiquidDelegated = encoded, copy.LiquidDelegated
}

func cloneVestingCoins(coins sdk.Coins) sdk.Coins {
	if coins == nil {
		return nil
	}
	result := make(sdk.Coins, len(coins))
	for i, coin := range coins {
		result[i] = sdk.NewCoin(coin.Denom, math.NewIntFromBigInt(coin.Amount.BigInt()))
	}
	return result
}

func cloneVestingPeriods(periods vestingtypes.Periods) vestingtypes.Periods {
	if periods == nil {
		return nil
	}
	result := make(vestingtypes.Periods, len(periods))
	for i, period := range periods {
		result[i] = vestingtypes.Period{Length: period.Length, Amount: cloneVestingCoins(period.Amount)}
	}
	return result
}

func encodeVestingSchedule(source vestingexported.VestingAccount) (*VaultVestingSchedule, error) {
	encoded := &VaultVestingSchedule{}
	switch s := source.(type) {
	case *vestingtypes.ContinuousVestingAccount:
		encoded.Kind = 1
	case *vestingtypes.DelayedVestingAccount:
		encoded.Kind = 2
	case *vestingtypes.PeriodicVestingAccount:
		encoded.Kind = 3
		encoded.Periods = cloneVestingPeriods(s.VestingPeriods)
	case *vestingtypes.PermanentLockedAccount:
		encoded.Kind = 4
	default:
		return nil, fmt.Errorf("unsupported vesting schedule %T", source)
	}
	encoded.SourceAddress = bytes.Clone(source.GetAddress())
	encoded.StartTime, encoded.EndTime = source.GetStartTime(), source.GetEndTime()
	encoded.OriginalVesting = cloneVestingCoins(source.GetOriginalVesting())
	encoded.DelegatedFree = cloneVestingCoins(source.GetDelegatedFree())
	encoded.DelegatedVesting = cloneVestingCoins(source.GetDelegatedVesting())
	return encoded, nil
}

// Reuse SDK arithmetic with a concrete, non-recursive representation. Historical
// keys and account sequence numbers are not schedule data or vault authority.
func decodeVestingSchedule(encoded *VaultVestingSchedule) (vestingexported.VestingAccount, error) {
	if encoded == nil {
		return nil, fmt.Errorf("nil vesting schedule")
	}
	if encoded.Kind < 1 || encoded.Kind > 4 {
		return nil, fmt.Errorf("unsupported vesting schedule kind %d", encoded.Kind)
	}
	for _, coins := range []sdk.Coins{encoded.OriginalVesting, encoded.DelegatedFree, encoded.DelegatedVesting} {
		if !coins.IsValid() {
			return nil, fmt.Errorf("invalid vesting schedule coins")
		}
	}
	for _, period := range encoded.Periods {
		if !period.Amount.IsValid() {
			return nil, fmt.Errorf("invalid vesting period coins")
		}
	}
	address := sdk.AccAddress(bytes.Clone(encoded.SourceAddress))
	if err := sdk.VerifyAddressFormat(address); err != nil {
		return nil, err
	}
	if encoded.Kind != 3 && len(encoded.Periods) != 0 {
		return nil, fmt.Errorf("periods on non-periodic schedule")
	}
	if (encoded.Kind == 2 || encoded.Kind == 4) && encoded.StartTime != 0 {
		return nil, fmt.Errorf("unexpected vesting start time")
	}
	if encoded.Kind == 4 && encoded.EndTime != 0 {
		return nil, fmt.Errorf("permanent vesting cannot end")
	}
	base := &vestingtypes.BaseVestingAccount{
		BaseAccount: authtypes.NewBaseAccountWithAddress(address), EndTime: encoded.EndTime,
		OriginalVesting:  cloneVestingCoins(encoded.OriginalVesting),
		DelegatedFree:    cloneVestingCoins(encoded.DelegatedFree),
		DelegatedVesting: cloneVestingCoins(encoded.DelegatedVesting),
	}
	switch encoded.Kind {
	case 1:
		return &vestingtypes.ContinuousVestingAccount{BaseVestingAccount: base, StartTime: encoded.StartTime}, nil
	case 2:
		return &vestingtypes.DelayedVestingAccount{BaseVestingAccount: base}, nil
	case 3:
		return &vestingtypes.PeriodicVestingAccount{BaseVestingAccount: base, StartTime: encoded.StartTime, VestingPeriods: cloneVestingPeriods(encoded.Periods)}, nil
	case 4:
		return &vestingtypes.PermanentLockedAccount{BaseVestingAccount: base}, nil
	default:
		return nil, fmt.Errorf("unsupported vesting schedule kind %d", encoded.Kind)
	}
}
