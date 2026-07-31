package app_test

import (
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/core/comet"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"structs/app"
	structskeeper "structs/x/structs/keeper"
	structstypes "structs/x/structs/types"
)

const jailGateChainID = "structs-jailgate"

// TestDowntimeJailZeroesReactorEnergy is the release gate for jail-gated reactor
// energy. Every other test for this feature drives the gate through the mock
// staking keeper, which proves our own logic but says nothing about whether the
// hook is actually wired into staking or whether a real jail reaches it.
//
// This one runs the real staking and slashing modules end to end: a validator is
// created and bonded, its reactor starts producing energy, the validator then
// misses enough blocks for x/slashing to jail it for downtime, and staking's
// validator-set update (the same call staking makes in its EndBlock) must leave
// the reactor at a zero energy ratio.
//
// If this test fails after an SDK upgrade, the offline-node exploit is open
// again: an operator could infuse alpha, shut the node down, and keep producing
// energy indefinitely. Do not skip or weaken it without closing that hole
// another way.
func TestDowntimeJailZeroesReactorEnergy(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	// Fund an operator account. The structs module account carries Minter, so it
	// is the available faucet in a test chain with no mint module.
	operatorAcc := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	stake := math.NewInt(10_000_000)
	funding := sdk.NewCoins(sdk.NewCoin(bondDenom, stake.MulRaw(2)))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, structstypes.ModuleName, funding))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, structstypes.ModuleName, operatorAcc, funding))

	valAddr := sdk.ValAddress(operatorAcc)
	consPubKey := ed25519.GenPrivKey().PubKey()

	// Creating the validator fires AfterValidatorCreated, which is what builds
	// the reactor and its first infusion.
	createMsg, err := stakingtypes.NewMsgCreateValidator(
		valAddr.String(),
		consPubKey,
		sdk.NewCoin(bondDenom, stake),
		stakingtypes.NewDescription("jail-gate", "", "", "", ""),
		stakingtypes.NewCommissionRates(
			math.LegacyNewDecWithPrec(1, 1),
			math.LegacyNewDecWithPrec(2, 1),
			math.LegacyNewDecWithPrec(1, 2),
		),
		math.OneInt(),
	)
	require.NoError(t, err)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	_, err = stakingMsgServer.CreateValidator(ctx, createMsg)
	require.NoError(t, err)

	// Staking's EndBlock moves the new validator into the bonded set. This also
	// registers the slashing signing info that the downtime tracker needs.
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	validator, err := bApp.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.True(t, validator.IsBonded(), "validator should be bonded before we jail it")

	reactorId := jailGateReactorId(t, bApp, ctx, valAddr)
	ratio, fuel := jailGateInfusion(t, bApp, ctx, reactorId, operatorAcc)
	require.NotZero(t, ratio, "a healthy bonded validator should be producing energy")
	require.NotZero(t, fuel, "the self-delegation should be infused as fuel")

	// Take the node offline. x/slashing counts missed blocks in a sliding window
	// and jails once the miss count crosses its tolerance.
	signedBlocksWindow, err := bApp.SlashingKeeper.SignedBlocksWindow(ctx)
	require.NoError(t, err)

	power := validator.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx))
	startHeight := ctx.BlockHeight()
	for height := startHeight; height <= startHeight+signedBlocksWindow+1; height++ {
		offlineCtx := ctx.WithBlockHeight(height)
		require.NoError(t, bApp.SlashingKeeper.HandleValidatorSignature(
			offlineCtx, consPubKey.Address(), power, comet.BlockIDFlagAbsent))

		jailed, err := bApp.StakingKeeper.GetValidator(offlineCtx, valAddr)
		require.NoError(t, err)
		if jailed.IsJailed() {
			ctx = offlineCtx
			break
		}
	}

	validator, err = bApp.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.True(t, validator.IsJailed(), "missing an entire signing window should jail for downtime")

	// The jail itself only sets a flag and drops the power index, so energy is
	// still flowing at this point. Asserting that here is what stops this test
	// from passing for the wrong reason: it pins the gate to the validator-set
	// update below rather than to the jail call.
	ratio, _ = jailGateInfusion(t, bApp, ctx, reactorId, operatorAcc)
	require.NotZero(t, ratio, "the jail flag alone does not gate; the set update does")

	// This is the call staking makes in its own EndBlock. It routes the jailed
	// validator through bondedToUnbonding, which fires AfterValidatorBeginUnbonding
	// and therefore our gate, in the same block as the jail.
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	validator, err = bApp.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.True(t, validator.IsUnbonding(), "the jailed validator should have left the bonded set")

	ratio, fuel = jailGateInfusion(t, bApp, ctx, reactorId, operatorAcc)
	require.Equal(t, uint64(0), ratio, "a jailed validator must produce no energy")
	require.NotZero(t, fuel, "the operator's stake must survive the gate untouched")

	capacity := bApp.StructsKeeper.GetGridAttribute(ctx,
		structskeeper.GetGridAttributeIDByObjectId(structstypes.GridAttributeType_capacity, reactorId))
	require.Equal(t, uint64(0), capacity, "the reactor's commission energy is gated too")
}

// TestStructsEndBlockerRunsAfterStaking guards the ordering the gate depends on.
// The gate fires from staking's EndBlock, and the grid cascade that resolves its
// fallout runs in the structs EndBlocker, so structs must come after staking for
// a jail and its consequences to land in one block.
func TestStructsEndBlockerRunsAfterStaking(t *testing.T) {
	bApp, _ := setupJailGateApp(t)

	order := bApp.ModuleManager.OrderEndBlockers
	stakingAt, structsAt := -1, -1
	for i, name := range order {
		switch name {
		case stakingtypes.ModuleName:
			stakingAt = i
		case structstypes.ModuleName:
			structsAt = i
		}
	}

	require.NotEqual(t, -1, stakingAt, "staking must be in the EndBlocker order")
	require.NotEqual(t, -1, structsAt, "structs must be in the EndBlocker order")
	require.Less(t, stakingAt, structsAt, "structs must run after staking in EndBlock")
}

func setupJailGateApp(t *testing.T) (*app.App, sdk.Context) {
	t.Helper()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = app.DefaultNodeHome

	bApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions,
		baseapp.SetChainID(jailGateChainID))
	require.NoError(t, err)

	// Staking's InitGenesis refuses an empty validator set, so seed one unrelated
	// genesis validator. The validator this test actually jails is created later
	// through MsgCreateValidator so it travels the real creation path.
	seedConsPubKey, err := cryptocodec.ToCmtPubKeyInterface(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)
	valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{
		cmttypes.NewValidator(seedConsPubKey, 1),
	})

	seedPrivKey := secp256k1.GenPrivKey()
	seedAcc := authtypes.NewBaseAccount(seedPrivKey.PubKey().Address().Bytes(), seedPrivKey.PubKey(), 0, 0)
	seedBalance := banktypes.Balance{
		Address: seedAcc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.DefaultPowerReduction.MulRaw(100))),
	}

	genesisState, err := simtestutil.GenesisStateWithValSet(
		bApp.AppCodec(), bApp.DefaultGenesis(), valSet,
		[]authtypes.GenesisAccount{seedAcc}, seedBalance)
	require.NoError(t, err)

	genesisBytes, err := json.Marshal(genesisState)
	require.NoError(t, err)

	_, err = bApp.InitChain(&abci.RequestInitChain{
		ChainId:         jailGateChainID,
		InitialHeight:   1,
		ConsensusParams: simtestutil.DefaultConsensusParams,
		AppStateBytes:   genesisBytes,
	})
	require.NoError(t, err)

	// Genesis state lives in the app's finalize-block state, so work from that
	// same branch rather than committing. This also means the whole test runs
	// inside one block, which is exactly the window the gate has to act within.
	ctx := bApp.NewContextLegacy(false, cmtproto.Header{
		Height:  1,
		ChainID: jailGateChainID,
		Time:    time.Now().UTC(),
	})

	return bApp, ctx
}

func jailGateReactorId(t *testing.T, bApp *app.App, ctx sdk.Context, valAddr sdk.ValAddress) string {
	t.Helper()

	reactorBytes, found := bApp.StructsKeeper.GetReactorBytesFromValidator(ctx, valAddr.Bytes())
	require.True(t, found, "AfterValidatorCreated should have built a reactor")

	reactor, found := bApp.StructsKeeper.GetReactorByBytes(ctx, reactorBytes)
	require.True(t, found)
	return reactor.Id
}

func jailGateInfusion(t *testing.T, bApp *app.App, ctx sdk.Context, reactorId string, delegator sdk.AccAddress) (ratio uint64, fuel uint64) {
	t.Helper()

	infusion, found := bApp.StructsKeeper.GetInfusion(ctx, reactorId, delegator.String())
	require.True(t, found, "the self-delegation should have an infusion record")
	return infusion.Ratio, infusion.Fuel
}
