package app_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	v22 "structs/app/upgrades/v0_22_0"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Exercise migration from the SDK's post-tombstone state. Evidence submission
// and its slash amount are outside this fixture; jail and signing-info state
// are installed through real keepers, not by editing validator records.
func TestTombstonedValidatorVaultMigrationPreservesWithdrawalWithoutPower(t *testing.T) {
	testTombstonedVaultMigration(t, nil, false)
}

func TestMaturityBacklogDoesNotDelayNativeVaultRefund(t *testing.T) {
	testTombstonedVaultMigration(t, nil, true)
}

func TestTombstonedVaultPendingRefundSurvivesExportAndRestart(t *testing.T) {
	for _, zeroHeight := range []bool{false, true} {
		name := "preserve-height"
		if zeroHeight {
			name = "zero-height"
		}
		t.Run(name, func(t *testing.T) { testTombstonedVaultMigration(t, &zeroHeight, false) })
	}
}

func testTombstonedVaultMigration(t *testing.T, restartAtZeroHeight *bool, maturityBacklog bool) {
	t.Helper()
	bApp, ctx := setupJailGateApp(t)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	validator := createRedelegationValidator(t, bApp, ctx, staking, denom, "tombstonemigration")
	// A real bonding transition initializes slashing signing information;
	// the fixture's preinstalled genesis validators do not have that history.
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	bonded, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	require.True(t, bonded.IsBonded())
	consensus, err := bonded.GetConsAddr()
	require.NoError(t, err)
	primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
	delegate(t, bApp, ctx, staking, denom, primary, validator, math.NewInt(3000000))
	delegate(t, bApp, ctx, staking, denom, secondary, validator, math.NewInt(4000000))
	first, err := bApp.StakingKeeper.GetDelegation(ctx, primary, validator)
	require.NoError(t, err)
	second, err := bApp.StakingKeeper.GetDelegation(ctx, secondary, validator)
	require.NoError(t, err)
	require.NoError(t, bApp.SlashingKeeper.Tombstone(ctx, consensus))
	require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	beforeValidator, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	selfStake, err := bApp.StakingKeeper.GetDelegation(ctx, sdk.AccAddress(validator), validator)
	require.NoError(t, err)
	keepers := delegationUpgradeKeepers(bApp)
	require.NoError(t, v22.MigrateVaultAssets(ctx, keepers))
	require.NoError(t, v22.MigrateReactorPower(ctx, keepers))

	vault, err := types.VaultAddress(playerID)
	require.NoError(t, err)
	for i, address := range []sdk.AccAddress{primary, secondary} {
		retained, err := bApp.StakingKeeper.GetDelegation(ctx, address, validator)
		require.NoError(t, err)
		require.Equal(t, []stakingtypes.Delegation{first, second}[i], retained)
	}
	_, err = bApp.StakingKeeper.GetDelegation(ctx, vault, validator)
	require.ErrorIs(t, err, stakingtypes.ErrNoDelegation)
	afterValidator, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	require.Equal(t, beforeValidator, afterValidator)
	retainedSelfStake, err := bApp.StakingKeeper.GetDelegation(ctx, sdk.AccAddress(validator), validator)
	require.NoError(t, err)
	require.Equal(t, selfStake, retainedSelfStake, "operator self-stake remains at its native address")
	require.True(t, bApp.SlashingKeeper.IsTombstoned(ctx, consensus))
	reactorID := jailGateReactorId(t, bApp, ctx, validator)
	row, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
	require.True(t, found)
	require.Equal(t, uint64(3000000), row.Fuel)

	require.Zero(t, row.Power)
	require.Zero(t, redelegationCapacity(t, bApp, ctx, playerID))
	_, err = slashingkeeper.NewMsgServerImpl(bApp.SlashingKeeper).Unjail(ctx, &slashingtypes.MsgUnjail{ValidatorAddr: validator.String()})
	require.ErrorIs(t, err, slashingtypes.ErrValidatorJailed)
	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	_, err = ms.ReactorRestart(ctx, &types.MsgReactorRestart{Creator: primary.String(), ValidatorAddress: validator.String()})
	require.NoError(t, err)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	afterRestart, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
	require.True(t, found)
	require.Equal(t, row, afterRestart, "restart cannot revive a tombstoned validator's output")
	_, err = bApp.DistrKeeper.WithdrawDelegationRewards(ctx, primary, validator)
	require.NoError(t, err, "the migrated distribution starting info remains usable")
	beforeBalance := bApp.BankKeeper.GetBalance(ctx, primary, denom)
	withdrawal, err := ms.ReactorDefuse(ctx, &types.MsgReactorDefuse{
		Creator: primary.String(), DelegatorAddress: primary.String(), ValidatorAddress: validator.String(), Amount: sdk.NewInt64Coin(denom, 3000000),
	})
	require.NoError(t, err, "the player's key can withdraw its migrated vault stake")
	require.Equal(t, sdk.NewInt64Coin(denom, 3000000), withdrawal.Amount)
	row, found = bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
	require.True(t, found)
	require.Zero(t, row.Fuel)
	require.Zero(t, row.Power)
	require.Equal(t, uint64(3000000), row.Defusing)
	if restartAtZeroHeight != nil {
		pending, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, primary, validator)
		require.NoError(t, err)
		require.Len(t, pending.Entries, 1)
		accountNumber := bApp.AccountKeeper.GetAccount(ctx, vault).GetAccountNumber()
		_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
		require.NoError(t, err)
		_, err = bApp.Commit()
		require.NoError(t, err)
		exported, err := bApp.ExportAppStateAndValidators(*restartAtZeroHeight, nil, nil)
		require.NoError(t, err)
		restarted, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true,
			simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}, baseapp.SetChainID(jailGateChainID))
		require.NoError(t, err)
		restartTime := ctx.BlockTime().Add(time.Second)
		_, err = restarted.InitChain(&abci.RequestInitChain{
			AppStateBytes: exported.AppState, ChainId: jailGateChainID,
			InitialHeight: exported.Height, Time: restartTime, ConsensusParams: &exported.ConsensusParams,
		})
		require.NoError(t, err)
		height := exported.Height
		if height == 0 {
			height = 1
		}
		bApp = restarted
		ctx = bApp.NewContextLegacy(false, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: restartTime})

		require.True(t, bApp.SlashingKeeper.IsTombstoned(ctx, consensus))
		owner, found := bApp.StructsKeeper.GetVaultOwner(ctx, vault)
		require.True(t, found)
		require.Equal(t, playerID, owner)
		require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, vault.String()))
		require.Equal(t, accountNumber, bApp.AccountKeeper.GetAccount(ctx, vault).GetAccountNumber())
		restored, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
		require.True(t, found)
		require.Equal(t, row, restored)
		restoredPending, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, primary, validator)
		require.NoError(t, err)
		require.Len(t, restoredPending.Entries, 1)
		// Zero-height export rebases creation heights, not refund amounts,
		// maturity times or native unbonding IDs.
		require.Equal(t, pending.Entries[0].Balance, restoredPending.Entries[0].Balance)
		require.Equal(t, pending.Entries[0].InitialBalance, restoredPending.Entries[0].InitialBalance)
		require.Equal(t, pending.Entries[0].CompletionTime, restoredPending.Entries[0].CompletionTime)
		require.Equal(t, pending.Entries[0].UnbondingId, restoredPending.Entries[0].UnbondingId)
		_, err = slashingkeeper.NewMsgServerImpl(bApp.SlashingKeeper).Unjail(ctx, &slashingtypes.MsgUnjail{ValidatorAddr: validator.String()})
		require.ErrorIs(t, err, slashingtypes.ErrValidatorJailed)
		_, err = keeperlib.NewMsgServerImpl(bApp.StructsKeeper).ReactorRestart(ctx, &types.MsgReactorRestart{Creator: primary.String(), ValidatorAddress: validator.String()})
		require.NoError(t, err)
		bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
		require.Zero(t, redelegationCapacity(t, bApp, ctx, playerID))
		requireStakingAndDistributionHealthy(t, bApp, ctx)
	}
	if maturityBacklog {
		// Stale queue entries can outlive their infusions. They must consume
		// the row budget too, while native bank refunds continue independently.
		for i := 0; i <= types.InfusionMaturitySweepBudget; i++ {
			bApp.StructsKeeper.EnqueueInfusionMaturitySweep(ctx, ctx.HeaderInfo().Time.Add(-time.Second), fmt.Sprintf("missing-%06d", i))
		}
	}
	maturityHeight := int64(1)
	if restartAtZeroHeight != nil {
		maturityHeight = ctx.BlockHeight()
	}
	maturityTime := withdrawal.CompletionTime.Add(time.Second)
	maturityBlock, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: maturityHeight, Time: maturityTime})
	require.NoError(t, err)
	refundEvents := 0
	completionEvents := 0
	for _, event := range maturityBlock.Events {
		if event.Type != "coin_received" && event.Type != "complete_unbonding" {
			continue
		}
		attributes := map[string]string{}
		for _, attribute := range event.Attributes {
			attributes[attribute.Key] = attribute.Value
		}
		if event.Type == "coin_received" && attributes["receiver"] == primary.String() {
			require.Equal(t, withdrawal.Amount.String(), attributes["amount"])
			refundEvents++
		}
		if event.Type == "complete_unbonding" && attributes["delegator"] == primary.String() {
			require.Equal(t, validator.String(), attributes["validator"])
			require.Equal(t, withdrawal.Amount.String(), attributes["amount"])
			completionEvents++
		}
	}
	require.Equal(t, 1, refundEvents, "one public bank credit describes the matured vault refund")
	require.Equal(t, 1, completionEvents, "one native completion event identifies the vault and validator")
	_, err = bApp.Commit()
	require.NoError(t, err)
	ctx = bApp.NewContextLegacy(true, cmtproto.Header{Height: maturityHeight, ChainID: jailGateChainID, Time: maturityTime})
	require.Equal(t, beforeBalance.Add(withdrawal.Amount), bApp.BankKeeper.GetBalance(ctx, primary, denom))
	require.True(t, bApp.BankKeeper.GetBalance(ctx, vault, denom).IsZero(), "no automatic refund sweep")
	require.True(t, bApp.BankKeeper.GetBalance(ctx, secondary, denom).IsZero())
	_, err = bApp.StakingKeeper.GetUnbondingDelegation(ctx, primary, validator)
	require.Error(t, err)
	if maturityBacklog {
		delayed, exists := bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
		require.True(t, exists)
		require.Equal(t, uint64(3000000), delayed.Defusing, "bookkeeping waits behind the bounded batch")
		require.Zero(t, delayed.Fuel)
		require.Zero(t, delayed.Power)
		nextTime := maturityTime.Add(time.Second)
		_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: maturityHeight + 1, Time: nextTime})
		require.NoError(t, err)
		_, err = bApp.Commit()
		require.NoError(t, err)
		ctx = bApp.NewContextLegacy(true, cmtproto.Header{Height: maturityHeight + 1, ChainID: jailGateChainID, Time: nextTime})
		require.Equal(t, beforeBalance.Add(withdrawal.Amount), bApp.BankKeeper.GetBalance(ctx, primary, denom), "later bookkeeping cannot refund twice")
		require.Empty(t, bApp.StructsKeeper.GetInfusionMaturitySweepQueueExport(ctx))
	}
	row, found = bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
	if found {
		require.Zero(t, row.Fuel)
		require.Zero(t, row.Defusing)
		require.Zero(t, row.Power)
	}
	require.True(t, bApp.SlashingKeeper.IsTombstoned(ctx, consensus))
	require.Zero(t, redelegationCapacity(t, bApp, ctx, playerID))
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
