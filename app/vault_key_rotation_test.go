package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestPlayerKeyRotationPreservesVaultStakeAndRevokesAccess(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	// Rotation must also preserve the independent vault position while native
	// primary-address stake moves through staking/distribution hook lifecycles.
	delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, primary, validator, math.NewInt(2000000))
	primaryStake, err := bApp.StakingKeeper.GetDelegation(ctx, primary, validator)
	require.NoError(t, err)
	vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(playerID)
	require.NoError(t, err)
	vault := vaults[0]
	fundRedelegationAccount(t, bApp, ctx, vault, sdk.NewCoins(sdk.NewInt64Coin(denom, 500000)))
	server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	infuse := func(signer string, amount int64) error {
		_, err := server.ReactorInfuse(ctx, &types.MsgReactorInfuse{Creator: signer, ValidatorAddress: validator.String(), Amount: sdk.NewInt64Coin(denom, amount)})
		return err
	}
	require.NoError(t, infuse(primary.String(), 200000))
	before, err := bApp.StakingKeeper.GetDelegation(ctx, vault, validator)
	require.NoError(t, err)
	number := bApp.AccountKeeper.GetAccount(ctx, vault).GetAccountNumber()
	capacityBefore := redelegationCapacity(t, bApp, ctx, playerID)
	_, err = server.PlayerUpdatePrimaryAddress(ctx, &types.MsgPlayerUpdatePrimaryAddress{Creator: primary.String(), PrimaryAddress: secondary.String()})
	require.NoError(t, err)
	moved, err := bApp.StakingKeeper.GetDelegation(ctx, vault, validator)
	require.NoError(t, err)
	require.Equal(t, before.Shares.Add(primaryStake.Shares), moved.Shares)
	_, err = server.AddressRevoke(ctx, &types.MsgAddressRevoke{Creator: secondary.String(), Address: primary.String()})
	require.NoError(t, err)
	owner, found := bApp.StructsKeeper.GetVaultOwner(ctx, vault)
	require.True(t, found)
	require.Equal(t, playerID, owner)
	require.Equal(t, number, bApp.AccountKeeper.GetAccount(ctx, vault).GetAccountNumber())
	after, err := bApp.StakingKeeper.GetDelegation(ctx, vault, validator)
	require.NoError(t, err)
	require.Equal(t, moved, after, "revoking the old key does not rekey the merged vault delegation")
	require.Equal(t, capacityBefore, redelegationCapacity(t, bApp, ctx, playerID))
	require.Equal(t, int64(300000), bApp.BankKeeper.GetBalance(ctx, vault, denom).Amount.Int64())
	require.Error(t, infuse(primary.String(), 100000), "the revoked key cannot spend vault assets")
	require.Equal(t, int64(300000), bApp.BankKeeper.GetBalance(ctx, vault, denom).Amount.Int64())
	require.NoError(t, infuse(secondary.String(), 100000))
	require.Equal(t, int64(200000), bApp.BankKeeper.GetBalance(ctx, vault, denom).Amount.Int64())
	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, vault.String()))
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

func TestPrimaryRotationPreservesBackingWhenMergingIntoVault(t *testing.T) {
	t.Run("bonded", func(t *testing.T) { testPrimaryRotationPreservesBacking(t, false) })
	t.Run("inactive", func(t *testing.T) { testPrimaryRotationPreservesBacking(t, true) })
}

func testPrimaryRotationPreservesBacking(t *testing.T, inactive bool) {
	t.Helper()
	bApp, ctx := setupJailGateApp(t)
	primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, staking, denom, primary, validator, math.NewInt(1000000))
	vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(playerID)
	require.NoError(t, err)
	vault := vaults[0]
	delegate(t, bApp, ctx, staking, denom, vault, validator, math.NewInt(1000000))
	reactorID := jailGateReactorId(t, bApp, ctx, validator)

	if inactive {
		createRedelegationValidator(t, bApp, ctx, staking, denom, "rotationcutoff")
		params, err := bApp.StakingKeeper.GetParams(ctx)
		require.NoError(t, err)
		params.MaxValidators = 1
		require.NoError(t, bApp.StakingKeeper.SetParams(ctx, params))
		_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
		require.NoError(t, err)
	}
	before := redelegationCapacity(t, bApp, ctx, playerID)
	server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	_, err = server.PlayerUpdatePrimaryAddress(ctx, &types.MsgPlayerUpdatePrimaryAddress{Creator: primary.String(), PrimaryAddress: secondary.String()})
	require.NoError(t, err)
	merged, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, vault.String())
	require.True(t, found)
	require.Equal(t, uint64(2000000), merged.Fuel)

	if inactive {
		require.Zero(t, merged.Power)
		require.Positive(t, before, "departure left stored power pending cleanup")
	} else {
		require.Equal(t, uint64(2000000), merged.Power)
	}
	if inactive {
		require.Zero(t, redelegationCapacity(t, bApp, ctx, playerID), "rotation reconciles both positions while the reactor is inactive")
	} else {
		require.Equal(t, before, redelegationCapacity(t, bApp, ctx, playerID))
	}
	if inactive {
		params, err := bApp.StakingKeeper.GetParams(ctx)
		require.NoError(t, err)
		params.MaxValidators = 2
		require.NoError(t, bApp.StakingKeeper.SetParams(ctx, params))
		_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
		require.NoError(t, err)
		merged, found = bApp.StructsKeeper.GetInfusion(ctx, reactorID, vault.String())
		require.True(t, found)
		require.Zero(t, merged.Power, "bonding queues restoration instead of scanning positions")
		bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
		merged, found = bApp.StructsKeeper.GetInfusion(ctx, reactorID, vault.String())
		require.True(t, found)

		require.Equal(t, uint64(2000000), merged.Power)
	}
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

func TestRotationCombinesFractionalStakeAfterSlash(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(playerID)
	require.NoError(t, err)
	vault := vaults[0]
	staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	totalShares := math.LegacyZeroDec()
	for _, address := range []sdk.AccAddress{primary, vault} {
		delegate(t, bApp, ctx, staking, denom, address, validator, math.OneInt())
		delegation, err := bApp.StakingKeeper.GetDelegation(ctx, address, validator)
		require.NoError(t, err)
		totalShares = totalShares.Add(delegation.Shares)
	}
	reactorID := jailGateReactorId(t, bApp, ctx, validator)
	before, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	consensus, err := before.GetConsAddr()
	require.NoError(t, err)
	_, err = bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(), before.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(5, 2))
	require.NoError(t, err)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	for _, address := range []sdk.AccAddress{primary, vault} {
		row, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, address.String())
		require.True(t, found)
		require.Zero(t, row.Fuel)

	}
	server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	_, err = server.PlayerUpdatePrimaryAddress(ctx, &types.MsgPlayerUpdatePrimaryAddress{Creator: primary.String(), PrimaryAddress: secondary.String()})
	require.NoError(t, err)
	merged, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, vault.String())
	require.True(t, found)
	require.Equal(t, uint64(1), merged.Fuel)

	require.Equal(t, uint64(1), merged.Power)
	delegation, err := bApp.StakingKeeper.GetDelegation(ctx, vault, validator)
	require.NoError(t, err)
	require.Equal(t, totalShares, delegation.Shares)
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

func TestMissingSourceInfusionRebuildsTransferredBacking(t *testing.T) {
	for _, destinationExists := range []bool{false, true} {
		t.Run(map[bool]string{false: "new-vault-position", true: "existing-vault-position"}[destinationExists], func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			delegate(t, bApp, ctx, staking, denom, primary, validator, math.NewInt(1000000))
			reactorID := jailGateReactorId(t, bApp, ctx, validator)
			vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(playerID)
			require.NoError(t, err)
			fuel := uint64(1000000)
			if destinationExists {
				delegate(t, bApp, ctx, staking, denom, vaults[0], validator, math.NewInt(1000000))
				fuel += 1000000
			}

			bApp.StructsKeeper.RemoveInfusion(ctx, reactorID, primary.String())
			_, err = keeperlib.NewMsgServerImpl(bApp.StructsKeeper).PlayerUpdatePrimaryAddress(ctx, &types.MsgPlayerUpdatePrimaryAddress{Creator: primary.String(), PrimaryAddress: secondary.String()})
			require.NoError(t, err)
			merged, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, vaults[0].String())
			require.True(t, found)
			require.Equal(t, fuel, merged.Fuel)
			require.Equal(t, fuel, merged.Power)

			requireStakingAndDistributionHealthy(t, bApp, ctx)
		})
	}
}
