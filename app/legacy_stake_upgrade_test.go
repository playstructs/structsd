package app_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Include raw queues, operation-ID indices, rewards, and withdrawal settings:
// a successful upgrade must not change even one native staking/distribution row.
func TestVaultUpgradeLeavesNativeStakingAndDistributionByteIdentical(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 2)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	source, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	destination, err := sdk.ValAddressFromBech32(validators[1].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	primary, secondary, id := newTransferPlayer(t, bApp, ctx)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	for _, address := range []sdk.AccAddress{primary, secondary} {
		delegate(t, bApp, ctx, server, denom, address, source, math.NewInt(3000000))
		_, err := server.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(address.String(), source.String(), destination.String(), sdk.NewInt64Coin(denom, 1000000)))
		require.NoError(t, err)
		_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(address.String(), destination.String(), sdk.NewInt64Coin(denom, 100000)))
		require.NoError(t, err)
		ubd, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, address, destination)
		require.NoError(t, err)
		require.NoError(t, bApp.StakingKeeper.PutUnbondingOnHold(ctx, ubd.Entries[0].UnbondingId))
		red, err := bApp.StakingKeeper.GetRedelegation(ctx, address, source, destination)
		require.NoError(t, err)
		require.NoError(t, bApp.StakingKeeper.PutUnbondingOnHold(ctx, red.Entries[0].UnbondingId))
		require.NoError(t, bApp.DistrKeeper.SetDelegatorWithdrawAddr(ctx, address, secondary))
		fundRedelegationAccount(t, bApp, ctx, address, sdk.NewCoins(sdk.NewInt64Coin(denom, 17)))
	}
	snapshot := func() map[string]string {
		result := map[string]string{}
		for _, module := range []string{stakingtypes.StoreKey, distrtypes.StoreKey} {
			it := ctx.KVStore(bApp.GetKey(module)).Iterator(nil, nil)
			for ; it.Valid(); it.Next() {
				result[module+":"+string(it.Key())] = string(it.Value())
			}
			it.Close()
		}
		return result
	}
	before := snapshot()
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, v22.MigrateVaultAssets(ctx, delegationUpgradeKeepers(bApp)))
	require.Equal(t, before, snapshot())
	require.Empty(t, migrationEvents[*types.EventVaultDelegationMigration](t, ctx, "EventVaultDelegationMigration"))
	require.Empty(t, migrationEvents[*types.EventVaultUnbondingMigration](t, ctx, "EventVaultUnbondingMigration"))
	require.Empty(t, migrationEvents[*types.EventVaultRedelegationMigration](t, ctx, "EventVaultRedelegationMigration"))
	vault, err := types.VaultAddress(id)
	require.NoError(t, err)
	require.Equal(t, int64(34), bApp.BankKeeper.GetBalance(ctx, vault, denom).Amount.Int64())
	for _, address := range []sdk.AccAddress{primary, secondary} {
		row, found := bApp.StructsKeeper.GetInfusion(ctx, jailGateReactorId(t, bApp, ctx, destination), address.String())
		require.True(t, found)
		require.Equal(t, uint64(900000), row.Fuel)
		require.Equal(t, uint64(100000), row.Defusing)
	}
	require.NoError(t, v22.MigrateVaultAssets(ctx, delegationUpgradeKeepers(bApp)))
	require.Equal(t, before, snapshot(), "rerunning the phase must not rewrite native records")
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

func TestLegacyStakeCanUnbondSendToVaultAndDelegateVoluntarily(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	primary, _, id := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, primary, validator, math.NewInt(1000000))
	require.NoError(t, v22.MigrateVaultAssets(ctx, delegationUpgradeKeepers(bApp)))
	vault, err := types.VaultAddress(id)
	require.NoError(t, err)
	_, err = bApp.StakingKeeper.GetDelegation(ctx, vault, validator)
	require.ErrorIs(t, err, stakingtypes.ErrNoDelegation)
	_, err = bApp.DistrKeeper.WithdrawDelegationRewards(ctx, primary, validator)
	require.NoError(t, err)
	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	response, err := ms.ReactorDefuse(ctx, &types.MsgReactorDefuse{Creator: primary.String(), DelegatorAddress: primary.String(), ValidatorAddress: validator.String(), Amount: sdk.NewInt64Coin(denom, 1000000)})
	require.NoError(t, err)
	ctx = ctx.WithBlockTime(response.CompletionTime.Add(time.Second))
	header := ctx.HeaderInfo()
	header.Time = ctx.BlockTime()
	ctx = ctx.WithHeaderInfo(header)
	_, err = bApp.StakingKeeper.DequeueAllMatureUBDQueue(ctx, ctx.BlockTime())
	require.NoError(t, err)
	paid, err := bApp.StakingKeeper.CompleteUnbonding(ctx, primary, validator)
	require.NoError(t, err)
	bApp.StructsKeeper.ProcessInfusionMaturitySweep(ctx)
	require.Equal(t, int64(1000000), paid.AmountOf(denom).Int64())
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, vault).IsZero())
	_, err = ms.PlayerSend(ctx, &types.MsgPlayerSend{Creator: primary.String(), FromAddress: primary.String(), ToAddress: vault.String(), Amount: paid})
	require.NoError(t, err)
	_, err = ms.ReactorInfuse(ctx, &types.MsgReactorInfuse{Creator: primary.String(), ValidatorAddress: validator.String(), Amount: paid[0]})
	require.NoError(t, err)
	_, err = bApp.StakingKeeper.GetDelegation(ctx, primary, validator)
	require.ErrorIs(t, err, stakingtypes.ErrNoDelegation)
	d, err := bApp.StakingKeeper.GetDelegation(ctx, vault, validator)
	require.NoError(t, err)
	v, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	require.Equal(t, int64(1000000), v.TokensFromShares(d.Shares).TruncateInt64())
	row, found := bApp.StructsKeeper.GetInfusion(ctx, jailGateReactorId(t, bApp, ctx, validator), vault.String())
	require.True(t, found)
	require.Equal(t, id, row.PlayerId)
	require.Equal(t, uint64(1000000), row.Fuel)
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
