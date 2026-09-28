package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

func TestReactorRepairRetainsFailedPositionAndRetriesWithoutDoubleCredit(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	first, _, _ := newTransferPlayer(t, bApp, ctx)
	second, _, _ := newTransferPlayer(t, bApp, ctx)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, first, validator, math.NewInt(1000000))
	delegate(t, bApp, ctx, server, denom, second, validator, math.NewInt(1000000))
	id := jailGateReactorId(t, bApp, ctx, validator)
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)

	cc.GetInfusion(id, second.String()).SetRatio(0)
	cc.CommitAll()

	original, found := bApp.StructsKeeper.GetInfusion(ctx, id, first.String())
	require.True(t, found)
	broken := original
	broken.DestinationType = types.ObjectType(500)
	bApp.StructsKeeper.SetInfusion(ctx, broken)
	bApp.StructsKeeper.EnqueueReactorRestart(ctx, id)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	queue := bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx)
	require.Len(t, queue, 1, "an arithmetic error must not be reported as completed repair")
	require.Equal(t, first.String(), queue[0].Cursor)
	unchanged, found := bApp.StructsKeeper.GetInfusion(ctx, id, first.String())
	require.True(t, found)
	require.Equal(t, broken, unchanged)
	restored, found := bApp.StructsKeeper.GetInfusion(ctx, id, second.String())
	require.True(t, found)
	require.Equal(t, uint64(1000000), restored.Power, "other positions in the batch should still progress")
	beforeRetry := bApp.StructsKeeper.GetAllGridExport(ctx)
	// Repair the deliberately corrupted fixture, without changing its accounted
	// power, then let the same queue retry. Already restored rows owe no new delta.
	bApp.StructsKeeper.SetInfusion(ctx, original)
	beforeEvents := len(ctx.EventManager().Events())
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
	require.Equal(t, beforeRetry, bApp.StructsKeeper.GetAllGridExport(ctx))
	require.Equal(t, beforeEvents, len(ctx.EventManager().Events()))
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

func TestFailedDelegationHookQueuesRepairAndRetainsRetry(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	address, _, _ := newTransferPlayer(t, bApp, ctx)
	delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, address, validator, math.NewInt(1000000))
	id := jailGateReactorId(t, bApp, ctx, validator)

	original, found := bApp.StructsKeeper.GetInfusion(ctx, id, address.String())
	require.True(t, found)
	broken := original
	broken.DestinationType = types.ObjectType(500)
	bApp.StructsKeeper.SetInfusion(ctx, broken)
	require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
	require.NoError(t, bApp.StructsKeeper.Hooks().AfterDelegationModified(ctx, address, validator))
	require.Len(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx), 1)
	for i := 0; i < 2; i++ {
		bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
		require.Len(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx), 1)
		actual, _ := bApp.StructsKeeper.GetInfusion(ctx, id, address.String())
		require.Equal(t, broken, actual)
	}
	bApp.StructsKeeper.SetInfusion(ctx, original)
	before := bApp.StructsKeeper.GetAllGridExport(ctx)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
	require.Equal(t, before, bApp.StructsKeeper.GetAllGridExport(ctx))
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	require.Equal(t, before, bApp.StructsKeeper.GetAllGridExport(ctx))
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

func TestFailedHookDoesNotRewindActiveRepair(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	addresses := make([]sdk.AccAddress, 2*types.ReactorSlashReconcileBudget)
	for i := range addresses {
		addresses[i], _, _ = newTransferPlayer(t, bApp, ctx)
		delegate(t, bApp, ctx, server, denom, addresses[i], validator, math.NewInt(1000000))
	}
	id := jailGateReactorId(t, bApp, ctx, validator)
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)

	cc.GetInfusion(id, addresses[len(addresses)-1].String()).SetRatio(0)
	cc.CommitAll()

	original, found := bApp.StructsKeeper.GetInfusion(ctx, id, addresses[0].String())
	require.True(t, found)
	broken := original
	broken.DestinationType = types.ObjectType(500)
	bApp.StructsKeeper.SetInfusion(ctx, broken)
	bApp.StructsKeeper.EnqueueReactorRestart(ctx, id)
	for pass := 0; pass < 4; pass++ {
		before, found, err := bApp.StructsKeeper.GetReactorRepairSnapshot(ctx, id)
		require.NoError(t, err)
		require.True(t, found)
		for attempt := 0; attempt < 3; attempt++ {
			require.NoError(t, bApp.StructsKeeper.Hooks().AfterDelegationModified(ctx, addresses[0], validator))
		}
		after, _, err := bApp.StructsKeeper.GetReactorRepairSnapshot(ctx, id)
		require.NoError(t, err)
		require.Equal(t, before, after, "failed hooks must not rewind or extend the finite pass")
		bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	}
	last, found := bApp.StructsKeeper.GetInfusion(ctx, id, addresses[len(addresses)-1].String())
	require.True(t, found)
	require.Equal(t, uint64(1000000), last.Power, "later delegators still receive their repairs")
	require.Len(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx), 1)
	bApp.StructsKeeper.SetInfusion(ctx, original)
	before := bApp.StructsKeeper.GetAllGridExport(ctx)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
	require.Equal(t, before, bApp.StructsKeeper.GetAllGridExport(ctx), "retry must not credit successful rows twice")
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
