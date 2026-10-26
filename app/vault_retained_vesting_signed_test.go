package app_test

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	"structs/app/ante"
	"structs/x/structs/types"
)

// Continue the real upgrade plus genesis-import fixture with signed transactions.
// Native vesting obligations must remain at the operator address after changing
// its Structs association, including when a later message rolls a rotation back.
func verifySignedRetainedVestingIdentity(t *testing.T, bApp *app.App, operatorKey, secondaryKey cryptotypes.PrivKey, playerID string, self, other sdk.ValAddress, height int64, when time.Time) {
	t.Helper()
	operator := sdk.AccAddress(operatorKey.PubKey().Address())
	secondary := sdk.AccAddress(secondaryKey.PubKey().Address())
	k := bApp.StructsKeeper
	ctx := bApp.NewContextLegacy(true, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: when})
	config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
	first := true
	deliver := func(key cryptotypes.PrivKey, messages ...sdk.Msg) *abci.ExecTxResult {
		t.Helper()
		account := bApp.AccountKeeper.GetAccount(ctx, sdk.AccAddress(key.PubKey().Address()))
		tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(height)), config, messages, nil,
			ante.DefaultFreeStakingGasCap, jailGateChainID, []uint64{account.GetAccountNumber()}, []uint64{account.GetSequence()}, key)
		require.NoError(t, err)
		wire, err := config.TxEncoder()(tx)
		require.NoError(t, err)
		if !first {
			_, err = bApp.Commit()
			require.NoError(t, err)
		}
		first = false
		block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: when, Txs: [][]byte{wire}})
		require.NoError(t, err)
		require.Len(t, block.TxResults, 1)
		ctx = bApp.NewContext(false).WithBlockTime(when)
		height++
		when = when.Add(time.Second)
		result := block.TxResults[0]
		require.NotContains(t, result.Log, "panic")
		return result
	}
	account := bApp.AccountKeeper.GetAccount(ctx, operator).(vestingexported.VestingAccount)
	beforeSelf, err := bApp.StakingKeeper.GetDelegation(ctx, operator, self)
	require.NoError(t, err)
	beforeOther, err := bApp.StakingKeeper.GetDelegation(ctx, operator, other)
	require.NoError(t, err)
	beforeUBD, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, operator, self)
	require.NoError(t, err)
	beforeRed, err := bApp.StakingKeeper.GetRedelegation(ctx, operator, other, self)
	require.NoError(t, err)
	vault, err := types.VaultAddress(playerID)
	require.NoError(t, err)
	beforeSource, beforeVault := bApp.BankKeeper.GetAllBalances(ctx, operator), bApp.BankKeeper.GetAllBalances(ctx, vault)
	assertNativeUnchanged := func() {
		t.Helper()
		after := bApp.AccountKeeper.GetAccount(ctx, operator).(vestingexported.VestingAccount)
		// Signing installs the public key and increments sequence in ante.
		// Normalize only those two fields on this detached decoded account;
		// compare the complete native schedule, counters and account identity.
		require.NoError(t, after.SetSequence(account.GetSequence()))
		require.NoError(t, after.SetPubKey(account.GetPubKey()))
		require.Equal(t, account, after)
		actualSelf, err := bApp.StakingKeeper.GetDelegation(ctx, operator, self)
		require.NoError(t, err)
		require.Equal(t, beforeSelf, actualSelf)
		actualOther, err := bApp.StakingKeeper.GetDelegation(ctx, operator, other)
		require.NoError(t, err)
		require.Equal(t, beforeOther, actualOther)
		actualUBD, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, operator, self)
		require.NoError(t, err)
		require.Equal(t, beforeUBD, actualUBD)
		actualRed, err := bApp.StakingKeeper.GetRedelegation(ctx, operator, other, self)
		require.NoError(t, err)
		require.Equal(t, beforeRed, actualRed)
	}
	locked := deliver(operatorKey, &types.MsgPlayerSend{Creator: operator.String(), FromAddress: operator.String(), ToAddress: secondary.String(), Amount: beforeSource})
	require.NotZero(t, locked.Code)
	require.Contains(t, locked.Log, "spendable")
	require.Equal(t, beforeSource, bApp.BankKeeper.GetAllBalances(ctx, operator))
	require.Equal(t, beforeVault, bApp.BankKeeper.GetAllBalances(ctx, vault))
	assertNativeUnchanged()
	rotation := &types.MsgPlayerUpdatePrimaryAddress{Creator: operator.String(), PrimaryAddress: secondary.String()}
	unknown := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	failed := deliver(operatorKey, rotation, &types.MsgPlayerUpdatePrimaryAddress{Creator: operator.String(), PrimaryAddress: unknown})
	require.NotZero(t, failed.Code)
	require.Contains(t, failed.Log, "message index: 1")
	require.Equal(t, operator.String(), k.NewCurrentContext(ctx).GetPlayer(playerID).GetPrimaryAddress())
	require.Equal(t, beforeSource, bApp.BankKeeper.GetAllBalances(ctx, operator))
	require.Equal(t, beforeVault, bApp.BankKeeper.GetAllBalances(ctx, vault))
	assertNativeUnchanged()
	for _, event := range failed.Events {
		require.False(t, strings.HasPrefix(event.Type, "structs.structs."), "failed batch must not publish identity or infusion mutations")
		require.NotContains(t, []string{"transfer", "coin_spent", "coin_received"}, event.Type)
	}
	// Spendability is evaluated at the upcoming delivery time, including native
	// continuous-vesting release between blocks. Only that amount is swept.
	spendable := bApp.BankKeeper.SpendableCoins(ctx.WithBlockTime(when), operator)
	rotated := deliver(operatorKey, rotation)
	require.Zero(t, rotated.Code, rotated.Log)
	require.Equal(t, secondary.String(), k.NewCurrentContext(ctx).GetPlayer(playerID).GetPrimaryAddress())
	require.Equal(t, beforeSource.Sub(spendable...), bApp.BankKeeper.GetAllBalances(ctx, operator))
	require.Equal(t, beforeVault.Add(spendable...), bApp.BankKeeper.GetAllBalances(ctx, vault))
	assertNativeUnchanged()
	revoked := deliver(secondaryKey, &types.MsgAddressRevoke{Creator: secondary.String(), Address: operator.String()})
	require.Zero(t, revoked.Code, revoked.Log)
	require.Zero(t, k.GetPlayerIndexFromAddress(ctx, operator.String()))
	assertNativeUnchanged()
	for _, validator := range []sdk.ValAddress{self, other} {
		reactor := jailGateReactorId(t, bApp, ctx, validator)
		if infusion, found := k.GetInfusion(ctx, reactor, operator.String()); found {
			require.Zero(t, infusion.Power, "revoked address must not leave spendable player capacity")
		}
	}
	// Revoking Structs authority does not revoke the native key or move its
	// vesting liabilities: an SDK-signed withdrawal must remain accessible.
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	withdrawal := deliver(operatorKey, stakingtypes.NewMsgUndelegate(operator.String(), other.String(), sdk.NewInt64Coin(denom, 100000)))
	require.Zero(t, withdrawal.Code, withdrawal.Log)
	pending, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, operator, other)
	require.NoError(t, err)
	require.Len(t, pending.Entries, 1)
	require.Equal(t, int64(100000), pending.Entries[0].Balance.Int64())
	unbondEvents := 0
	for _, event := range withdrawal.Events {
		if event.Type != stakingtypes.EventTypeUnbond {
			continue
		}
		unbondEvents++
		attributes := map[string]string{}
		for _, attribute := range event.Attributes {
			attributes[attribute.Key] = attribute.Value
		}
		require.Equal(t, operator.String(), attributes[stakingtypes.AttributeKeyDelegator])
		require.Equal(t, other.String(), attributes[stakingtypes.AttributeKeyValidator])
		require.Equal(t, sdk.NewInt64Coin(denom, 100000).String(), attributes[sdk.AttributeKeyAmount])
		require.Equal(t, pending.Entries[0].CompletionTime.Format(time.RFC3339), attributes[stakingtypes.AttributeKeyCompletionTime])
	}
	require.Equal(t, 1, unbondEvents)
	require.NotEqual(t, k.NewCurrentContext(ctx).GetPlayer(playerID).GetIndex(), k.GetPlayerIndexFromAddress(ctx, operator.String()), "native staking must not restore the revoked player association")
	after := bApp.AccountKeeper.GetAccount(ctx, operator).(vestingexported.VestingAccount)
	require.IsType(t, account, after)
	require.Equal(t, account.GetOriginalVesting(), after.GetOriginalVesting())
	require.Equal(t, account.GetEndTime(), after.GetEndTime())
	requireStakingAndDistributionHealthy(t, bApp, ctx)
	_, err = bApp.Commit()
	require.NoError(t, err)
}
