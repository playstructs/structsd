package app_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Empty offerings isolate signed authority and payout atomicity. Outstanding
// consumer obligations have separate settlement/retirement integration tests.
func TestSignedProviderPayoutAuthorityAndRollback(t *testing.T) {
	for _, operation := range []string{"withdraw", "delete"} {
		scenarios := []string{"owner", "delegate", "rank", "demoted", "departed", "member_only", "guild_grant", "wrong_action", "key_ceiling", "owner_key_ceiling", "rank_key_ceiling", "missing", "wrong_kind", "later_failure", "payout_failure", "send_disabled"}
		if operation == "withdraw" {
			scenarios = append(scenarios, "external", "invalid_recipient", "blocked_recipient", "repeat_withdraw")
		}
		for _, guildOwned := range []bool{false, true} {
			for _, scenario := range scenarios {
				t.Run(fmt.Sprintf("%s/guild=%t/%s", operation, guildOwned, scenario), func(t *testing.T) {
					bApp, ctx := setupReactorAppGenesis(t, 1, true)
					k := bApp.StructsKeeper
					newPlayer := func() types.Player {
						return appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
					}
					actor, owner, witness := newPlayer(), newPlayer(), newPlayer()
					if scenario == "owner" || scenario == "owner_key_ceiling" {
						owner = actor
					}
					key := secp256k1.GenPrivKey()
					address := sdk.AccAddress(key.PubKey().Address())
					require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), actor.Index))
					guild := types.CreateEmptyGuild()
					guild.Id, guild.Index, guild.Owner, guild.Creator = "0-979", 979, owner.Id, owner.Creator
					k.SetGuild(ctx, guild)
					ownerID := owner.Id
					if guildOwned {
						ownerID = guild.Id
					}
					k.SetSubstation(ctx, types.Substation{Id: "4-979", Owner: ownerID})
					cc := k.NewCurrentContext(ctx)
					provider := cc.NewProvider(types.Provider{Owner: ownerID, Creator: owner.Creator, SubstationId: "4-979",
						Rate: sdk.NewInt64Coin("ualpha", 3), AccessPolicy: types.ProviderAccessPolicy_openMarket,
						CapacityMinimum: 1, CapacityMaximum: 1000, DurationMinimum: 1, DurationMaximum: 100,
						ProviderCancellationPenalty: math.LegacyZeroDec(), ConsumerCancellationPenalty: math.LegacyZeroDec()})
					cc.CommitAll()
					original := provider.GetProvider()
					collateral, earnings := provider.GetCollateralPoolLocation(), provider.GetEarningsPoolLocation()
					collateralCoins := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 11), sdk.NewInt64Coin("ubonus", 3))
					earningsCoins := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 7), sdk.NewInt64Coin("ubonus", 5))
					fundRedelegationAccount(t, bApp, ctx, collateral, collateralCoins)
					fundRedelegationAccount(t, bApp, ctx, earnings, earningsCoins)
					vault, err := types.VaultAddress(ownerID)
					require.NoError(t, err)
					require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, vault))
					_, associated := k.GetVaultOwner(ctx, vault)
					require.False(t, associated)
					permission, otherPermission := types.PermProviderWithdraw, types.PermDelete
					if operation == "delete" {
						permission, otherPermission = otherPermission, permission
					}
					bits, grant, wantError := permission, true, ""
					switch scenario {
					case "owner", "owner_key_ceiling":
						grant = false
						require.Empty(t, actor.GuildId)
					case "rank", "demoted", "departed", "rank_key_ceiling":
						grant = false
						actor.GuildId, actor.GuildRank = guild.Id, 3
						if scenario == "demoted" {
							actor.GuildRank, wantError = 4, "permission"
						}
						if scenario == "departed" {
							actor.GuildId, wantError = "", "permission"
						}
						k.SetPlayer(ctx, actor)
						k.SetGuildRankPermissionStoreOnly(ctx, provider.ID(), guild.Id, permission, 3)
					case "member_only":
						grant, wantError = false, "permission"
						actor.GuildId, actor.GuildRank = guild.Id, 1
						k.SetPlayer(ctx, actor)
					case "guild_grant":
						grant, wantError = false, "permission"
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(guild.Id, actor.Id), permission)
					case "wrong_action":
						grant, wantError = false, "permission"
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(provider.ID(), actor.Id), otherPermission)
					}
					if grant {
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(provider.ID(), actor.Id), permission)
					}
					if strings.HasSuffix(scenario, "key_ceiling") {
						bits, wantError = types.PermAll&^permission, "address lacks required permission bits"
					}
					k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
					// Deletion must clear all of this provider's grants while
					// preserving the same grantee's unrelated object authority.
					k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(provider.ID(), witness.Id), types.PermUpdate)
					k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(guild.Id, witness.Id), types.PermUpdate)
					k.SetGuildRankPermissionStoreOnly(ctx, provider.ID(), guild.Id, types.PermUpdate, 9)
					id, destination, recipient := provider.ID(), "", vault
					switch scenario {
					case "missing":
						id, wantError = "10-999999", "not found"
					case "wrong_kind":
						id, wantError = actor.Id, "not found"
					case "later_failure":
						wantError = "message index: 1"
					case "external":
						recipient, destination = sdk.MustAccAddressFromBech32(witness.Creator), witness.Creator
					case "invalid_recipient":
						destination, wantError = "invalid", "couldn't be validated as a real address"
					case "blocked_recipient":
						destination, wantError = bApp.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName).String(), "blocked_recipient"
					case "send_disabled":
						bApp.BankKeeper.SetSendEnabled(ctx, "ubonus", false)
						wantError = "send transactions are disabled"
					case "payout_failure":
						wantError = "forced payout failure"
					}
					failureReached := false
					if scenario == "payout_failure" {
						bApp.BankKeeper.AppendSendRestriction(func(goCtx context.Context, from, to sdk.AccAddress, coins sdk.Coins) (sdk.AccAddress, error) {
							if from.Equals(earnings) {
								failureReached = true
								_, found := k.GetVaultOwner(goCtx, vault)
								require.True(t, found, "vault provisioning precedes the failed send")
								if operation == "delete" {
									require.Empty(t, bApp.BankKeeper.GetAllBalances(goCtx, collateral), "the first transfer must already have executed")
									require.Equal(t, collateralCoins.Add(earningsCoins...), coins)
								}
								return to, errors.New("forced payout failure")
							}
							return to, nil
						})
					}
					message := func(providerID string) sdk.Msg {
						if operation == "delete" {
							return &types.MsgProviderDelete{Creator: address.String(), ProviderId: providerID}
						}
						return &types.MsgProviderWithdrawBalance{Creator: address.String(), ProviderId: providerID, DestinationAddress: destination}
					}
					messages := []sdk.Msg{message(id)}
					if scenario == "later_failure" {
						messages = append(messages, message("10-999999"))
					}
					if scenario == "repeat_withdraw" {
						messages = append(messages, message(id))
					}
					balances := func(ctx sdk.Context) map[string]string {
						rows := map[string]string{}
						for _, row := range bApp.BankKeeper.GetAccountsBalances(ctx) {
							if !row.Coins.IsZero() {
								rows[row.Address] = row.Coins.String()
							}
						}
						return rows
					}
					expectedBalances := balances(ctx)
					beforeAssets := bankAndAuthStateRows(bApp, ctx)
					beforePermissions, beforeRanks := k.GetAllPermissionExport(ctx), k.GetAllGuildRankPermissionExport(ctx)
					beforeGrid, beforeCount := k.GetAllGridExport(ctx), k.GetProviderCount(ctx)
					signer := bApp.AccountKeeper.GetAccount(ctx, address)
					config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
					tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil,
						ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
					require.NoError(t, err)
					wire, err := config.TxEncoder()(tx)
					require.NoError(t, err)
					block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
					require.NoError(t, err)
					require.Len(t, block.TxResults, 1)
					result := block.TxResults[0]
					current := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
					if wantError == "" {
						require.Zero(t, result.Code, result.Log)
						require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
					} else {
						require.NotZero(t, result.Code, "refused payout must not move assets: recipient=%s collateral=%s earnings=%s", bApp.BankKeeper.GetAllBalances(current, recipient), bApp.BankKeeper.GetAllBalances(current, collateral), bApp.BankKeeper.GetAllBalances(current, earnings))
						require.Contains(t, result.Log, wantError)
						require.NotContains(t, result.Log, "panic")
					}
					require.Equal(t, scenario == "payout_failure", failureReached)
					deleted := operation == "delete" && wantError == ""
					actual, found := k.GetProvider(current, original.Id)
					require.Equal(t, !deleted, found)
					if !deleted {
						require.Equal(t, original, actual)
					}
					require.Equal(t, beforeCount, k.GetProviderCount(current))
					expectedGrid, actualGrid := map[string]uint64{}, map[string]uint64{}
					for _, row := range beforeGrid {
						expectedGrid[row.AttributeId] = row.Value
					}
					if deleted {
						delete(expectedGrid, provider.CheckpointBlockAttributeId)
						delete(expectedGrid, provider.AgreementLoadAttributeId)
					}
					for _, row := range k.GetAllGridExport(current) {
						actualGrid[row.AttributeId] = row.Value
					}
					require.Equal(t, expectedGrid, actualGrid)
					expectedPermissions, actualPermissions := map[string]uint64{}, map[string]uint64{}
					for _, row := range beforePermissions {
						if !deleted || !strings.HasPrefix(row.PermissionId, provider.ID()+"@") {
							expectedPermissions[row.PermissionId] = row.Value
						}
					}
					for _, row := range k.GetAllPermissionExport(current) {
						actualPermissions[row.PermissionId] = row.Value
					}
					require.Equal(t, expectedPermissions, actualPermissions)
					expectedRanks := []*types.GuildRankPermissionRecord{}
					for _, row := range beforeRanks {
						if !deleted || row.ObjectId != provider.ID() {
							expectedRanks = append(expectedRanks, row)
						}
					}
					require.ElementsMatch(t, expectedRanks, k.GetAllGuildRankPermissionExport(current))
					for _, pool := range []sdk.AccAddress{collateral, earnings} {
						_, _, indexed := k.GetProviderPoolAddress(current, pool.String())
						require.Equal(t, !deleted, indexed)
					}
					expectedClearedPermissions, clearedPermissions := map[string]uint64{}, map[string]uint64{}
					expectedClearedRanks, clearedRanks := map[string]uint64{}, map[string]uint64{}
					expectedGridEvents, gridEvents := map[string]uint64{}, map[string]uint64{}
					if deleted {
						for _, row := range beforePermissions {
							if strings.HasPrefix(row.PermissionId, provider.ID()+"@") {
								expectedClearedPermissions[row.PermissionId] = 0
							}
						}
						for _, row := range beforeRanks {
							if row.ObjectId == provider.ID() {
								expectedClearedRanks[fmt.Sprintf("%s/%s/%d", row.ObjectId, row.GuildId, row.Permissions)] = 0
							}
						}
						expectedGridEvents[provider.CheckpointBlockAttributeId], expectedGridEvents[provider.AgreementLoadAttributeId] = 0, 0
					}
					var transfers []string
					vaultEvents, deletionEvents := 0, 0
					for _, event := range result.Events {
						if wantError != "" {
							require.NotContains(t, []string{"transfer", "coin_spent", "coin_received", "structs.structs.EventVaultAddress", "structs.structs.EventDelete", "structs.structs.EventPermission", "structs.structs.EventGuildRankPermission", "structs.structs.EventGrid"}, event.Type)
						}
						require.NotEqual(t, "structs.structs.EventProvider", event.Type)
						switch event.Type {
						case "structs.structs.EventPermission":
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							row := parsed.(*types.EventPermission).PermissionRecord
							_, duplicate := clearedPermissions[row.PermissionId]
							require.False(t, duplicate)
							clearedPermissions[row.PermissionId] = row.Value
						case "structs.structs.EventGuildRankPermission":
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							row := parsed.(*types.EventGuildRankPermission).GuildRankPermissionRecord
							key := fmt.Sprintf("%s/%s/%d", row.ObjectId, row.GuildId, row.Permissions)
							_, duplicate := clearedRanks[key]
							require.False(t, duplicate)
							clearedRanks[key] = row.Rank
						case "structs.structs.EventGrid":
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							row := parsed.(*types.EventGrid).GridRecord
							_, duplicate := gridEvents[row.AttributeId]
							require.False(t, duplicate)
							gridEvents[row.AttributeId] = row.Value
						case "transfer":
							attributes := map[string]string{}
							for _, a := range event.Attributes {
								attributes[a.Key] = a.Value
							}
							transfers = append(transfers, attributes["sender"]+">"+attributes["recipient"]+":"+attributes["amount"])
						case "structs.structs.EventVaultAddress":
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							row := parsed.(*types.EventVaultAddress).EventVaultAddressDetail
							require.Equal(t, ownerID, row.OwnerId)
							require.Equal(t, vault.String(), row.VaultAddress)
							require.True(t, row.AccountCreated)
							require.Equal(t, bApp.AccountKeeper.GetAccount(current, vault).GetAccountNumber(), row.AccountNumber)
							vaultEvents++
						case "structs.structs.EventDelete":
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							require.Equal(t, provider.ID(), parsed.(*types.EventDelete).ObjectId)
							deletionEvents++
						}
					}
					require.Equal(t, expectedClearedPermissions, clearedPermissions)
					require.Equal(t, expectedClearedRanks, clearedRanks)
					require.Equal(t, expectedGridEvents, gridEvents)
					if wantError == "" {
						payout := earningsCoins
						if deleted {
							payout = payout.Add(collateralCoins...)
							delete(expectedBalances, collateral.String())
						}
						delete(expectedBalances, earnings.String())
						expectedBalances[recipient.String()] = payout.String()
						expectedTransfers := []string{earnings.String() + ">" + recipient.String() + ":" + payout.String()}
						if deleted {
							expectedTransfers = append([]string{collateral.String() + ">" + earnings.String() + ":" + collateralCoins.String()}, expectedTransfers...)
						}
						if scenario == "repeat_withdraw" {
							expectedTransfers = append(expectedTransfers, earnings.String()+">"+recipient.String()+":")
						}
						require.Equal(t, expectedTransfers, transfers)
						if scenario == "external" {
							require.Zero(t, vaultEvents)
						} else {
							require.Equal(t, 1, vaultEvents)
						}
					} else {
						require.Empty(t, transfers)
						require.Zero(t, vaultEvents)
						afterAssets := bankAndAuthStateRows(bApp, current)
						signerKey, err := collections.EncodeKeyWithPrefix(authtypes.AddressStoreKeyPrefix.Bytes(), sdk.AccAddressKey, address)
						require.NoError(t, err)
						delete(beforeAssets, authtypes.StoreKey+"/"+string(signerKey))
						delete(afterAssets, authtypes.StoreKey+"/"+string(signerKey))
						require.Equal(t, beforeAssets, afterAssets)
					}
					if deleted {
						require.Equal(t, 1, deletionEvents)
					} else {
						require.Zero(t, deletionEvents)
					}
					require.Equal(t, expectedBalances, balances(current), "every account's balances must match the exact intended movement")
					vaultOwner, associated := k.GetVaultOwner(current, vault)
					wantVault := wantError == "" && scenario != "external"
					require.Equal(t, wantVault, associated)
					if wantVault {
						require.Equal(t, ownerID, vaultOwner)
					} else {
						require.Nil(t, bApp.AccountKeeper.GetAccount(current, vault))
					}
					require.Zero(t, k.GetPlayerIndexFromAddress(current, vault.String()), "a vault is not a signing-key association")
					wantSequence := signer.GetSequence() + 1
					if strings.HasSuffix(scenario, "key_ceiling") {
						wantSequence--
					}
					afterSigner := bApp.AccountKeeper.GetAccount(current, address)
					require.Equal(t, signer.GetAccountNumber(), afterSigner.GetAccountNumber())
					require.Equal(t, wantSequence, afterSigner.GetSequence())
					if scenario == "send_disabled" {
						// Governance's policy can later reopen the denomination.
						// The refused transaction must leave the complete payout
						// reachable for a newly signed transaction after commit.
						bApp.BankKeeper.SetSendEnabled(current, "ubonus", true)
						// FinalizeBlock already flushed this cache while computing
						// its app hash. Persist this test-only policy fixture too;
						// the test does not exercise a governance proposal.
						current.MultiStore().(storetypes.CacheMultiStore).Write()
						_, err = bApp.Commit()
						require.NoError(t, err)
						retry, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(2)), config, []sdk.Msg{message(original.Id)}, nil,
							ante.DefaultFreeGasCap, jailGateChainID, []uint64{afterSigner.GetAccountNumber()}, []uint64{wantSequence}, key)
						require.NoError(t, err)
						retryWire, err := config.TxEncoder()(retry)
						require.NoError(t, err)
						next, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: ctx.BlockTime().Add(time.Second), Txs: [][]byte{retryWire}})
						require.NoError(t, err)
						require.Len(t, next.TxResults, 1)
						require.Zero(t, next.TxResults[0].Code, next.TxResults[0].Log)
						continued := bApp.NewContext(false)
						payout := earningsCoins
						if operation == "delete" {
							payout = payout.Add(collateralCoins...)
							delete(expectedBalances, collateral.String())
						}
						delete(expectedBalances, earnings.String())
						expectedBalances[vault.String()] = payout.String()
						require.Equal(t, expectedBalances, balances(continued))
						_, present := k.GetProvider(continued, original.Id)
						require.Equal(t, operation == "withdraw", present)
						actualOwner, provisioned := k.GetVaultOwner(continued, vault)
						require.True(t, provisioned)
						require.Equal(t, ownerID, actualOwner)
						require.Equal(t, wantSequence+1, bApp.AccountKeeper.GetAccount(continued, address).GetSequence())
						payoutEvents := 0
						for _, event := range next.TxResults[0].Events {
							if event.Type == "transfer" {
								payoutEvents++
							}
						}
						if operation == "delete" {
							require.Equal(t, 2, payoutEvents)
						} else {
							require.Equal(t, 1, payoutEvents)
						}
					}
					if deleted {
						// A separately signed retry after a committed block cannot
						// pay again or recreate a removed provider/permission row.
						_, err = bApp.Commit()
						require.NoError(t, err)
						retry, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(2)), config, []sdk.Msg{message(original.Id)}, nil,
							ante.DefaultFreeGasCap, jailGateChainID, []uint64{afterSigner.GetAccountNumber()}, []uint64{wantSequence}, key)
						require.NoError(t, err)
						retryWire, err := config.TxEncoder()(retry)
						require.NoError(t, err)
						next, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: ctx.BlockTime().Add(time.Second), Txs: [][]byte{retryWire}})
						require.NoError(t, err)
						require.Len(t, next.TxResults, 1)
						require.NotZero(t, next.TxResults[0].Code)
						require.Contains(t, next.TxResults[0].Log, "not found")
						require.NotContains(t, next.TxResults[0].Log, "panic")
						continued := bApp.NewContext(false)
						_, present := k.GetProvider(continued, original.Id)
						require.False(t, present)
						require.Equal(t, expectedBalances, balances(continued))
						require.Equal(t, wantSequence+1, bApp.AccountKeeper.GetAccount(continued, address).GetSequence())
						for _, event := range next.TxResults[0].Events {
							require.False(t, strings.HasPrefix(event.Type, "structs.structs.Event"), event.Type)
							require.NotContains(t, []string{"transfer", "coin_spent", "coin_received"}, event.Type)
						}
					}
				})
			}
		}
	}
}
