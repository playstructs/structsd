package app_test

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Full BaseApp execution complements the metered module-hook diagnostics.
// Setup is outside the measured blocks; Commit uses an in-memory database.
// This exercises all application phases, not CometBFT evidence validation or
// a maximum-size transaction block on production validator hardware.
func TestOrdinaryFinalizeBlockCombinedWork(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 2)
	start := ctx.BlockTime()
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	evidenceValidator := createRedelegationValidator(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, "combined-evidence")
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: start})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: start.Add(time.Second)})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 3, Time: start.Add(2 * time.Second)})
	require.NoError(t, err)
	ctx = bApp.NewContextLegacy(false, cmtproto.Header{Height: 3, ChainID: jailGateChainID, Time: start.Add(2 * time.Second)})
	historical, err := bApp.StakingKeeper.GetValidator(ctx, evidenceValidator)
	require.NoError(t, err)
	consensus, err := historical.GetConsAddr()
	require.NoError(t, err)
	require.True(t, historical.IsBonded())
	require.True(t, bApp.SlashingKeeper.HasValidatorSigningInfo(ctx, consensus))
	power := historical.GetConsensusPower(bApp.StakingKeeper.PowerReduction(ctx))
	slashParams, err := bApp.SlashingKeeper.GetParams(ctx)
	require.NoError(t, err)
	wantBurn := slashParams.SlashFractionDoubleSign.MulInt(bApp.StakingKeeper.TokensFromConsensusPower(ctx, power)).TruncateInt()

	// Keep native maturities inside evidence's accepted age window.
	params, err := bApp.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.UnbondingTime = time.Minute
	require.NoError(t, bApp.StakingKeeper.SetParams(ctx, params))
	_, _, generator, _, donors := setupGeneratorQueueOnApp(t, bApp, ctx, types.GeneratorPowerReconcileBudget+2)
	_, _, provider, ids, consumers := setupFundedAgreementsWithDuration(t, bApp, ctx, 1, 0, 1)
	k := bApp.StructsKeeper
	template, found := k.GetAgreement(ctx, ids[0])
	require.True(t, found)
	require.Equal(t, uint64(4), template.EndBlock)
	allocation, found := k.GetAllocation(ctx, template.AllocationId)
	require.True(t, found)
	cc := k.NewCurrentContext(ctx)
	const agreements = types.AgreementExpirationBucketCap
	for i := 1; i < agreements; i++ {
		row, link := template, allocation
		row.Id = keeperlib.GetObjectID(types.ObjectType_agreement, uint64(800000+i))
		row.AllocationId = keeperlib.GetObjectID(types.ObjectType_allocation, uint64(800000+i))
		link.Id, link.Index = row.AllocationId, uint64(800000+i)
		cc.GenesisImportAllocation(link, row.Capacity)
		cc.GenesisImportAgreement(row)
		ids = append(ids, row.Id)
	}
	cc.CommitAll()
	fundRedelegationAccount(t, bApp, ctx, keeperlib.GetProviderCollateralPoolLocation(provider), sdk.NewCoins(sdk.NewInt64Coin("ualpha", 3*(agreements-1))))
	for i := 0; i < types.PermissionCleanupBudget+1; i++ {
		address := make([]byte, 20)
		address[0] = 0x65
		binary.BigEndian.PutUint64(address[12:], uint64(i+1))
		player := appendAppPlayer(t, k, ctx, sdk.AccAddress(address))
		for _, id := range ids {
			k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(id, player.Id), types.PermUpdate)
		}
	}
	_, _, gridOwner := newTransferPlayer(t, bApp, ctx)
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, gridOwner), 1)
	cc = k.NewCurrentContext(ctx)
	source := gridOwner
	for i := 0; i < types.GridCascadeBlockBudget+2; i++ {
		next := fmt.Sprintf("4-%08d", 850000+i)
		k.SetSubstation(ctx, types.Substation{Id: next, Owner: gridOwner})
		cc.GenesisImportAllocation(types.Allocation{Id: fmt.Sprintf("6-%08d", 850000+i), SourceObjectId: source, DestinationId: next, Controller: gridOwner, Type: types.AllocationType_dynamic}, 1)
		source = next
	}
	cc.CommitAll()
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, gridOwner), 0)
	k.AppendGridCascadeQueue(ctx, gridOwner)
	cc = k.NewCurrentContext(ctx)
	cc.GetStruct(generator).GoOffline()
	cc.CommitAll()
	// GetLastValidators is address ordered; never let random fixture keys
	// choose the evidence validator for the independent deferred slash work.
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	var workValidator sdk.ValAddress
	for _, validator := range validators {
		if validator.OperatorAddress != evidenceValidator.String() {
			workValidator, err = sdk.ValAddressFromBech32(validator.OperatorAddress)
			require.NoError(t, err)
			break
		}
	}
	require.NotEmpty(t, workValidator)
	work, mature := setupOrdinaryStakingWorkForValidator(t, bApp, ctx, workValidator)

	key := secp256k1.GenPrivKey()
	actor := appendAppPlayer(t, k, ctx, sdk.AccAddress(key.PubKey().Address()))
	peer := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
	vaults, err := k.NewCurrentContext(ctx).EnsureVaults(actor.Id, peer.Id)
	require.NoError(t, err)
	fundRedelegationAccount(t, bApp, ctx, vaults[0], sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100)))
	work.supply = bApp.BankKeeper.GetSupply(ctx, denom)
	account := bApp.AccountKeeper.GetAccount(ctx, sdk.AccAddress(key.PubKey().Address()))
	config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
	send := func(recipient string, amount int64) sdk.Msg {
		return &types.MsgVaultSend{Creator: actor.PrimaryAddress, OwnerId: actor.Id, RecipientOwnerId: recipient, Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", amount))}
	}
	var txs [][]byte
	for index, messages := range [][]sdk.Msg{{send(peer.Id, 7)}, {send(peer.Id, 5), send("1-99999999", 1)}} {
		tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(int64(index))), config, messages, nil,
			ante.DefaultFreeGasCap, jailGateChainID, []uint64{account.GetAccountNumber()}, []uint64{account.GetSequence() + uint64(index)}, key)
		require.NoError(t, err)
		wire, err := config.TxEncoder()(tx)
		require.NoError(t, err)
		txs = append(txs, wire)
	}
	bApp.SimWriteState()
	_, err = bApp.Commit()
	require.NoError(t, err)
	beforeSupply := work.supply.Amount
	work.supply = sdk.NewCoin(denom, beforeSupply.Sub(wantBurn))
	evidence := []abci.Misbehavior{{Type: abci.MisbehaviorType_DUPLICATE_VOTE,
		// The validator-set update published at height 1 signs height 3.
		Validator: abci.Validator{Address: consensus, Power: power}, Height: 3,
		Time: start.Add(2 * time.Second), TotalVotingPower: power + 2}}
	drained := false
	for pass := 0; pass < 300; pass++ {
		height, when := int64(4+pass), mature.BlockTime().Add(time.Duration(pass)*time.Second)
		request := &abci.RequestFinalizeBlock{Height: height, Time: when}
		if pass == 0 {
			request.Txs, request.Misbehavior = txs, evidence
		}
		began := time.Now()
		result, err := bApp.FinalizeBlock(request)
		finalizeTime := time.Since(began)
		require.NoError(t, err)
		began = time.Now()
		_, err = bApp.Commit()
		commitTime := time.Since(began)
		require.NoError(t, err)
		ctx = bApp.NewContextLegacy(true, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: when})
		permissionEvents := 0
		var typedEvents []abci.Event
		burned := sdk.NewCoins()
		for _, event := range result.Events {
			if event.Type == "structs.structs.EventPermission" || event.Type == "structs.structs.EventGuildRankPermission" {
				permissionEvents++
			}
			payload := abci.Event{Type: event.Type}
			for _, attribute := range event.Attributes {
				if attribute.Key != "mode" {
					payload.Attributes = append(payload.Attributes, attribute)
				}
				if event.Type == banktypes.EventTypeCoinBurn && attribute.Key == sdk.AttributeKeyAmount {
					coins, err := sdk.ParseCoinsNormalized(attribute.Value)
					require.NoError(t, err)
					burned = burned.Add(coins...)
				}
			}
			typedEvents = append(typedEvents, payload)
		}
		require.LessOrEqual(t, permissionEvents, types.PermissionCleanupBudget)
		if pass == 0 {
			require.Len(t, result.TxResults, 2)
			require.Zero(t, result.TxResults[0].Code, result.TxResults[0].Log)
			require.NotZero(t, result.TxResults[1].Code)
			require.NotContains(t, result.TxResults[1].Log, "panic")
			require.Contains(t, result.TxResults[1].Log, "message index: 1", "the first transfer must execute before the second message refuses")
			var movements []string
			for _, event := range result.TxResults[0].Events {
				if event.Type != banktypes.EventTypeCoinSpent && event.Type != banktypes.EventTypeCoinReceived {
					continue
				}
				attributes := map[string]string{}
				for _, attribute := range event.Attributes {
					attributes[attribute.Key] = attribute.Value
				}
				address := attributes[banktypes.AttributeKeySpender] + attributes[banktypes.AttributeKeyReceiver]
				movements = append(movements, event.Type+":"+address+"="+attributes[sdk.AttributeKeyAmount])
			}
			require.Equal(t, []string{"coin_spent:" + vaults[0].String() + "=7ualpha", "coin_received:" + vaults[1].String() + "=7ualpha"}, movements)
			for _, event := range result.TxResults[1].Events {
				require.NotEqual(t, banktypes.EventTypeCoinSpent, event.Type, "failed batch must not publish vault movement")
				require.NotEqual(t, banktypes.EventTypeCoinReceived, event.Type)
			}
			require.True(t, burned.AmountOf(denom).Equal(wantBurn))
			require.True(t, bApp.SlashingKeeper.IsTombstoned(ctx, consensus))
			require.NotZero(t, k.GetGridCascadeQueueLength(ctx))
			require.NotEmpty(t, k.GetPermissionCleanupQueue(ctx))
		} else {
			require.True(t, burned.IsZero(), "later repair must not repeat the evidence penalty")
		}
		require.Equal(t, int64(93), bApp.BankKeeper.GetBalance(ctx, vaults[0], "ualpha").Amount.Int64())
		require.Equal(t, int64(7), bApp.BankKeeper.GetBalance(ctx, vaults[1], "ualpha").Amount.Int64())
		// One served block: floor(1.8*N) aggregate earnings plus floor(1.2)
		// per agreement; the remaining rounding dust stays in collateral.
		const earned = 18*agreements/10 + agreements
		require.Equal(t, int64(earned), bApp.BankKeeper.GetBalance(ctx, keeperlib.GetProviderEarningsPoolLocation(provider), "ualpha").Amount.Int64())
		require.Equal(t, int64(3*agreements-earned), bApp.BankKeeper.GetBalance(ctx, keeperlib.GetProviderCollateralPoolLocation(provider), "ualpha").Amount.Int64())
		require.True(t, bApp.BankKeeper.GetAllBalances(ctx, consumers[0]).IsZero())
		require.Empty(t, k.GetAllAgreementIdByProviderIndex(ctx, provider))
		require.Zero(t, k.NewCurrentContext(ctx).GetProvider(provider).GetAgreementLoad())
		stakingDone := work.verify(t, bApp, ctx, pass, typedEvents)
		_, generatorQueued, err := k.GetGeneratorPowerSnapshot(ctx, generator)
		require.NoError(t, err)
		if pass == 0 {
			require.True(t, generatorQueued)
		}
		drained = stakingDone && !generatorQueued && k.GetGridCascadeQueueLength(ctx) == 0 && len(k.GetPermissionCleanupQueue(ctx)) == 0
		if pass < 4 || drained {
			wire, err := result.Marshal()
			require.NoError(t, err)
			rpc, err := cmtjson.Marshal(&coretypes.ResultBlockResults{Height: height, TxsResults: result.TxResults, FinalizeBlockEvents: result.Events, ValidatorUpdates: result.ValidatorUpdates, ConsensusParamUpdates: result.ConsensusParamUpdates, AppHash: result.AppHash})
			require.NoError(t, err)
			t.Logf("pass=%d finalize=%s commit=%s txs=%d events=%d abci_bytes=%d rpc_result_bytes=%d drained=%t", pass, finalizeTime, commitTime, len(result.TxResults), len(result.Events), len(wire), len(rpc), drained)
		}
		if drained && pass >= 3 {
			break
		}
	}
	require.True(t, drained)
	for _, donor := range donors {
		require.Zero(t, k.NewCurrentContext(ctx).GetPlayer(donor.Id).GetCapacity())
	}
	for _, id := range ids {
		_, found := k.GetAgreement(ctx, id)
		require.False(t, found)
		require.Empty(t, k.GetPermissionsByObject(ctx, id))
	}
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
