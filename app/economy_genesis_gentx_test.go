package app_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	"structs/x/structs/types"
)

// Simulation genesis helpers install validators directly and never call the
// BaseApp genesis transaction handler. Real `collect-gentxs` genesis does.
func TestEconomyGenesisSignedGentxsShareInitializedModuleState(t *testing.T) {
	db := dbm.NewMemDB()
	options := simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}
	bApp, err := app.New(log.NewNopLogger(), db, nil, true, options, baseapp.SetChainID(jailGateChainID))
	require.NoError(t, err)
	genesis := bApp.DefaultGenesis()
	var structsGenesis types.GenesisState
	bApp.AppCodec().MustUnmarshalJSON(genesis[types.ModuleName], &structsGenesis)

	power, capacity := uint64(1000000), uint64(960000)
	genesis[types.ModuleName] = bApp.AppCodec().MustMarshalJSON(&structsGenesis)
	config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
	accounts := []authtypes.GenesisAccount{}
	keys := []*secp256k1.PrivKey{}
	balances := []banktypes.Balance{}
	txs := []sdk.Tx{}
	for i := 0; i < 2; i++ {
		key := secp256k1.GenPrivKey()
		keys = append(keys, key)
		address := sdk.AccAddress(key.PubKey().Address())
		accounts = append(accounts, authtypes.NewBaseAccount(address, key.PubKey(), uint64(i), 0))
		balances = append(balances, banktypes.Balance{Address: address.String(), Coins: sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 2000000))})
		msg, err := stakingtypes.NewMsgCreateValidator(sdk.ValAddress(address).String(), ed25519.GenPrivKey().PubKey(),
			sdk.NewInt64Coin(sdk.DefaultBondDenom, 1000000), stakingtypes.NewDescription(fmt.Sprintf("seed-%d", i), "", "", "", ""),
			stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(1, 1), math.LegacyNewDecWithPrec(2, 1), math.LegacyNewDecWithPrec(1, 2)), math.OneInt())
		require.NoError(t, err)
		// Genesis signature checks use account number zero for every signer.
		tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(int64(i+1))), config, []sdk.Msg{msg}, nil, 200000,
			jailGateChainID, []uint64{0}, []uint64{0}, key)
		require.NoError(t, err)
		require.NotEmpty(t, tx.(sdk.TxWithMemo).GetMemo())
		txs = append(txs, tx)
	}
	genesis[authtypes.ModuleName] = bApp.AppCodec().MustMarshalJSON(authtypes.NewGenesisState(authtypes.DefaultParams(), accounts))
	bankGenesis := banktypes.DefaultGenesisState()
	bankGenesis.Balances = balances
	genesis[banktypes.ModuleName] = bApp.AppCodec().MustMarshalJSON(bankGenesis)
	genesis[genutiltypes.ModuleName] = bApp.AppCodec().MustMarshalJSON(genutiltypes.NewGenesisStateFromTx(config.TxJSONEncoder(), txs))
	encoded, err := json.Marshal(genesis)
	require.NoError(t, err)
	header := cmtproto.Header{Height: 1, ChainID: jailGateChainID, Time: time.Unix(1800000000, 0)}
	// Refuse a later gentx after the first has debited its account. The
	// SDK may retain failed initialization in memory, but it must not
	// persist it, and a corrected InitChain must start from a new cache.
	badGenutil := genutiltypes.NewGenesisStateFromTx(config.TxJSONEncoder(), txs)
	var badTx map[string]interface{}
	require.NoError(t, json.Unmarshal(badGenutil.GenTxs[1], &badTx))
	badTx["body"].(map[string]interface{})["memo"] = strings.Repeat("x", 257)
	badGenutil.GenTxs[1], err = json.Marshal(badTx)
	require.NoError(t, err)
	genesis[genutiltypes.ModuleName] = bApp.AppCodec().MustMarshalJSON(badGenutil)
	bad, err := json.Marshal(genesis)
	require.NoError(t, err)
	require.Panics(t, func() {
		_, _ = bApp.InitChain(&abci.RequestInitChain{AppStateBytes: bad, ChainId: jailGateChainID, InitialHeight: 1, Time: header.Time, ConsensusParams: simtestutil.DefaultConsensusParams})
	})
	failed := bApp.NewContextLegacy(false, header)
	require.Equal(t, uint64(1), bApp.AccountKeeper.GetAccount(failed, accounts[0].GetAddress()).GetSequence(), "the first signed gentx must have executed")
	require.Equal(t, uint64(0), bApp.AccountKeeper.GetAccount(failed, accounts[1].GetAddress()).GetSequence())
	persisted := bApp.NewUncachedContext(false, header)
	require.Nil(t, bApp.AccountKeeper.GetAccount(persisted, accounts[0].GetAddress()))
	require.True(t, bApp.BankKeeper.GetSupply(persisted, sdk.DefaultBondDenom).Amount.IsZero())
	require.Zero(t, bApp.LastBlockHeight())
	var response *abci.ResponseInitChain
	require.NotPanics(t, func() {
		response, err = bApp.InitChain(&abci.RequestInitChain{AppStateBytes: encoded, ChainId: jailGateChainID, InitialHeight: 1, Time: header.Time, ConsensusParams: simtestutil.DefaultConsensusParams})
	})
	require.NoError(t, err)
	require.Len(t, response.Validators, 2)
	verify := func(ctx sdk.Context) {
		for _, account := range accounts {
			address := account.GetAddress()
			require.Equal(t, uint64(1), bApp.AccountKeeper.GetAccount(ctx, address).GetSequence())
			require.Equal(t, "1000000", bApp.BankKeeper.GetBalance(ctx, address, sdk.DefaultBondDenom).Amount.String())
			delegation, err := bApp.StakingKeeper.GetDelegation(ctx, address, sdk.ValAddress(address))
			require.NoError(t, err)
			reactorID := jailGateReactorId(t, bApp, ctx, sdk.ValAddress(address))
			infusion, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, address.String())
			require.True(t, found)
			require.Equal(t, uint64(1000000), infusion.Fuel)

			require.True(t, delegation.Shares.Equal(math.LegacyNewDec(1000000)))
			_, err = bApp.DistrKeeper.GetDelegatorStartingInfo(ctx, sdk.ValAddress(address), address)
			require.NoError(t, err)
		}
		require.Equal(t, "2000000", bApp.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(stakingtypes.BondedPoolName), sdk.DefaultBondDenom).Amount.String())
	}
	verify(bApp.NewContextLegacy(false, header))
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: header.Time})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	beforeRestart := bApp
	bApp, err = app.New(log.NewNopLogger(), db, nil, true, options, baseapp.SetChainID(jailGateChainID))
	require.NoError(t, err)
	verify(bApp.NewUncachedContext(false, header))
	ctx := bApp.NewUncachedContext(false, header).WithBlockHeight(2)
	allocationTxs := [][]byte{}
	for i, account := range accounts {
		address := account.GetAddress()
		reactorID := jailGateReactorId(t, bApp, ctx, sdk.ValAddress(address))
		infusion, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, address.String())
		require.True(t, found)
		require.Equal(t, power, infusion.Power, "the first block must reconcile the newly bonded validator")
		player, err := bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayerByAddress(address.String())
		require.NoError(t, err)
		require.Equal(t, capacity, player.GetCapacity())
		msg := &types.MsgAllocationCreate{
			Creator: address.String(), SourceObjectId: player.ID(), Power: capacity, AllocationType: types.AllocationType_dynamic,
		}
		signer := bApp.AccountKeeper.GetAccount(ctx, address)
		tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(int64(i+1))), config, []sdk.Msg{msg}, nil, 1000000,
			jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, keys[i])
		require.NoError(t, err)
		encoded, err := config.TxEncoder()(tx)
		require.NoError(t, err)
		for _, simulator := range []*app.App{beforeRestart, bApp} {
			before := applicationAssetStateRows(simulator, simulator.NewUncachedContext(false, header))
			gas, _, err := simulator.Simulate(encoded)
			require.NoError(t, err, "the CLI's gas simulation must accept the allocation before and after restart")
			require.Positive(t, gas.GasUsed)
			require.Equal(t, before, applicationAssetStateRows(simulator, simulator.NewUncachedContext(false, header)), "simulation must discard application writes")
		}
		allocationTxs = append(allocationTxs, encoded)
	}
	block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: header.Time.Add(time.Second), Txs: allocationTxs})
	require.NoError(t, err)
	require.Len(t, block.TxResults, 2)
	for _, result := range block.TxResults {
		require.Zero(t, result.Code, "reported genesis capacity must also pass signed admission: %s", result.Log)
	}
	_, err = bApp.Commit()
	require.NoError(t, err)
}
