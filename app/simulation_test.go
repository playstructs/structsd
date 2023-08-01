package app_test

import (
	"os"
	"testing"
	"time"


	simulationtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

 "encoding/json"
    "fmt"
    "math/rand"
    "runtime/debug"
    "strings"

    dbm "github.com/cometbft/cometbft-db"
    "github.com/cometbft/cometbft/libs/log"
    "github.com/cosmos/cosmos-sdk/baseapp"
    "github.com/cosmos/cosmos-sdk/client/flags"
    "github.com/cosmos/cosmos-sdk/server"
    storetypes "github.com/cosmos/cosmos-sdk/store/types"
    simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
    sdk "github.com/cosmos/cosmos-sdk/types"
    authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
    authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
    banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
    capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
    distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
    evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
    govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
    minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
    paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
    simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"
    slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
    stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"structs/app"
)

type storeKeysPrefixes struct {
    A        storetypes.StoreKey
    B        storetypes.StoreKey
    Prefixes [][]byte
}

func init() {
	simcli.GetSimulatorFlags()
}

func fauxMerkleModeOpt(bapp *baseapp.BaseApp) {
    bapp.SetFauxMerkleMode()
}

// BenchmarkSimulation run the chain simulation
// Running using starport command:
// `ignite chain simulate -v --numBlocks 200 --blockSize 50`
// Running as go benchmark test:
// `go test -benchmem -run=^$ -bench ^BenchmarkSimulation ./app -NumBlocks=200 -BlockSize 50 -Commit=true -Verbose=true -Enabled=true`
func BenchmarkSimulation(b *testing.B) {
    simcli.FlagSeedValue = time.Now().Unix()
    simcli.FlagVerboseValue = true
    simcli.FlagCommitValue = true
    simcli.FlagEnabledValue = true

    config := simcli.NewConfigFromFlags()
    config.ChainID = "mars-simapp"
    db, dir, logger, _, err := simtestutil.SetupSimulation(
        config,
        "leveldb-bApp-sim",
        "Simulation",
        simcli.FlagVerboseValue,
        simcli.FlagEnabledValue,
    )

	require.NoError(b, err, "simulation setup failed")

	b.Cleanup(func() {
        require.NoError(b, db.Close())
        require.NoError(b, os.RemoveAll(dir))
	})

    appOptions := make(simtestutil.AppOptionsMap, 0)
    appOptions[flags.FlagHome] = app.DefaultNodeHome
    appOptions[server.FlagInvCheckPeriod] = simcli.FlagPeriodValue

	bApp := app.New(
		logger,
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		0,
        app.MakeEncodingConfig(),
        appOptions,
        baseapp.SetChainID(config.ChainID),
	)
    require.Equal(b, app.Name, bApp.Name())


	// Run randomized simulations
	_, simParams, simErr := simulation.SimulateFromSeed(
		b,
		os.Stdout,
        bApp.BaseApp,
        simtestutil.AppStateFn(
            bApp.AppCodec(),
            bApp.SimulationManager(),
            app.NewDefaultGenesisState(bApp.AppCodec()),
        ),
        simtestutil.SimulationOperations(bApp, bApp.AppCodec(), config),
        bApp.ModuleAccountAddrs(),
        config,
        bApp.AppCodec(),
	)

	// export state and simParams before the simulation error is checked
    err = simtestutil.CheckExportSimulation(bApp, config, simParams)

	require.NoError(b, err)
	require.NoError(b, simErr)

	if config.Commit {
		simtestutil.PrintStats(db)
	}
}
