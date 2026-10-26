package app_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

func TestVaultBackingSurvivesRealAppExportAndInitChain(t *testing.T) {
	for _, inactive := range []bool{false, true} {
		name := "bonded"
		if inactive {
			name = "inactive"
		}
		t.Run(name, func(t *testing.T) {
			t.Run("preserve-height", func(t *testing.T) { testVaultBackingRestart(t, false, inactive, false) })
			t.Run("zero-height", func(t *testing.T) { testVaultBackingRestart(t, true, inactive, false) })
		})
	}
}

func TestV22HandlerExportAndRestart(t *testing.T) {
	for _, inactive := range []bool{false, true} {
		name := "bonded"
		if inactive {
			name = "inactive"
		}
		t.Run(name, func(t *testing.T) {
			t.Run("preserve-height", func(t *testing.T) { testVaultBackingRestart(t, false, inactive, true) })
			t.Run("zero-height", func(t *testing.T) { testVaultBackingRestart(t, true, inactive, true) })
		})
	}
}

func testVaultBackingRestart(t *testing.T, zeroHeight, inactive, namedHandler bool) {
	t.Helper()
	bApp, ctx := setupJailGateApp(t)
	primary, _, playerID := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(2000000))
	keepers := delegationUpgradeKeepers(bApp)
	vault, err := types.VaultAddress(playerID)
	require.NoError(t, err)
	reactorID := jailGateReactorId(t, bApp, ctx, validator)
	if namedHandler {
		restoreLegacyCascadeQueueFixture(t, bApp, ctx)
		handler := v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), keepers)
		_, err := handler(ctx, upgradetypes.Plan{Name: v22.UpgradeName}, bApp.ModuleManager.GetVersionMap())
		require.NoError(t, err)

	} else {
		require.NoError(t, v22.MigrateProvisionVaults(ctx, keepers))
		require.NoError(t, v22.MigrateVaultAssets(ctx, keepers))
	}

	_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(primary.String(), validator.String(), sdk.NewInt64Coin(denom, 500000)))
	require.NoError(t, err)
	before, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
	require.True(t, found)
	require.Equal(t, uint64(1500000), before.Power)
	capacityBefore := redelegationCapacity(t, bApp, ctx, playerID)
	accountNumber := bApp.AccountKeeper.GetAccount(ctx, vault).GetAccountNumber()

	if inactive {
		// A larger validator displaces this reactor without jail. Keep its
		// outstanding unbonding, backing and account identity across restart.
		createRedelegationValidator(t, bApp, ctx, server, denom, "restartcutoff")
		params, err := bApp.StakingKeeper.GetParams(ctx)
		require.NoError(t, err)
		params.MaxValidators = 1
		require.NoError(t, bApp.StakingKeeper.SetParams(ctx, params))
		_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
		require.NoError(t, err)

		before, found = bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
		require.True(t, found)
		require.Positive(t, before.Power, "departure queues cleanup without rewriting the position")

		capacityBefore = redelegationCapacity(t, bApp, ctx, playerID)
		require.Positive(t, capacityBefore)
		require.False(t, bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayer(playerID).IsOnline())
	}
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	if inactive {
		committed := bApp.NewContext(true).WithBlockTime(ctx.BlockTime())
		before, found = bApp.StructsKeeper.GetInfusion(committed, reactorID, primary.String())
		require.True(t, found)
		require.Zero(t, before.Power, "the bounded worker reached this small fixture in the completed block")
		capacityBefore = redelegationCapacity(t, bApp, committed, playerID)
		require.Zero(t, capacityBefore)
	}
	exported, err := bApp.ExportAppStateAndValidators(zeroHeight, nil, nil)
	require.NoError(t, err)
	options := simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}
	restarted, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, options, baseapp.SetChainID(jailGateChainID))
	require.NoError(t, err)
	restartTime := ctx.BlockTime().Add(time.Second)
	_, err = restarted.InitChain(&abci.RequestInitChain{AppStateBytes: exported.AppState, ChainId: jailGateChainID, InitialHeight: exported.Height, Time: restartTime, ConsensusParams: &exported.ConsensusParams})
	require.NoError(t, err)
	restartHeight := exported.Height
	if restartHeight == 0 {
		restartHeight = 1
	}
	restoredCtx := restarted.NewContextLegacy(false, cmtproto.Header{Height: restartHeight, ChainID: jailGateChainID, Time: restartTime})

	after, found := restarted.StructsKeeper.GetInfusion(restoredCtx, reactorID, primary.String())
	require.True(t, found)
	require.Equal(t, before, after)
	require.Equal(t, capacityBefore, redelegationCapacity(t, restarted, restoredCtx, playerID))
	require.Equal(t, accountNumber, restarted.AccountKeeper.GetAccount(restoredCtx, vault).GetAccountNumber())
	require.Zero(t, restarted.StructsKeeper.GetPlayerIndexFromAddress(restoredCtx, vault.String()))
	owner, found := restarted.StructsKeeper.GetVaultOwner(restoredCtx, vault)
	require.True(t, found)
	require.Equal(t, playerID, owner)
	_, err = restarted.DistrKeeper.WithdrawDelegationRewards(restoredCtx, primary, validator)
	require.NoError(t, err)
	restartedServer := stakingkeeper.NewMsgServerImpl(restarted.StakingKeeper)
	_, err = restartedServer.Undelegate(restoredCtx, stakingtypes.NewMsgUndelegate(primary.String(), validator.String(), sdk.NewInt64Coin(denom, 100000)))
	require.NoError(t, err)
	continued, found := restarted.StructsKeeper.GetInfusion(restoredCtx, reactorID, primary.String())
	require.True(t, found)
	if inactive {
		require.Zero(t, continued.Power)
	} else {
		require.Equal(t, uint64(1400000), continued.Power)
	}
	require.Equal(t, uint64(600000), continued.Defusing)
	delegate(t, restarted, restoredCtx, restartedServer, denom, primary, validator, math.NewInt(100000))
	continued, found = restarted.StructsKeeper.GetInfusion(restoredCtx, reactorID, primary.String())
	require.True(t, found)
	require.Equal(t, uint64(1500000), continued.Fuel)
	if inactive {
		require.Zero(t, continued.Power)

		params, err := restarted.StakingKeeper.GetParams(restoredCtx)
		require.NoError(t, err)
		params.MaxValidators = 2
		require.NoError(t, restarted.StakingKeeper.SetParams(restoredCtx, params))
		_, err = restarted.StakingKeeper.ApplyAndReturnValidatorSetUpdates(restoredCtx)
		require.NoError(t, err)
		continued, found = restarted.StructsKeeper.GetInfusion(restoredCtx, reactorID, primary.String())
		require.True(t, found)
		require.Zero(t, continued.Power, "restoration waits for bounded reconciliation after rebond")
		restarted.StructsKeeper.ProcessReactorSlashReconcileQueue(restoredCtx)
		continued, found = restarted.StructsKeeper.GetInfusion(restoredCtx, reactorID, primary.String())
		require.True(t, found)
	}
	require.Equal(t, uint64(1500000), continued.Power, "all live backing contributes flat power")
	requireStakingAndDistributionHealthy(t, restarted, restoredCtx)
}
