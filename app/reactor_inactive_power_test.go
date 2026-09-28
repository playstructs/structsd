package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
)

func TestInactiveDepositsProduceFlatPowerOnlyWhenEligible(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 5)
	primary, _, _ := newTransferPlayer(t, bApp, ctx)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	// The fixture seeds 1M per validator; raise them above this new validator's
	// 10M self-stake so reducing MaxValidators later removes it.
	seeds, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	for _, seed := range seeds {
		address, err := sdk.ValAddressFromBech32(seed.OperatorAddress)
		require.NoError(t, err)
		delegate(t, bApp, ctx, staking, denom, primary, address, math.NewInt(20000000))
	}
	validator := createRedelegationValidator(t, bApp, ctx, staking, denom, "inactiveentry")
	delegate(t, bApp, ctx, staking, denom, primary, validator, math.NewInt(1000000))
	reactorID := jailGateReactorId(t, bApp, ctx, validator)
	check := func(fuel, power uint64) {
		t.Helper()
		row, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
		require.True(t, found)
		require.Equal(t, fuel, row.Fuel)
		require.Equal(t, power, row.Power)

	}
	check(1000000, 0)
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	check(1000000, 0)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	check(1000000, 1000000)

	delegate(t, bApp, ctx, staking, denom, primary, validator, math.NewInt(1000000))
	check(2000000, 2000000)
	value, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	consensus, err := value.GetConsAddr()
	require.NoError(t, err)
	require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
	// Deposits while jailed retain fuel but cannot restore power.
	delegate(t, bApp, ctx, staking, denom, primary, validator, math.NewInt(1000000))
	check(3000000, 0)
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	require.NoError(t, bApp.StakingKeeper.Unjail(ctx, consensus))
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	check(3000000, 0)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	check(3000000, 3000000)
	// Leaving the active set without jail must also stop output. Deposits made
	// outside the cutoff retain fuel but produce no power.
	params, err := bApp.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.MaxValidators = 5
	require.NoError(t, bApp.StakingKeeper.SetParams(ctx, params))
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	check(3000000, 3000000) // departure queues stored cleanup
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	check(3000000, 0)
	delegate(t, bApp, ctx, staking, denom, primary, validator, math.NewInt(1000000))
	check(4000000, 0)
	params.MaxValidators = 6
	require.NoError(t, bApp.StakingKeeper.SetParams(ctx, params))
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	check(4000000, 0)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	check(4000000, 4000000)
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

// A top-up must retain existing backing and credit exactly the new fuel.
func TestFlatTopUpPreservesExistingFuel(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	primary, _, _ := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)

	delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(1000000))
	reactorID := jailGateReactorId(t, bApp, ctx, validator)
	before, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
	require.True(t, found)

	require.Equal(t, uint64(1000000), before.Power)

	delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(1000000))
	after, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
	require.True(t, found)
	require.Equal(t, uint64(2000000), after.Fuel)

	require.Equal(t, uint64(2000000), after.Power)
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
