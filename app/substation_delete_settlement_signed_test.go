package app_test

import (
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

// A consumer deleting the receiving station cannot select the provider's
// cancellation price. Deleting the source station settles immediately through
// the ordinary provider-cancellation path, including same-transaction rollback.
func TestSignedSubstationDeletionSettlementDirection(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		for _, direction := range []string{"incoming", "outgoing"} {
			for _, deleteHeight := range []int64{1, 3} {
				for _, rollback := range []bool{false, true} {
					t.Run(fmt.Sprintf("guild=%t/%s/height=%d/rollback=%t", guildOwned, direction, deleteHeight, rollback), func(t *testing.T) {
						bApp, ctx := setupReactorAppGenesis(t, 1, true)
						startTime := ctx.BlockTime()
						bApp, ctx, providerID, ids, vaults := setupFundedAgreementsOnGraph(t, bApp, ctx, 2, 0)
						k := bApp.StructsKeeper
						provider, found := k.GetProvider(ctx, providerID)
						require.True(t, found)
						agreements := map[string]types.Agreement{}
						for _, id := range ids {
							row, found := k.GetAgreement(ctx, id)
							require.True(t, found)
							agreements[id] = row
						}
						actorID, stationID := provider.Owner, provider.SubstationId
						if direction == "incoming" {
							actorID, stationID = agreements[ids[0]].Owner, "4-990"
						}
						actor, found := k.GetPlayer(ctx, actorID)
						require.True(t, found)
						owner := actorID
						if guildOwned {
							guild := types.CreateEmptyGuild()
							guild.Id, guild.Index, guild.Owner, guild.Creator = "0-990", 990, actorID, actor.Creator
							k.SetGuild(ctx, guild)
							owner = guild.Id
						}
						station, found := k.GetSubstation(ctx, stationID)
						if direction == "outgoing" {
							require.True(t, found)
						}
						station.Id, station.Owner = stationID, owner
						k.SetSubstation(ctx, station)
						cc := k.NewCurrentContext(ctx)
						allocation, found := cc.GetAllocation(agreements[ids[0]].AllocationId)
						require.True(t, found)
						if direction == "incoming" {
							require.NoError(t, allocation.SetDestination(stationID))
							cc.CommitAll()
						}
						originalAllocation := allocation.GetAllocation()
						key := secp256k1.GenPrivKey()
						address := sdk.AccAddress(key.PubKey().Address())
						require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), actor.Index))
						k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), types.PermDelete)
						collateral, earnings := keeperlib.GetProviderCollateralPoolLocation(providerID), keeperlib.GetProviderEarningsPoolLocation(providerID)
						balances := func(c sdk.Context) map[string]string {
							rows := map[string]string{}
							for _, row := range bApp.BankKeeper.GetAccountsBalances(c) {
								if !row.Coins.IsZero() {
									rows[row.Address] = row.Coins.String()
								}
							}
							return rows
						}
						beforeBalances := balances(ctx)
						for height := int64(1); height < deleteHeight; height++ {
							_, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: startTime.Add(time.Duration(height-1) * time.Second)})
							require.NoError(t, err)
							_, err = bApp.Commit()
							require.NoError(t, err)
						}
						if deleteHeight > 1 {
							ctx = bApp.NewUncachedContext(false, ctx.BlockHeader()).WithBlockHeight(deleteHeight).WithBlockTime(startTime.Add(time.Duration(deleteHeight-1) * time.Second))
						}
						signer := bApp.AccountKeeper.GetAccount(ctx, address)
						messages := []sdk.Msg{&types.MsgSubstationDelete{Creator: address.String(), SubstationId: stationID}}
						if rollback {
							messages = append(messages, &types.MsgSubstationDelete{Creator: address.String(), SubstationId: "4-999999"})
						}
						config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
						tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil, ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
						require.NoError(t, err)
						wire, err := config.TxEncoder()(tx)
						require.NoError(t, err)
						block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: deleteHeight, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
						require.NoError(t, err)
						require.Len(t, block.TxResults, 1)
						result := block.TxResults[0]
						if rollback {
							require.NotZero(t, result.Code)
							require.Contains(t, result.Log, "message index: 1")
						} else {
							require.Zero(t, result.Code, result.Log)
						}
						require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))

						retired := direction == "outgoing" && !rollback
						paid := map[string]sdk.Coins{}
						for _, event := range result.Events {
							require.False(t, strings.HasPrefix(event.Type, "structs.structs.EventService"))
							if rollback {
								require.NotContains(t, []string{"transfer", "coin_spent", "coin_received", "structs.structs.EventDelete", "structs.structs.EventAllocation"}, event.Type)
							}
							if event.Type == "transfer" {
								attrs := map[string]string{}
								for _, a := range event.Attributes {
									attrs[a.Key] = a.Value
								}
								require.Equal(t, collateral.String(), attrs["sender"])
								coins, err := sdk.ParseCoinsNormalized(attrs["amount"])
								require.NoError(t, err)
								paid[attrs["recipient"]] = paid[attrs["recipient"]].Add(coins...)
							}
						}
						current := bApp.NewContext(false)
						actualStation, found := k.GetSubstation(current, stationID)
						require.Equal(t, rollback, found)
						if rollback {
							require.Equal(t, station, actualStation)
						}
						actualProvider, found := k.GetProvider(current, providerID)
						require.True(t, found)
						require.Equal(t, provider, actualProvider)
						for _, id := range ids {
							row, live := k.GetAgreement(current, id)
							require.Equal(t, !retired, live)
							if live {
								require.Equal(t, agreements[id], row)
							}
							_, live = k.GetAllocation(current, agreements[id].AllocationId)
							require.Equal(t, !retired, live)
						}
						if direction == "incoming" && !rollback {
							disconnected, found := k.GetAllocation(current, originalAllocation.Id)
							require.True(t, found)
							originalAllocation.DestinationId = ""
							require.Equal(t, originalAllocation, disconnected)
						}
						if !retired {
							require.Equal(t, beforeBalances, balances(current))
							require.Empty(t, paid)
						} else {
							elapsed := deleteHeight - 1
							refund := 21 - elapsed*3 + elapsed*3*4/10
							earned := elapsed * 3 * 2 * 6 / 10
							require.Equal(t, earned, bApp.BankKeeper.GetBalance(current, earnings, "ualpha").Amount.Int64())
							require.Equal(t, 42-2*refund-earned, bApp.BankKeeper.GetBalance(current, collateral, "ualpha").Amount.Int64())
							for _, vault := range vaults {
								require.Equal(t, refund, bApp.BankKeeper.GetBalance(current, vault, "ualpha").Amount.Int64())
								require.Equal(t, refund, paid[vault.String()].AmountOf("ualpha").Int64())
							}
							require.Equal(t, earned, paid[earnings.String()].AmountOf("ualpha").Int64())
						}
						message, broken := keeperlib.ProviderCollateralSolvencyInvariant(k)(current)
						require.False(t, broken, message)
						_, err = bApp.Commit()
						require.NoError(t, err)
						// Agreements left alive by destination deletion or rollback
						// expire at height 8; no historical payment worker is needed.
						for height := deleteHeight + 1; height <= 8; height++ {
							_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: startTime.Add(time.Duration(height-1) * time.Second)})
							require.NoError(t, err)
							_, err = bApp.Commit()
							require.NoError(t, err)
						}
						current = bApp.NewContext(true)
						require.Zero(t, k.NewCurrentContext(current).GetProvider(providerID).GetAgreementLoad())
						for _, id := range ids {
							_, live := k.GetAgreement(current, id)
							require.False(t, live)
						}
						if !retired {
							for _, vault := range vaults {
								require.True(t, bApp.BankKeeper.GetBalance(current, vault, "ualpha").IsZero())
							}
							require.Equal(t, int64(41), bApp.BankKeeper.GetBalance(current, earnings, "ualpha").Amount.Int64())
							require.Equal(t, int64(1), bApp.BankKeeper.GetBalance(current, collateral, "ualpha").Amount.Int64())
						}
					})
				}
			}
		}
	}
}
