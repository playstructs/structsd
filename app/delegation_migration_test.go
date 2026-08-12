package app_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"structs/app"
	"structs/app/upgrades"
	v0_21_0 "structs/app/upgrades/v0_21_0"
)

// The two delegation migrations run against real staking and distribution
// state, which the keeper-test mocks cannot supply -- upgrades.Keepers holds the
// SDK's concrete keeper types -- so they are tested here rather than beside the
// other v0.21.0 migration tests.

func delegationUpgradeKeepers(bApp *app.App) *upgrades.Keepers {
	return &upgrades.Keepers{
		StructsKeeper: bApp.StructsKeeper,
		AccountKeeper: bApp.AccountKeeper,
		BankKeeper:    bApp.BankKeeper,
		StakingKeeper: bApp.StakingKeeper,
		DistrKeeper:   bApp.DistrKeeper,
	}
}

// rekeyLikeOldBinary reproduces exactly what the pre-v0.21.0 address sweep left
// on disk: the delegation is moved with the two bare staking calls, so the
// destination has no starting info and the source's record is abandoned rather
// than settled -- distribution's BeforeDelegationRemoved being a no-op, with the
// withdrawal living on BeforeDelegationSharesModified, which a removal does not
// fire.
func rekeyLikeOldBinary(t *testing.T, bApp *app.App, ctx sdk.Context, from, to sdk.AccAddress, valAddr sdk.ValAddress) {
	t.Helper()

	delegation, err := bApp.StakingKeeper.GetDelegation(ctx, from, valAddr)
	require.NoError(t, err)

	require.NoError(t, bApp.StakingKeeper.RemoveDelegation(ctx, delegation))

	delegation.DelegatorAddress = to.String()
	require.NoError(t, bApp.StakingKeeper.SetDelegation(ctx, delegation))

	abandoned, err := bApp.DistrKeeper.HasDelegatorStartingInfo(ctx, valAddr, from)
	require.NoError(t, err)
	require.True(t, abandoned, "the old rekey abandons the source record rather than settling it")

	orphanedDelegation, err := bApp.DistrKeeper.HasDelegatorStartingInfo(ctx, valAddr, to)
	require.NoError(t, err)
	require.False(t, orphanedDelegation, "and leaves the destination with none")
}

// TestMigrateDelegationDistributionStateRehomesTheAbandonedRecord covers the
// case a single address move leaves, which is the one the repair is for. The
// abandoned record and the stranded delegation are two halves of one move, so
// re-homing repairs both and prices the rewards from the period the stake was
// really delegated at.
func TestMigrateDelegationDistributionStateRehomesTheAbandonedRecord(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	keepers := delegationUpgradeKeepers(bApp)

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	valAddr := createRedelegationValidator(t, bApp, ctx, stakingMsgServer, bondDenom, "migratedistrepair")

	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	newAcc, oldAcc, _ := newTransferPlayer(t, bApp, ctx)

	// A second, untouched delegator on the same validator, to prove the repair
	// does not disturb the healthy rows around it.
	healthyAcc := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())

	delegate(t, bApp, ctx, stakingMsgServer, bondDenom, oldAcc, valAddr, math.NewInt(2_000_000))
	delegate(t, bApp, ctx, stakingMsgServer, bondDenom, healthyAcc, valAddr, math.NewInt(3_000_000))

	abandonedBefore, err := bApp.DistrKeeper.GetDelegatorStartingInfo(ctx, valAddr, oldAcc)
	require.NoError(t, err)
	healthyBefore, err := bApp.DistrKeeper.GetDelegatorStartingInfo(ctx, valAddr, healthyAcc)
	require.NoError(t, err)

	rekeyLikeOldBinary(t, bApp, ctx, oldAcc, newAcc, valAddr)

	distrMsgServer := distrkeeper.NewMsgServerImpl(bApp.DistrKeeper)
	_, err = distrMsgServer.WithdrawDelegatorReward(ctx, &distrtypes.MsgWithdrawDelegatorReward{
		DelegatorAddress: newAcc.String(),
		ValidatorAddress: valAddr.String(),
	})
	require.ErrorIs(t, err, distrtypes.ErrEmptyDelegationDistInfo,
		"precondition: the rekeyed delegation cannot touch its own rewards")

	require.NoError(t, v0_21_0.MigrateDelegationDistributionState(ctx, keepers))

	rehomed, err := bApp.DistrKeeper.GetDelegatorStartingInfo(ctx, valAddr, newAcc)
	require.NoError(t, err)
	require.Equal(t, abandonedBefore.PreviousPeriod, rehomed.PreviousPeriod,
		"the destination must inherit the period its stake was actually delegated at")
	require.Equal(t, abandonedBefore.Stake, rehomed.Stake)
	require.Equal(t, abandonedBefore.Height, rehomed.Height)

	stillAbandoned, err := bApp.DistrKeeper.HasDelegatorStartingInfo(ctx, valAddr, oldAcc)
	require.NoError(t, err)
	require.False(t, stillAbandoned, "and the record it came from is gone, not duplicated")

	_, err = distrMsgServer.WithdrawDelegatorReward(ctx, &distrtypes.MsgWithdrawDelegatorReward{
		DelegatorAddress: newAcc.String(),
		ValidatorAddress: valAddr.String(),
	})
	require.NoError(t, err, "the repaired delegation can withdraw")

	_, err = stakingMsgServer.Undelegate(ctx, stakingtypes.NewMsgUndelegate(
		newAcc.String(), valAddr.String(), sdk.NewCoin(bondDenom, math.NewInt(1_000_000))))
	require.NoError(t, err, "and undelegate")

	// Re-initializing a delegation that already has a period would move its
	// reward baseline forward and silently forfeit what it had accrued.
	healthyAfter, err := bApp.DistrKeeper.GetDelegatorStartingInfo(ctx, valAddr, healthyAcc)
	require.NoError(t, err)
	require.Equal(t, healthyBefore.PreviousPeriod, healthyAfter.PreviousPeriod)
	require.Equal(t, healthyBefore.Stake, healthyAfter.Stake)
	require.Equal(t, healthyBefore.Height, healthyAfter.Height)

	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

// TestMigrateDelegationDistributionStateFallsBackWhenAmbiguous covers two moves
// on one validator. Nothing on disk says which abandoned record belongs to
// which stranded delegation, so the repair must not guess: it initializes the
// delegations fresh -- which restores their usability, the point of the
// migration -- and leaves the abandoned records alone rather than deleting
// evidence it cannot yet act on.
func TestMigrateDelegationDistributionStateFallsBackWhenAmbiguous(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	keepers := delegationUpgradeKeepers(bApp)

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	valAddr := createRedelegationValidator(t, bApp, ctx, stakingMsgServer, bondDenom, "migratedistambig")

	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	firstNew, firstOld, _ := newTransferPlayer(t, bApp, ctx)
	secondNew, secondOld, _ := newTransferPlayer(t, bApp, ctx)

	delegate(t, bApp, ctx, stakingMsgServer, bondDenom, firstOld, valAddr, math.NewInt(2_000_000))
	delegate(t, bApp, ctx, stakingMsgServer, bondDenom, secondOld, valAddr, math.NewInt(3_000_000))

	rekeyLikeOldBinary(t, bApp, ctx, firstOld, firstNew, valAddr)
	rekeyLikeOldBinary(t, bApp, ctx, secondOld, secondNew, valAddr)

	require.NoError(t, v0_21_0.MigrateDelegationDistributionState(ctx, keepers))

	for _, delegator := range []sdk.AccAddress{firstNew, secondNew} {
		usable, err := bApp.DistrKeeper.HasDelegatorStartingInfo(ctx, valAddr, delegator)
		require.NoError(t, err)
		require.True(t, usable, "an ambiguous case must still leave a usable delegation")
	}

	for _, source := range []sdk.AccAddress{firstOld, secondOld} {
		left, err := bApp.DistrKeeper.HasDelegatorStartingInfo(ctx, valAddr, source)
		require.NoError(t, err)
		require.True(t, left,
			"the abandoned record stays: deleting it cannot decrement its historical-rewards reference and would destroy the evidence needed to re-home it later")
	}

	distrMsgServer := distrkeeper.NewMsgServerImpl(bApp.DistrKeeper)
	_, err = distrMsgServer.WithdrawDelegatorReward(ctx, &distrtypes.MsgWithdrawDelegatorReward{
		DelegatorAddress: firstNew.String(),
		ValidatorAddress: valAddr.String(),
	})
	require.NoError(t, err)

	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

// TestMigrateDelegationDistributionStateIsIdempotent covers a replayed upgrade
// block. The second pass must find nothing to do, and in particular must not
// re-initialize the delegations the first pass repaired.
func TestMigrateDelegationDistributionStateIsIdempotent(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	keepers := delegationUpgradeKeepers(bApp)

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	valAddr := createRedelegationValidator(t, bApp, ctx, stakingMsgServer, bondDenom, "migratedistidem")

	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	newAcc, oldAcc, _ := newTransferPlayer(t, bApp, ctx)
	delegate(t, bApp, ctx, stakingMsgServer, bondDenom, oldAcc, valAddr, math.NewInt(2_000_000))
	rekeyLikeOldBinary(t, bApp, ctx, oldAcc, newAcc, valAddr)

	require.NoError(t, v0_21_0.MigrateDelegationDistributionState(ctx, keepers))
	firstPass, err := bApp.DistrKeeper.GetDelegatorStartingInfo(ctx, valAddr, newAcc)
	require.NoError(t, err)

	require.NoError(t, v0_21_0.MigrateDelegationDistributionState(ctx, keepers))
	secondPass, err := bApp.DistrKeeper.GetDelegatorStartingInfo(ctx, valAddr, newAcc)
	require.NoError(t, err)

	require.Equal(t, firstPass.PreviousPeriod, secondPass.PreviousPeriod,
		"a second pass must not move the reward baseline it just established")
	require.Equal(t, firstPass.Stake, secondPass.Stake)

	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

// TestDelegationMigrationsOnUntouchedChain is the case every chain that never
// hit the bug will take: both migrations must run clean and write nothing.
func TestDelegationMigrationsOnUntouchedChain(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	keepers := delegationUpgradeKeepers(bApp)

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	valAddr := createRedelegationValidator(t, bApp, ctx, stakingMsgServer, bondDenom, "migrateuntouched")

	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	delegatorAcc, _, _ := newTransferPlayer(t, bApp, ctx)
	delegate(t, bApp, ctx, stakingMsgServer, bondDenom, delegatorAcc, valAddr, math.NewInt(2_000_000))

	before, err := bApp.DistrKeeper.GetDelegatorStartingInfo(ctx, valAddr, delegatorAcc)
	require.NoError(t, err)

	require.NoError(t, v0_21_0.MigrateDelegationDistributionState(ctx, keepers))
	require.NoError(t, v0_21_0.MigrateAuditDelegatorShares(ctx, keepers))

	after, err := bApp.DistrKeeper.GetDelegatorStartingInfo(ctx, valAddr, delegatorAcc)
	require.NoError(t, err)
	require.Equal(t, before.PreviousPeriod, after.PreviousPeriod)
	require.Equal(t, before.Stake, after.Stake)

	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

// TestMigrateAuditDelegatorSharesReportsWithoutRepairing pins the deliberate
// half of the audit. A share discrepancy is known per validator but not per
// delegator, so there is no way to repair it that does not either reprice every
// other delegation to that validator or mint one player's stake on a guess. The
// migration reports and leaves state alone.
func TestMigrateAuditDelegatorSharesReportsWithoutRepairing(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	keepers := delegationUpgradeKeepers(bApp)

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	valAddr := createRedelegationValidator(t, bApp, ctx, stakingMsgServer, bondDenom, "migrateaudit")

	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	delegatorAcc, _, _ := newTransferPlayer(t, bApp, ctx)
	delegate(t, bApp, ctx, stakingMsgServer, bondDenom, delegatorAcc, valAddr, math.NewInt(2_000_000))

	// What an overwriting rekey leaves: the validator counts shares that no
	// delegation holds any more.
	validator, err := bApp.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)
	orphaned := math.LegacyNewDecFromInt(math.NewInt(1_000_000))
	validator.DelegatorShares = validator.DelegatorShares.Add(orphaned)
	require.NoError(t, bApp.StakingKeeper.SetValidator(ctx, validator))

	require.NoError(t, v0_21_0.MigrateAuditDelegatorShares(ctx, keepers),
		"the audit reports, it does not fail the upgrade")

	unchanged, err := bApp.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, validator.DelegatorShares, unchanged.DelegatorShares,
		"the audit must not repair the validator's total")

	delegation, err := bApp.StakingKeeper.GetDelegation(ctx, delegatorAcc, valAddr)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecFromInt(math.NewInt(2_000_000)), delegation.Shares,
		"nor invent shares for a delegation to make the sum work")
}
