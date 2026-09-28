package app_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
)

func TestVaultInfusionMigrationRebuildsFuelAndUnbondingMaturity(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	validators, err := bApp.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(3000000))
	delegate(t, bApp, ctx, server, denom, secondary, validator, math.NewInt(4000000))
	_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(primary.String(), validator.String(), sdk.NewInt64Coin(denom, 1000000)))
	require.NoError(t, err)
	reactorBytes, found := bApp.StructsKeeper.GetReactorBytesFromValidator(ctx, validator)
	require.True(t, found)
	reactor, found := bApp.StructsKeeper.GetReactorByBytes(ctx, reactorBytes)
	require.True(t, found)
	playerCapacity := redelegationCapacity(t, bApp, ctx, playerID)
	require.NotZero(t, playerCapacity)
	keepers := &upgrades.Keepers{StructsKeeper: bApp.StructsKeeper, StakingKeeper: bApp.StakingKeeper, AccountKeeper: bApp.AccountKeeper, DistrKeeper: bApp.DistrKeeper}
	require.NoError(t, v22.MigrateProvisionVaults(ctx, keepers))
	require.NoError(t, v22.MigrateLegacyReactorInfusions(ctx, keepers))
	infusion, found := bApp.StructsKeeper.GetInfusion(ctx, reactor.Id, primary.String())
	require.True(t, found)
	require.Equal(t, playerID, infusion.PlayerId)
	require.Equal(t, uint64(2000000), infusion.Fuel)
	require.Equal(t, uint64(1000000), infusion.Defusing)
	other, found := bApp.StructsKeeper.GetInfusion(ctx, reactor.Id, secondary.String())
	require.True(t, found)
	require.Equal(t, uint64(4000000), other.Fuel)
	require.Equal(t, playerCapacity, redelegationCapacity(t, bApp, ctx, playerID))
	queue := bApp.StructsKeeper.GetInfusionMaturitySweepQueueExport(ctx)
	require.NoError(t, v22.MigrateLegacyReactorInfusions(ctx, keepers))
	repeated, found := bApp.StructsKeeper.GetInfusion(ctx, reactor.Id, primary.String())
	require.True(t, found)
	require.Equal(t, infusion, repeated)
	require.Equal(t, playerCapacity, redelegationCapacity(t, bApp, ctx, playerID))
	require.Equal(t, queue, bApp.StructsKeeper.GetInfusionMaturitySweepQueueExport(ctx))
	ubd, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, primary, validator)
	require.NoError(t, err)
	ctx = ctx.WithBlockTime(ubd.Entries[0].CompletionTime.Add(time.Second))
	header := ctx.HeaderInfo()
	header.Time = ctx.BlockTime()
	ctx = ctx.WithHeaderInfo(header)
	_, err = bApp.StakingKeeper.DequeueAllMatureUBDQueue(ctx, ctx.BlockTime())
	require.NoError(t, err)
	paid, err := bApp.StakingKeeper.CompleteUnbonding(ctx, primary, validator)
	require.NoError(t, err)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(denom, 1000000)), paid)
	bApp.StructsKeeper.ProcessInfusionMaturitySweep(ctx)
	after, found := bApp.StructsKeeper.GetInfusion(ctx, reactor.Id, primary.String())
	require.True(t, found)
	require.Zero(t, after.Defusing)
	require.Equal(t, infusion.Fuel, after.Fuel)
	require.NotZero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, primary.String()))
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
