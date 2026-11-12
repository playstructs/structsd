package app_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"
	"structs/app"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Opt-in companion to tests/run_v023_upgrade_rehearsal.py. Replay the same
// stopped, pre-upgrade application DB twice with identical synthetic ABCI input.
// The source is copied, never opened. No consensus keys or network are needed.
// Native halt/restart is tested separately by the Python runner.
func TestV23NativeCheckpointDeterministicReplay(t *testing.T) {
	root := os.Getenv("STRUCTS_V023_REHEARSAL")
	if root == "" {
		t.Skip("set STRUCTS_V023_REHEARSAL to a completed fresh-chain rehearsal")
	}
	data, err := os.ReadFile(filepath.Join(root, "upgrade-headers.json"))
	require.NoError(t, err)
	var headers []struct {
		ChainID string    `json:"chain_id"`
		Time    time.Time `json:"time"`
	}
	require.NoError(t, json.Unmarshal(data, &headers))
	require.GreaterOrEqual(t, len(headers), 2)
	var outputs [][]byte
	for _, name := range []string{"deterministic-replay-1", "deterministic-replay-2"} {
		home := filepath.Join(root, name)
		require.NoError(t, os.Mkdir(home, 0700), "refuse to overwrite an earlier replay")
		target := filepath.Join(home, "data", "application.db")
		require.NoError(t, os.CopyFS(target, os.DirFS(filepath.Join(root, "checkpoint", "data", "application.db"))))
		options := simtestutil.AppOptionsMap{flags.FlagHome: home}
		open := func() *app.App {
			database, openErr := dbm.NewDB("application", dbm.GoLevelDBBackend, filepath.Join(home, "data"))
			require.NoError(t, openErr)
			instance, openErr := app.New(log.NewNopLogger(), database, nil, true, options, baseapp.SetChainID(headers[0].ChainID))
			require.NoError(t, openErr)
			return instance
		}
		instance := open()
		height := instance.LastBlockHeight()
		ctx := instance.NewUncachedContext(false, cmtproto.Header{ChainID: headers[0].ChainID, Height: height, Time: headers[0].Time})
		plan, err := instance.UpgradeKeeper.GetUpgradePlan(ctx)
		require.NoError(t, err)
		require.Equal(t, "v0.23.0", plan.Name)
		require.Equal(t, height+1, plan.Height)
		before := instance.StructsKeeper.GetAllPlanet(ctx)
		require.NotEmpty(t, before)
		var results []*abci.ResponseFinalizeBlock
		for i := int64(1); i <= 4; i++ {
			if i == 4 {
				require.NoError(t, instance.Close())
				instance = open()
			}
			result, err := instance.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height + i, Time: headers[0].Time.Add(time.Duration(i) * time.Second)})
			require.NoError(t, err)
			results = append(results, result)
			_, err = instance.Commit()
			require.NoError(t, err)
		}
		ctx = instance.NewUncachedContext(false, cmtproto.Header{ChainID: headers[0].ChainID, Height: height + 4, Time: headers[0].Time.Add(4 * time.Second)})
		done, err := instance.UpgradeKeeper.GetDoneHeight(ctx, "v0.23.0")
		require.NoError(t, err)
		require.Equal(t, height+1, done)
		after := instance.StructsKeeper.GetAllPlanet(ctx)
		require.Len(t, after, len(before))
		for i := range before {
			require.True(t, types.IsPlanetSector(after[i].Sector))
			before[i].Sector = after[i].Sector
			require.Equal(t, before[i], after[i])
		}
		for _, invariant := range []func() (string, bool){
			func() (string, bool) {
				return keeperlib.VaultOwnershipConsistencyInvariant(instance.StructsKeeper)(ctx)
			},
			func() (string, bool) {
				return keeperlib.ProviderCollateralSolvencyInvariant(instance.StructsKeeper)(ctx)
			},
			func() (string, bool) { return keeperlib.AgreementExpiryLivenessInvariant(instance.StructsKeeper)(ctx) },
		} {
			message, broken := invariant()
			require.False(t, broken, message)
		}
		requireStakingAndDistributionHealthy(t, instance, ctx)
		require.NoError(t, instance.StructsKeeper.CheckReactorDiversityAccounting(ctx))
		exported, err := instance.ExportAppStateAndValidators(false, nil, nil)
		require.NoError(t, err)
		output, err := json.Marshal(struct {
			Blocks []*abci.ResponseFinalizeBlock `json:"blocks"`
			State  json.RawMessage               `json:"app_state"`
		}{results, json.RawMessage(exported.AppState)})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(home, "result.json"), output, 0600))
		outputs = append(outputs, output)
		require.NoError(t, instance.Close())
	}
	require.Equal(t, string(outputs[0]), string(outputs[1]), "identical input must produce identical hashes, events and exports")
}
