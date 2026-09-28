package app_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
)

func TestMigratedHeldUnbondingClearsDefusingAfterLateRelease(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	primary, _, _ := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(2000000))
	withdrawal, err := server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(primary.String(), validator.String(), sdk.NewInt64Coin(denom, 1000000)))
	require.NoError(t, err)
	pending, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, primary, validator)
	require.NoError(t, err)
	require.Len(t, pending.Entries, 1)
	id := pending.Entries[0].UnbondingId
	require.NoError(t, bApp.StakingKeeper.PutUnbondingOnHold(ctx, id))
	require.NoError(t, bApp.StakingKeeper.PutUnbondingOnHold(ctx, id))
	keepers := delegationUpgradeKeepers(bApp)
	require.NoError(t, v22.MigrateVaultAssets(ctx, keepers))
	require.NoError(t, v22.MigrateReactorPower(ctx, keepers))

	vault := primary // Pending refunds stay at their original address.
	reactorID := jailGateReactorId(t, bApp, ctx, validator)
	held, err := bApp.StakingKeeper.GetUnbondingDelegationByUnbondingID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, vault.String(), held.DelegatorAddress)
	require.Equal(t, int64(2), held.Entries[0].UnbondingOnHoldRefCount)
	beforeBalance := bApp.BankKeeper.GetBalance(ctx, vault, denom)
	maturityTime := withdrawal.CompletionTime.Add(time.Second)
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: maturityTime})
	require.NoError(t, err)
	// Release through the actual SDK API after Structs has processed this
	// block's maturity queue, modeling an external module acting afterward.
	// This is keeper lifecycle coverage, not a signed hold-release transaction.
	releaseCtx := bApp.NewContextLegacy(false, cmtproto.Header{Height: 1, ChainID: jailGateChainID, Time: maturityTime})
	require.Equal(t, beforeBalance, bApp.BankKeeper.GetBalance(releaseCtx, vault, denom))
	row, found := bApp.StructsKeeper.GetInfusion(releaseCtx, reactorID, vault.String())
	require.True(t, found)
	require.Equal(t, uint64(1000000), row.Defusing)
	require.Len(t, bApp.StructsKeeper.GetInfusionMaturitySweepQueueExport(releaseCtx), 1)
	_, err = bApp.Commit()
	require.NoError(t, err)
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: maturityTime.Add(time.Second)})
	require.NoError(t, err)
	releaseCtx = bApp.NewContextLegacy(false, cmtproto.Header{Height: 2, ChainID: jailGateChainID, Time: maturityTime.Add(time.Second)})
	require.Len(t, bApp.StructsKeeper.GetInfusionMaturitySweepQueueExport(releaseCtx), 1, "repeated held checks replace their retry rather than accumulate rows")
	releaseCtx = releaseCtx.WithEventManager(sdk.NewEventManager())
	pool := bApp.AccountKeeper.GetModuleAddress(stakingtypes.NotBondedPoolName)
	poolBefore := bApp.BankKeeper.GetBalance(releaseCtx, pool, denom)
	require.NoError(t, bApp.StakingKeeper.UnbondingCanComplete(releaseCtx, id))
	require.Equal(t, beforeBalance, bApp.BankKeeper.GetBalance(releaseCtx, vault, denom), "one hold still prevents payment")
	require.Empty(t, releaseCtx.EventManager().Events(), "removing a nonfinal hold must not announce payment")
	require.NoError(t, bApp.StakingKeeper.UnbondingCanComplete(releaseCtx, id))
	refund := sdk.NewInt64Coin(denom, 1000000)
	expectedBalance := beforeBalance.Add(refund)
	require.Equal(t, expectedBalance, bApp.BankKeeper.GetBalance(releaseCtx, vault, denom))
	require.True(t, poolBefore.Sub(refund).IsEqual(bApp.BankKeeper.GetBalance(releaseCtx, pool, denom)))
	// Hold release pays through the bank keeper directly, without the staking
	// EndBlock completion event or a bank transfer summary. The public bank
	// events must describe exactly the pool debit and vault credit that occurred.
	wantEvents := sdk.Events{
		banktypes.NewCoinSpentEvent(pool, sdk.NewCoins(refund)),
		banktypes.NewCoinReceivedEvent(vault, sdk.NewCoins(refund)),
	}
	require.Equal(t, wantEvents.ToABCIEvents(), releaseCtx.EventManager().ABCIEvents())
	_, err = bApp.StakingKeeper.GetUnbondingDelegationByUnbondingID(releaseCtx, id)
	require.Error(t, err)
	require.Error(t, bApp.StakingKeeper.UnbondingCanComplete(releaseCtx, id))
	require.Equal(t, wantEvents.ToABCIEvents(), releaseCtx.EventManager().ABCIEvents(), "a repeated release must not emit another payment")
	// FinalizeBlock already flushed its branch before the external-module
	// call above. Persist that later keeper mutation before committing.
	bApp.SimWriteState()
	_, err = bApp.Commit()
	require.NoError(t, err)
	reconciled, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 3, Time: maturityTime.Add(2*time.Second)})
	require.NoError(t, err)
	for _, event := range reconciled.Events {
		if event.Type != banktypes.EventTypeCoinReceived {
			continue
		}
		for _, attribute := range event.Attributes {
			if attribute.Key == banktypes.AttributeKeyReceiver {
				require.NotEqual(t, vault.String(), attribute.Value, "maturity reconciliation must not announce a second vault credit")
			}
		}
	}
	_, err = bApp.Commit()
	require.NoError(t, err)
	afterCtx := bApp.NewContextLegacy(true, cmtproto.Header{Height: 3, ChainID: jailGateChainID, Time: maturityTime.Add(2*time.Second)})
	after, found := bApp.StructsKeeper.GetInfusion(afterCtx, reactorID, vault.String())
	require.True(t, found)
	require.Zero(t, after.Defusing, "a hold released after the scheduled sweep must not leave permanent phantom defusing")
	require.Equal(t, row.Fuel, after.Fuel)
	require.Equal(t, row.Power, after.Power)

	require.Equal(t, expectedBalance, bApp.BankKeeper.GetBalance(afterCtx, vault, denom), "reconciliation must not pay again")
	require.Empty(t, bApp.StructsKeeper.GetInfusionMaturitySweepQueueExport(afterCtx))
	requireStakingAndDistributionHealthy(t, bApp, afterCtx)
}
