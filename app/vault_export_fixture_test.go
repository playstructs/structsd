package app_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

// Opt-in fixture generator for the offline export verifier. It uses real SDK
// state and module exporters but starts no node and sends no network traffic.
// Every invocation creates a fresh directory, never overwriting an old fixture.
func TestWriteVaultStakingMigrationExportFixture(t *testing.T) {
	root := os.Getenv("STRUCTS_UPGRADE_EXPORT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("set STRUCTS_UPGRADE_EXPORT_FIXTURE_ROOT to retain before/after SDK exports")
	}
	directory, err := os.MkdirTemp(root, "vault-staking-")
	require.NoError(t, err)
	bApp, ctx := setupJailGateAppWithValidators(t, 2)
	primary, secondary, _ := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	source, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	target, err := sdk.ValAddressFromBech32(validators[1].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	for _, address := range []sdk.AccAddress{primary, secondary} {
		delegate(t, bApp, ctx, server, denom, address, source, math.NewInt(3000000))
		_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(address.String(), source.String(), sdk.NewInt64Coin(denom, 100000)))
		require.NoError(t, err)
		_, err = server.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(address.String(), source.String(), target.String(), sdk.NewInt64Coin(denom, 200000)))
		require.NoError(t, err)
	}
	write := func(name string, value interface{}) {
		data, err := json.MarshalIndent(value, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(directory, name), data, 0600))
	}
	export := func(name string) {
		state, err := bApp.ModuleManager.ExportGenesisForModules(ctx, bApp.AppCodec(), nil)
		require.NoError(t, err)
		write(name, state)
	}
	export("before.json")
	require.NoError(t, v22.MigrateVaultAssets(ctx, delegationUpgradeKeepers(bApp)))
	export("after.json")
	owners, err := bApp.StructsKeeper.GetVaultOwnersForExport(ctx)
	require.NoError(t, err)
	vaults := map[string]string{}
	for _, owner := range owners {
		address, err := types.VaultAddress(owner)
		require.NoError(t, err)
		vaults[owner] = address.String()
	}
	write("vault-map.json", vaults)
	requireStakingAndDistributionHealthy(t, bApp, ctx)
	t.Logf("SDK migration fixture: %s", directory)
}
