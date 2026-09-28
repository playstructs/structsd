package app_test

import (
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"
	"structs/app"
	v22 "structs/app/upgrades/v0_22_0"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Opt-in rehearsal against a STOPPED, state-synced testnet node. It copies only
// application.db, never opens the source database and never connects to peers.
// The upgrade is scheduled on the private copy, then delivered by x/upgrade in
// FinalizeBlock. This preserves the actual stores instead of re-importing them
// through new-version genesis logic before exercising migration.
func TestTestnetStateUpgradeRehearsal(t *testing.T) {
	source := os.Getenv("STRUCTS_TESTNET_REHEARSAL_HOME")
	if source == "" {
		t.Skip("set STRUCTS_TESTNET_REHEARSAL_HOME to a stopped testnet node")
	}
	output := os.Getenv("STRUCTS_TESTNET_REHEARSAL_OUTPUT")
	require.NotEmpty(t, output)
	require.NoError(t, os.MkdirAll(output, 0700))
	home := t.TempDir()
	sourceDB := filepath.Join(source, "data", "application.db")
	targetDB := filepath.Join(home, "data", "application.db")
	require.NoError(t, filepath.WalkDir(sourceDB, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDB, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(targetDB, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0700)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		closeErr := out.Close()
		if err != nil {
			return err
		}
		return closeErr
	}))
	database, err := dbm.NewDB("application", dbm.GoLevelDBBackend, filepath.Join(home, "data"))
	require.NoError(t, err)
	defer database.Close()
	options := simtestutil.AppOptionsMap{flags.FlagHome: home}
	bApp, err := app.New(log.NewNopLogger(), database, nil, true, options, baseapp.SetChainID("structstestnet-111"))
	require.NoError(t, err)
	height := bApp.LastBlockHeight()
	require.Greater(t, height, int64(2700000), "requires the current testnet, not an old genesis fixture")
	now := time.Date(2026, 9, 22, 21, 0, 0, 0, time.UTC)
	ctx := bApp.NewUncachedContext(false, cmtproto.Header{ChainID: "structstestnet-111", Height: height, Time: now})
	beforeSupply := bApp.BankKeeper.GetSupply(ctx, "ualpha")
	beforePlayers := len(bApp.StructsKeeper.GetAllPlayer(ctx))
	require.NoError(t, bApp.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: v22.UpgradeName, Height: height + 1}))
	start := time.Now()
	var blocks []map[string]interface{}
	for i := int64(1); i <= 4; i++ {
		if i == 4 {
			require.NoError(t, bApp.Close())
			database, err = dbm.NewDB("application", dbm.GoLevelDBBackend, filepath.Join(home, "data"))
			require.NoError(t, err)
			defer database.Close()
			bApp, err = app.New(log.NewNopLogger(), database, nil, true, options, baseapp.SetChainID("structstestnet-111"))
			require.NoError(t, err)
		}
		result, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height + i, Time: now.Add(time.Duration(i) * time.Second)})
		require.NoError(t, err)
		blocks = append(blocks, map[string]interface{}{"height": height + i, "finalize_block_events": result.Events, "txs_results": result.TxResults})
		data, err := json.Marshal(result)
		require.NoError(t, err)
		if i == 1 {
			require.NoError(t, os.WriteFile(filepath.Join(output, "upgrade-block.json"), data, 0600))
		}
		_, err = bApp.Commit()
		require.NoError(t, err)
	}
	ctx = bApp.NewUncachedContext(false, cmtproto.Header{ChainID: "structstestnet-111", Height: height + 4, Time: now.Add(4 * time.Second)})

	require.Equal(t, beforePlayers, len(bApp.StructsKeeper.GetAllPlayer(ctx)))
	// Block minting may legitimately increase supply; migration must not burn it.
	require.True(t, bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount.GTE(beforeSupply.Amount))
	for _, invariant := range []struct {
		name  string
		check func() (string, bool)
	}{
		{"vault", func() (string, bool) { return keeperlib.VaultOwnershipConsistencyInvariant(bApp.StructsKeeper)(ctx) }},
		{"collateral", func() (string, bool) { return keeperlib.ProviderCollateralSolvencyInvariant(bApp.StructsKeeper)(ctx) }},
		{"expiry", func() (string, bool) { return keeperlib.AgreementExpiryLivenessInvariant(bApp.StructsKeeper)(ctx) }},
	} {
		message, broken := invariant.check()
		require.False(t, broken, "%s: %s", invariant.name, message)
	}
	requireStakingAndDistributionHealthy(t, bApp, ctx)
	require.NoError(t, bApp.StructsKeeper.CheckReactorDiversityAccounting(ctx))
	require.True(t, bApp.StructsKeeper.ReactorDiversityReady(ctx))
	state, err := bApp.ModuleManager.ExportGenesisForModules(ctx, bApp.AppCodec(), nil)
	require.NoError(t, err)
	data, err := json.Marshal(map[string]interface{}{"chain_id": "structstestnet-111", "initial_height": height + 5, "app_state": state})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(output, "after-export.json"), data, 0600))
	events, err := json.Marshal(blocks)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(output, "blocks.json"), events, 0600))
	owners, err := bApp.StructsKeeper.GetVaultOwnersForExport(ctx)
	require.NoError(t, err)
	vaults := map[string]string{}
	for _, owner := range owners {
		address, err := types.VaultAddress(owner)
		require.NoError(t, err)
		vaults[owner] = address.String()
	}
	encodedVaults, err := json.Marshal(vaults)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(output, "vault-map.json"), encodedVaults, 0600))
	t.Logf("testnet state %d upgraded at %d; continued through %d; players=%d elapsed=%s", height, height+1, height+4, beforePlayers, time.Since(start))
}
