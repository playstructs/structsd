package app_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

// Retained stake cannot share vesting accounting through a SendRestriction:
// native maturity refunds update the account directly without visiting it.
func TestNativeVestingRefundBypassesSendRestrictionAndUpdatesCountersAtMaturity(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	primary, _, _ := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	coins := func(n int64) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin(denom, n)) }
	base := bApp.AccountKeeper.GetAccount(ctx, primary).(*authtypes.BaseAccount)
	vesting, err := vestingtypes.NewDelayedVestingAccount(base, coins(1000000), ctx.BlockTime().Add(365*24*time.Hour).Unix())
	require.NoError(t, err)
	bApp.AccountKeeper.SetAccount(ctx, vesting)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(1000000))
	_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(primary.String(), validator.String(), coins(200000)[0]))
	require.NoError(t, err)
	pending := bApp.AccountKeeper.GetAccount(ctx, primary).(*vestingtypes.DelayedVestingAccount)
	require.Equal(t, coins(1000000), pending.GetDelegatedVesting(), "initiating unbonding does not release the vesting counters")
	ubd, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, primary, validator)
	require.NoError(t, err)
	restrictionCalls := 0
	bApp.BankKeeper.AppendSendRestriction(func(_ context.Context, _, to sdk.AccAddress, _ sdk.Coins) (sdk.AccAddress, error) {
		restrictionCalls++
		return to, nil
	})
	mature := ctx.WithBlockTime(ubd.Entries[0].CompletionTime.Add(time.Second))
	paid, err := bApp.StakingKeeper.CompleteUnbonding(mature, primary, validator)
	require.NoError(t, err)
	require.Equal(t, coins(200000), paid)
	require.Zero(t, restrictionCalls, "maturity does not call SendCoins or its restrictions")
	after := bApp.AccountKeeper.GetAccount(mature, primary).(*vestingtypes.DelayedVestingAccount)
	require.Equal(t, coins(800000), after.GetDelegatedVesting())
	require.Equal(t, coins(200000), bApp.BankKeeper.GetAllBalances(mature, primary))
	require.Equal(t, coins(200000), after.LockedCoins(mature.BlockTime()))
	require.True(t, bApp.BankKeeper.SpendableCoins(mature, primary).IsZero())
	requireStakingAndDistributionHealthy(t, bApp, mature)
}
