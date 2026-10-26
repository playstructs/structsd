package app_test

import (
	"bytes"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Uses auth persistence and real bank/staking hooks. The source schedules are
// installed explicitly; this proves continuation, not the unwired migration.
func TestVaultVestingRealStakingPreservesLocksThroughCancellationAndMaturity(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
	_, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(playerID)
	require.NoError(t, err)
	vault, err := types.VaultAddress(playerID)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	coins := func(n int64) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin(denom, n)) }
	params, err := bApp.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	firstEnd := ctx.BlockTime().Add(params.UnbondingTime + 24*time.Hour)
	secondEnd := firstEnd.Add(24 * time.Hour)
	firstAddress, secondAddress := primary, secondary
	if bytes.Compare(firstAddress, secondAddress) > 0 {
		firstAddress, secondAddress = secondAddress, firstAddress
	}
	first, err := vestingtypes.NewDelayedVestingAccount(authtypes.NewBaseAccountWithAddress(firstAddress), coins(500000), firstEnd.Unix())
	require.NoError(t, err)
	second, err := vestingtypes.NewDelayedVestingAccount(authtypes.NewBaseAccountWithAddress(secondAddress), coins(500000), secondEnd.Unix())
	require.NoError(t, err)
	account := bApp.AccountKeeper.GetAccount(ctx, vault)
	composite, err := types.NewVaultVestingAccount(playerID, account.(*authtypes.BaseAccount), nil, first, second)
	require.NoError(t, err)
	bApp.AccountKeeper.SetAccount(ctx, composite)
	fundRedelegationAccount(t, bApp, ctx, vault, coins(1000000))
	read := func(at sdk.Context) *types.VaultVestingAccount {
		stored := bApp.AccountKeeper.GetAccount(at, vault).(*types.VaultVestingAccount)
		require.NoError(t, stored.Validate())
		require.Equal(t, account.GetAccountNumber(), stored.GetAccountNumber())
		return stored
	}
	require.True(t, bApp.BankKeeper.SpendableCoins(ctx, vault).IsZero())
	// Use a throwaway cache because this is a failed bank operation rather than
	// BaseApp's transaction branch.
	failedCtx, _ := ctx.CacheContext()
	require.Error(t, bApp.BankKeeper.SendCoins(failedCtx, vault, primary, coins(1)))
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	_, err = ms.ReactorInfuse(ctx, &types.MsgReactorInfuse{Creator: primary.String(), ValidatorAddress: validator.String(), Amount: coins(750000)[0]})
	require.NoError(t, err)
	require.Equal(t, coins(750000), read(ctx).GetDelegatedVesting())
	require.Equal(t, coins(250000), read(ctx).LockedCoins(ctx.BlockTime()))
	_, err = ms.ReactorDefuse(ctx, &types.MsgReactorDefuse{Creator: primary.String(), ValidatorAddress: validator.String(), Amount: coins(300000)[0]})
	require.NoError(t, err)
	_, err = ms.ReactorCancelDefusion(ctx, &types.MsgReactorCancelDefusion{Creator: primary.String(), ValidatorAddress: validator.String(), Amount: coins(100000)[0], CreationHeight: ctx.BlockHeight()})
	require.NoError(t, err)
	require.Equal(t, coins(750000), read(ctx).GetDelegatedVesting(), "pending or cancelled unbondings have not returned bank funds")
	ubd, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, vault, validator)
	require.NoError(t, err)
	require.Len(t, ubd.Entries, 1)
	matureCtx := ctx.WithBlockTime(ubd.Entries[0].CompletionTime.Add(time.Second))
	paid, err := bApp.StakingKeeper.CompleteUnbonding(matureCtx, vault, validator)
	require.NoError(t, err)
	require.Equal(t, coins(200000), paid)
	require.Equal(t, coins(550000), read(matureCtx).GetDelegatedVesting())
	require.Equal(t, coins(450000), read(matureCtx).LockedCoins(matureCtx.BlockTime()))
	require.True(t, bApp.BankKeeper.SpendableCoins(matureCtx, vault).IsZero())
	// The first component in deterministic address order received the original
	// 500k delegation and its 200k refund. Expiring that component unlocks only
	// its returned coins; the other source's remaining 250k stays locked.
	unlockedCtx := matureCtx.WithBlockTime(firstEnd)
	require.Equal(t, coins(200000), bApp.BankKeeper.SpendableCoins(unlockedCtx, vault))
	endCtx := matureCtx.WithBlockTime(secondEnd)
	require.Equal(t, coins(450000), bApp.BankKeeper.SpendableCoins(endCtx, vault))
	require.NoError(t, bApp.BankKeeper.SendCoins(endCtx, vault, primary, coins(450000)))
	require.True(t, bApp.BankKeeper.GetAllBalances(endCtx, vault).IsZero())
	requireStakingAndDistributionHealthy(t, bApp, endCtx)
}
