package app_test

import (
	"testing"
	"encoding/json"
	"os"
	"path/filepath"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

// Compose actual SDK delegation/redelegation state with the unwired ordinary
// account phases. This does not claim self-delegation, vesting or grid rebasing.
func TestRetainedSelfStakeRedelegationSlashReachesPendingUnbonding(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	validators, err := bApp.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	source, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	destination := createRedelegationValidator(t, bApp, ctx, server, denom, "retainedslash")
	operator := sdk.AccAddress(destination)
	delegate(t, bApp, ctx, server, denom, operator, source, math.NewInt(3000000))
	_, err = server.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(operator.String(), source.String(), destination.String(), sdk.NewInt64Coin(denom, 2000000)))
	require.NoError(t, err)
	_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(operator.String(), destination.String(), sdk.NewInt64Coin(denom, 1000000)))
	require.NoError(t, err)
	red, err := bApp.StakingKeeper.GetRedelegation(ctx, operator, source, destination)
	require.NoError(t, err)
	ubd, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, operator, destination)
	require.NoError(t, err)
	queue, err := bApp.StakingKeeper.GetUBDQueueTimeSlice(ctx, ubd.Entries[0].CompletionTime)
	require.NoError(t, err)
	fixtureRoot := os.Getenv("STRUCTS_SELF_STAKE_FIXTURE_ROOT")
	var beforeExport map[string]json.RawMessage
	if fixtureRoot != "" {
		beforeExport, err = bApp.ModuleManager.ExportGenesisForModules(ctx, bApp.AppCodec(), nil)
		require.NoError(t, err)
	}
	require.NoError(t, v22.MigrateVaultAssets(ctx, delegationUpgradeKeepers(bApp)))
	if fixtureRoot != "" {
		afterExport, err := bApp.ModuleManager.ExportGenesisForModules(ctx, bApp.AppCodec(), nil)
		require.NoError(t, err)
		owners, err := bApp.StructsKeeper.GetVaultOwnersForExport(ctx)
		require.NoError(t, err)
		vaults := map[string]string{}
		for _, owner := range owners {
			address, err := types.VaultAddress(owner)
			require.NoError(t, err)
			vaults[owner] = address.String()
		}
		directory, err := os.MkdirTemp(fixtureRoot, "retained-self-stake-")
		require.NoError(t, err)
		for name, value := range map[string]interface{}{"before.json": beforeExport, "after.json": afterExport, "vault-map.json": vaults} {
			data, err := json.MarshalIndent(value, "", "  ")
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(directory, name), data, 0600))
		}
		t.Logf("Retained self-stake export fixture: %s", directory)
	}

	afterQueue, err := bApp.StakingKeeper.GetUBDQueueTimeSlice(ctx, ubd.Entries[0].CompletionTime)
	require.NoError(t, err)
	require.Equal(t, queue, afterQueue)
	retained, err := bApp.StakingKeeper.GetRedelegationByUnbondingID(ctx, red.Entries[0].UnbondingId)
	require.NoError(t, err)
	require.Equal(t, red, retained)
	retainedUBD, err := bApp.StakingKeeper.GetUnbondingDelegationByUnbondingID(ctx, ubd.Entries[0].UnbondingId)
	require.NoError(t, err)
	require.Equal(t, ubd, retainedUBD, "pending self-stake must stay reachable by its redelegation")
	before, err := bApp.StakingKeeper.GetDelegation(ctx, operator, destination)
	require.NoError(t, err)
	src, err := bApp.StakingKeeper.GetValidator(ctx, source)
	require.NoError(t, err)
	supply := bApp.BankKeeper.GetSupply(ctx, denom).Amount
	slashed, err := bApp.StakingKeeper.SlashRedelegation(ctx, src, retained, red.Entries[0].CreationHeight, math.LegacyMustNewDecFromStr("0.1"))
	require.NoError(t, err)
	require.Equal(t, int64(200000), slashed.Int64())
	after, err := bApp.StakingKeeper.GetDelegation(ctx, operator, destination)
	require.NoError(t, err)
	require.Equal(t, before, after, "SDK prioritizes pending unbonding over bonded self-stake")
	remaining, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, operator, destination)
	require.NoError(t, err)
	require.Equal(t, int64(800000), remaining.Entries[0].Balance.Int64())
	require.Equal(t, int64(200000), supply.Sub(bApp.BankKeeper.GetSupply(ctx, denom).Amount).Int64())
}
