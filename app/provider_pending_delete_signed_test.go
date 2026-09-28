package app_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Provider deletion settles consumers immediately and drains all remaining
// funds to the owner. Any failed transfer or later message rolls back all legs.
func TestSignedProviderDeletionSettlesConsumersBeforeOwner(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		for _, scenario := range []string{"success", "success_after_service", "send_disabled", "payout_failure", "consumer_failure", "later_failure"} {
			t.Run(fmt.Sprintf("guild=%t/%s", guildOwned, scenario), func(t *testing.T) {
				bApp, ctx := setupReactorAppGenesis(t, 1, true)
				deleteHeight := int64(1)
				if scenario == "success_after_service" {
					deleteHeight = 3
				}
				startTime := ctx.BlockTime()
				bApp, ctx, id, _, _ := setupFundedAgreementsOnGraph(t, bApp, ctx, 0, 0)
				k := bApp.StructsKeeper
				original, found := k.GetProvider(ctx, id)
				require.True(t, found)
				playerOwner := original.Owner
				if guildOwned {
					guild := types.CreateEmptyGuild()
					guild.Id, guild.Index, guild.Owner, guild.Creator = "0-978", 978, playerOwner, original.Creator
					k.SetGuild(ctx, guild)
					original.Owner = guild.Id
					_, err := k.SetProvider(ctx, original)
					require.NoError(t, err)
				}
				key := secp256k1.GenPrivKey()
				address := sdk.AccAddress(key.PubKey().Address())
				owner := k.NewCurrentContext(ctx).GetPlayer(playerOwner)
				require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), owner.GetIndex()))
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), types.PermDelete)
				vault, err := types.VaultAddress(original.Owner)
				require.NoError(t, err)
				collateral, earnings := keeperlib.GetProviderCollateralPoolLocation(id), keeperlib.GetProviderEarningsPoolLocation(id)
				consumers := []sdk.AccAddress{}
				ids := []string{}
				allocations := map[string]string{}
				server := keeperlib.NewMsgServerImpl(k)
				for i := 0; i < 2; i++ {
					primary, _, consumer := newTransferPlayer(t, bApp, ctx)
					accounts, err := k.NewCurrentContext(ctx).EnsureVaults(consumer)
					require.NoError(t, err)
					fundRedelegationAccount(t, bApp, ctx, accounts[0], sdk.NewCoins(sdk.NewInt64Coin("ualpha", 21)))
					_, err = server.AgreementOpen(ctx, &types.MsgAgreementOpen{Creator: primary.String(), ProviderId: id, Capacity: 1, Duration: 7})
					require.NoError(t, err)
					consumers = append(consumers, accounts[0])
				}
				ids = k.GetAllAgreementIdByProviderIndex(ctx, id)
				require.Len(t, ids, 2)
				for _, claimID := range ids {
					a, found := k.GetAgreement(ctx, claimID)
					require.True(t, found)
					allocations[claimID] = a.AllocationId
				}
				fundRedelegationAccount(t, bApp, ctx, collateral, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 11), sdk.NewInt64Coin("ubonus", 3)))
				fundRedelegationAccount(t, bApp, ctx, earnings, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 7), sdk.NewInt64Coin("ubonus", 5)))
				_, associated := k.GetVaultOwner(ctx, vault)
				require.False(t, associated)
				if scenario == "send_disabled" {
					bApp.BankKeeper.SetSendEnabled(ctx, "ubonus", false)
				}
				failedPayout := false
				bApp.BankKeeper.AppendSendRestriction(func(goCtx context.Context, from, to sdk.AccAddress, coins sdk.Coins) (sdk.AccAddress, error) {
					if scenario == "consumer_failure" && from.Equals(collateral) && to.Equals(consumers[0]) {
						return to, errors.New("forced consumer payout failure")
					}
					if scenario == "payout_failure" && from.Equals(earnings) {
						failedPayout = true
						for _, consumer := range consumers {
							require.Equal(t, int64(21), bApp.BankKeeper.GetBalance(goCtx, consumer, "ualpha").Amount.Int64(), "consumers settle before the owner drain")
						}
						require.True(t, bApp.BankKeeper.GetAllBalances(goCtx, collateral).IsZero())
						return to, errors.New("forced provider payout failure")
					}
					return to, nil
				})
				balances := func(c sdk.Context) map[string]string {
					rows := map[string]string{}
					for _, row := range bApp.BankKeeper.GetAccountsBalances(c) {
						if !row.Coins.IsZero() {
							rows[row.Address] = row.Coins.String()
						}
					}
					return rows
				}
				for height := int64(1); height < deleteHeight; height++ {
					_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: startTime.Add(time.Duration(height-1) * time.Second)})
					require.NoError(t, err)
					_, err = bApp.Commit()
					require.NoError(t, err)
				}
				if deleteHeight > 1 {
					ctx = bApp.NewUncachedContext(false, ctx.BlockHeader()).WithBlockHeight(deleteHeight).WithBlockTime(startTime.Add(time.Duration(deleteHeight-1) * time.Second))
				}
				beforeBalances := balances(ctx)
				config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
				signed := func(c sdk.Context, messages []sdk.Msg) []byte {
					signer := bApp.AccountKeeper.GetAccount(c, address)
					tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil, ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
					require.NoError(t, err)
					wire, err := config.TxEncoder()(tx)
					require.NoError(t, err)
					return wire
				}
				messages := []sdk.Msg{&types.MsgProviderDelete{Creator: address.String(), ProviderId: id}}
				if scenario == "later_failure" {
					messages = append(messages, &types.MsgProviderDelete{Creator: address.String(), ProviderId: "10-999999"})
				}
				block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: deleteHeight, Time: ctx.BlockTime(), Txs: [][]byte{signed(ctx, messages)}})
				require.NoError(t, err)
				require.Len(t, block.TxResults, 1)
				result := block.TxResults[0]
				current := bApp.NewContext(false)
				success := strings.HasPrefix(scenario, "success")
				require.Equal(t, scenario == "payout_failure", failedPayout)
				if success {
					require.Zero(t, result.Code, result.Log)
				} else {
					require.NotZero(t, result.Code)
					reason := map[string]string{"send_disabled": "send transactions are disabled", "payout_failure": "forced provider payout failure", "consumer_failure": "collateral payout", "later_failure": "message index: 1"}[scenario]
					require.Contains(t, result.Log, reason)
					require.NotContains(t, result.Log, "panic")
					require.Equal(t, beforeBalances, balances(current))
					actual, found := k.GetProvider(current, id)
					require.True(t, found)
					require.Equal(t, original, actual)
					require.ElementsMatch(t, ids, k.GetAllAgreementIdByProviderIndex(current, id))
					_, associated = k.GetVaultOwner(current, vault)
					require.False(t, associated)
					require.Nil(t, bApp.AccountKeeper.GetAccount(current, vault))
				}
				transferAmounts := map[string]sdk.Coins{}
				for _, event := range result.Events {
					require.False(t, strings.HasPrefix(event.Type, "structs.structs.EventService"))
					if !success {
						require.NotContains(t, []string{"transfer", "coin_spent", "coin_received", "structs.structs.EventVaultAddress", "structs.structs.EventDelete"}, event.Type)
					}
					if event.Type == "transfer" {
						attrs := map[string]string{}
						for _, a := range event.Attributes {
							attrs[a.Key] = a.Value
						}
						paid, err := sdk.ParseCoinsNormalized(attrs["amount"])
						require.NoError(t, err)
						transferAmounts[attrs["recipient"]] = transferAmounts[attrs["recipient"]].Add(paid...)
					}
				}
				for _, id := range ids {
					_, live := k.GetAgreement(current, id)
					require.Equal(t, !success, live)
					_, live = k.GetAllocation(current, allocations[id])
					require.Equal(t, !success, live)
				}
				for _, pool := range []sdk.AccAddress{collateral, earnings} {
					_, _, indexed := k.GetProviderPoolAddress(current, pool.String())
					require.Equal(t, !success, indexed)
				}
				message, broken := keeperlib.ProviderCollateralSolvencyInvariant(k)(current)
				require.False(t, broken, message)
				if !success {
					return
				}
				elapsed := deleteHeight - 1
				refund := 21 - elapsed*3 + elapsed*3*4/10
				for _, consumer := range consumers {
					require.Equal(t, refund, bApp.BankKeeper.GetBalance(current, consumer, "ualpha").Amount.Int64())
					require.Equal(t, refund, transferAmounts[consumer.String()].AmountOf("ualpha").Int64())
				}
				wantOwner := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 60-2*refund), sdk.NewInt64Coin("ubonus", 8))
				require.Equal(t, wantOwner, bApp.BankKeeper.GetAllBalances(current, vault))
				require.Equal(t, wantOwner, transferAmounts[vault.String()])
				require.True(t, bApp.BankKeeper.GetAllBalances(current, collateral).IsZero())
				require.True(t, bApp.BankKeeper.GetAllBalances(current, earnings).IsZero())
				_, exists := k.GetProvider(current, id)
				require.False(t, exists)
				expected := balances(current)
				_, err = bApp.Commit()
				require.NoError(t, err)
				block, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: deleteHeight + 1, Time: ctx.BlockTime().Add(time.Second), Txs: [][]byte{signed(bApp.NewContext(true), messages)}})
				require.NoError(t, err)
				require.NotZero(t, block.TxResults[0].Code)
				require.Contains(t, block.TxResults[0].Log, "not found")
				require.Equal(t, expected, balances(bApp.NewContext(false)), "neither replay nor EndBlock pays a deleted provider twice")
				for _, event := range block.TxResults[0].Events {
					require.NotContains(t, []string{"transfer", "coin_spent", "coin_received", "structs.structs.EventDelete"}, event.Type)
				}
			})
		}
	}
}
