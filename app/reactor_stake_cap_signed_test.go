package app_test

import (
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"math/rand"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	"structs/x/structs/types"
)

func TestSignedReactorStakeCapBatchRollbackAndNextBlockRetry(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 4)
	key := secp256k1.GenPrivKey()
	address := sdk.AccAddress(key.PubKey().Address())
	appendAppPlayer(t, bApp.StructsKeeper, ctx, address)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	fundRedelegationAccount(t, bApp, ctx, address, sdk.NewCoins(sdk.NewInt64Coin(denom, 1000000)))
	secondKey := secp256k1.GenPrivKey()
	secondAddress := sdk.AccAddress(secondKey.PubKey().Address())
	appendAppPlayer(t, bApp.StructsKeeper, ctx, secondAddress)
	fundRedelegationAccount(t, bApp, ctx, secondAddress, sdk.NewCoins(sdk.NewInt64Coin(denom, 1000000)))
	activateTestStakeCap(t, bApp, ctx)
	signer := bApp.AccountKeeper.GetAccount(ctx, address)
	config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
	secondSigner := bApp.AccountKeeper.GetAccount(ctx, secondAddress)
	encode := func(sequence uint64, batch bool) []byte {
		t.Helper()
		messages := []sdk.Msg{stakingtypes.NewMsgDelegate(address.String(), validator.String(), sdk.NewInt64Coin(denom, 100000))}
		numbers, sequences := []uint64{signer.GetAccountNumber()}, []uint64{sequence}
		keys := []cryptotypes.PrivKey{key}
		if batch {
			messages = append(messages, stakingtypes.NewMsgDelegate(secondAddress.String(), validator.String(), sdk.NewInt64Coin(denom, 200000)))
			numbers, sequences = append(numbers, secondSigner.GetAccountNumber()), append(sequences, secondSigner.GetSequence())
			keys = append(keys, secondKey)
		}
		tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages,
			sdk.NewCoins(sdk.NewInt64Coin(denom, 1)), ante.DefaultFreeGasCap, jailGateChainID,
			numbers, sequences, keys...)
		require.NoError(t, err)
		wire, err := config.TxEncoder()(tx)
		require.NoError(t, err)
		return wire
	}
	// Each deposit alone fits. Combined, 1.3M / 4.3M exceeds 30%.
	// Different signers are intentional: ante allows one native staking message
	// per address per block, independently of the new stake cap.
	block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{encode(signer.GetSequence(), true)}})
	require.NoError(t, err)
	require.Len(t, block.TxResults, 1)
	rejected := block.TxResults[0]
	require.Equal(t, types.ErrReactorStakeCapExceeded.ABCICode(), rejected.Code, rejected.Log)
	require.Contains(t, rejected.Log, "message index: 1")
	failedCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
	for _, account := range []sdk.AccAddress{address, secondAddress} {
		_, err := bApp.StakingKeeper.GetDelegation(failedCtx, account, validator)
		require.ErrorIs(t, err, stakingtypes.ErrNoDelegation)
	}
	require.Equal(t, int64(1000000), bApp.BankKeeper.GetBalance(failedCtx, address, denom).Amount.Int64())
	require.Equal(t, int64(1000000), bApp.BankKeeper.GetBalance(failedCtx, secondAddress, denom).Amount.Int64())
	_, err = bApp.Commit()
	require.NoError(t, err)
	block, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: ctx.BlockTime().Add(time.Second), Txs: [][]byte{encode(signer.GetSequence()+1, false)}})
	require.NoError(t, err)
	require.Len(t, block.TxResults, 1)
	require.Zero(t, block.TxResults[0].Code, block.TxResults[0].Log)
	for _, event := range rejected.Events {
		require.NotContains(t, []string{"delegate", "structs.structs.EventInfusion", "structs.structs.EventGrid"}, event.Type)
	}
	finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
	delegation, err := bApp.StakingKeeper.GetDelegation(finalCtx, address, validator)
	require.NoError(t, err)
	liveValidator, err := bApp.StakingKeeper.GetValidator(finalCtx, validator)
	require.NoError(t, err)
	require.Equal(t, "100000", liveValidator.TokensFromShares(delegation.Shares).TruncateInt().String())
	require.Equal(t, int64(900000), bApp.BankKeeper.GetBalance(finalCtx, address, denom).Amount.Int64(), "only successful stake survives; native staking remains fee-exempt")
	reactor := jailGateReactorId(t, bApp, finalCtx, validator)
	row, found := bApp.StructsKeeper.GetInfusion(finalCtx, reactor, address.String())
	require.True(t, found)
	require.Equal(t, uint64(100000), row.Fuel)
	requireStakingAndDistributionHealthy(t, bApp, finalCtx)
}
