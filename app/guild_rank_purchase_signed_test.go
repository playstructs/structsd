package app_test

import (
	"bytes"
	"math/rand"
	"strings"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestSignedRankPurchasePermissionAndBatchRollback(t *testing.T) {
	for _, scenario := range []string{"quote_too_low", "admission_quote_too_low", "target_outside_range", "admission_target_outside_range", "stale_rank", "admission_stale_guild", "success", "free_success", "admission_free_success", "missing_membership_bit", "admission_missing_membership_bit", "free_missing_membership_bit", "admission_free_missing_membership_bit", "missing_token_bit", "stale_second_message", "admission_success", "admission_disabled", "admission_duplicate", "admission_missing_token_bit", "free_missing_token_bit", "admission_free_missing_token_bit", "vesting_locked", "admission_vesting_locked", "admission_closing_substation"} {
		t.Run(scenario, func(t *testing.T) {
			admission := strings.HasPrefix(scenario, "admission_")
			bApp, ctx := setupJailGateApp(t)
			key := secp256k1.GenPrivKey()
			address := sdk.AccAddress(key.PubKey().Address())
			buyer := appendAppPlayer(t, bApp.StructsKeeper, ctx, address)
			owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x73}, 20)))
			guild := types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner, guild.Creator = "0-913", 913, owner.Id, owner.Creator
			guild.EntryRank, guild.RankPurchaseMinimumRank = 5, 1
			guild.RankPurchaseBasePrice, guild.RankPurchaseStepIncrease = 10, 2
			if strings.Contains(scenario, "free_") {
				guild.RankPurchaseBasePrice, guild.RankPurchaseStepIncrease = 0, 0
			}
			guild.JoinByRankPurchaseEnabled = admission && scenario != "admission_disabled"
			if scenario == "admission_closing_substation" {
				guild.EntrySubstationId = "4-913"
				bApp.StructsKeeper.SetSubstation(ctx, types.Substation{Id: guild.EntrySubstationId, Owner: owner.Id, DeletionPending: true})
			}
			bApp.StructsKeeper.SetGuild(ctx, guild)
			buyer.GuildId, buyer.GuildRank = guild.Id, 5
			if admission {
				buyer.GuildId, buyer.GuildRank = "", 0
			}
			bApp.StructsKeeper.SetPlayer(ctx, buyer)
			vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(buyer.Id)
			require.NoError(t, err)
			denom := "uguild." + guild.Id
			initialTokens := int64(100)
			if scenario == "free_success" || scenario == "admission_free_success" {
				initialTokens = 0
			} else {
				fundRedelegationAccount(t, bApp, ctx, vaults[0], sdk.NewCoins(sdk.NewInt64Coin(denom, initialTokens)))
			}
			supplyBefore := bApp.BankKeeper.GetSupply(ctx, denom)
			var lockedAccount *types.VaultVestingAccount
			if strings.HasSuffix(scenario, "vesting_locked") {
				schedule, err := vestingtypes.NewDelayedVestingAccount(authtypes.NewBaseAccountWithAddress(address), sdk.NewCoins(sdk.NewInt64Coin(denom, 100)), ctx.BlockTime().Add(time.Hour).Unix())
				require.NoError(t, err)
				lockedAccount, err = types.NewVaultVestingAccount(buyer.Id, bApp.AccountKeeper.GetAccount(ctx, vaults[0]).(*authtypes.BaseAccount), nil, schedule)
				require.NoError(t, err)
				bApp.AccountKeeper.SetAccount(ctx, lockedAccount)
				require.True(t, bApp.BankKeeper.SpendableCoins(ctx, vaults[0]).IsZero())
			}
			treasury, err := types.VaultAddress(guild.Id)
			require.NoError(t, err)
			if strings.HasSuffix(scenario, "missing_token_bit") {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), types.PermAll&^types.PermTokenTransfer)
			}
			if strings.HasSuffix(scenario, "missing_membership_bit") {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), types.PermAll&^types.PermGuildMembership)
			}
			targetRank, maxPrice, expectedRank := uint64(3), uint64(22), uint64(5)
			expectedGuild := ""
			if strings.HasSuffix(scenario, "quote_too_low") {
				maxPrice = 21
			}
			if strings.HasSuffix(scenario, "target_outside_range") {
				targetRank = 6
			}
			if scenario == "stale_rank" {
				expectedRank = 4
			}
			if scenario == "admission_stale_guild" {
				expectedGuild = "0-999"
			}
			var message sdk.Msg = &types.MsgGuildRankPurchase{Creator: address.String(), GuildId: guild.Id, ExpectedRank: expectedRank, TargetRank: targetRank, MaxPrice: maxPrice}
			if admission {
				message = &types.MsgGuildMembershipJoinByRankPurchase{Creator: address.String(), GuildId: guild.Id, ExpectedGuildId: expectedGuild, TargetRank: targetRank, MaxPrice: maxPrice}
			}
			messages := []sdk.Msg{message}
			if scenario == "stale_second_message" || scenario == "admission_duplicate" {
				messages = append(messages, message)
			}
			signer := bApp.AccountKeeper.GetAccount(ctx, address)
			config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil,
				ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
			require.NoError(t, err)
			encoded, err := config.TxEncoder()(tx)
			require.NoError(t, err)
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{encoded}})
			require.NoError(t, err)
			require.Len(t, block.TxResults, 1)
			result := block.TxResults[0]
			finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
			player, found := bApp.StructsKeeper.GetPlayer(finalCtx, buyer.Id)
			require.True(t, found)
			require.Equal(t, supplyBefore, bApp.BankKeeper.GetSupply(finalCtx, denom), "rank purchases neither mint nor burn guild tokens")
			if strings.HasSuffix(scenario, "success") {
				require.Zero(t, result.Code, result.Log)
				require.Equal(t, uint64(3), player.GuildRank)
				require.Equal(t, guild.Id, player.GuildId)
				price := int64(22)
				if strings.Contains(scenario, "free_") {
					price = 0
					require.Nil(t, bApp.AccountKeeper.GetAccount(finalCtx, treasury), "free purchase needs no treasury provisioning")
					_, indexed := bApp.StructsKeeper.GetVaultOwner(finalCtx, treasury)
					require.False(t, indexed)
				}
				require.Equal(t, initialTokens-price, bApp.BankKeeper.GetBalance(finalCtx, vaults[0], denom).Amount.Int64())
				require.Equal(t, price, bApp.BankKeeper.GetBalance(finalCtx, treasury, denom).Amount.Int64())
				purchaseEvents, vaultEvents, transfers := 0, 0, 0
				for _, event := range result.Events {
					require.NotContains(t, []string{"coinbase", "burn"}, event.Type)
					switch event.Type {
					case "structs.structs.EventGuildRankPurchase":
						decoded, decodeErr := sdk.ParseTypedEvent(event)
						require.NoError(t, decodeErr)
						require.Equal(t, &types.EventGuildRankPurchase{GuildId: guild.Id, PlayerId: buyer.Id,
							PreviousRank: buyer.GuildRank, Rank: 3, Price: uint64(price), Denom: denom, Joined: admission}, decoded)
						purchaseEvents++
					case "structs.structs.EventVaultAddress":
						require.Positive(t, price, "free purchases do not provision a treasury")
						decoded, decodeErr := sdk.ParseTypedEvent(event)
						require.NoError(t, decodeErr)
						detail := decoded.(*types.EventVaultAddress).EventVaultAddressDetail
						require.Equal(t, guild.Id, detail.OwnerId)
						require.Equal(t, treasury.String(), detail.VaultAddress)
						require.True(t, detail.AccountCreated)
						require.Equal(t, bApp.AccountKeeper.GetAccount(finalCtx, treasury).GetAccountNumber(), detail.AccountNumber)
						vaultEvents++
					case "transfer":
						attributes := make(map[string]string)
						for _, attribute := range event.Attributes {
							attributes[attribute.Key] = attribute.Value
						}
						require.Equal(t, vaults[0].String(), attributes["sender"])
						require.Equal(t, treasury.String(), attributes["recipient"])
						require.Equal(t, sdk.NewInt64Coin(denom, price).String(), attributes["amount"])
						transfers++
					case "coin_spent", "coin_received":
						require.Positive(t, price, "free purchases do not emit bank movement")
					}
				}
				require.Equal(t, 1, purchaseEvents)
				if price == 0 {
					require.Zero(t, vaultEvents)
					require.Zero(t, transfers)
				} else {
					require.Equal(t, 1, vaultEvents)
					require.Equal(t, 1, transfers)
				}
			} else {
				require.NotZero(t, result.Code)
				if strings.HasSuffix(scenario, "quote_too_low") {
					require.Contains(t, result.Log, "quoted_price_exceeded")
				} else if strings.HasSuffix(scenario, "target_outside_range") {
					require.Contains(t, result.Log, "outside_purchase_range")
				} else if scenario == "stale_rank" {
					require.Contains(t, result.Log, "rank_changed")
				} else if scenario == "admission_stale_guild" {
					require.Contains(t, result.Log, "membership_changed")
				} else if scenario == "stale_second_message" {
					require.Contains(t, result.Log, "rank_changed", "must reach the second handler after the first purchase")
				} else if scenario == "admission_duplicate" {
					require.Contains(t, result.Log, "message index: 1", "must reach the second handler after admission")
					require.Contains(t, result.Log, "already a member")
				} else if scenario == "admission_disabled" {
					require.Contains(t, result.Log, "rank_purchase_admission_disabled")
				} else if scenario == "admission_closing_substation" {
					require.Contains(t, result.Log, "substation_deleting")
				} else if lockedAccount != nil {
					require.Contains(t, result.Log, "spendable")
					require.Equal(t, lockedAccount, bApp.AccountKeeper.GetAccount(finalCtx, vaults[0]))
				} else {
					require.Contains(t, result.Log, "permission")
				}
				require.Equal(t, buyer.GuildRank, player.GuildRank)
				require.Equal(t, buyer.GuildId, player.GuildId)
				require.Equal(t, int64(100), bApp.BankKeeper.GetBalance(finalCtx, vaults[0], denom).Amount.Int64())
				require.Nil(t, bApp.AccountKeeper.GetAccount(finalCtx, treasury), "failed batch must discard first-message vault provisioning")
				_, indexed := bApp.StructsKeeper.GetVaultOwner(finalCtx, treasury)
				require.False(t, indexed, "failed batch must discard vault ownership index too")
				for _, event := range result.Events {
					require.NotContains(t, []string{"structs.structs.EventGuildRankPurchase", "structs.structs.EventVaultAddress",
						"transfer", "coin_spent", "coin_received", "coinbase", "burn"}, event.Type,
						"failed transactions must not publish rolled-back purchases, assets or associations")
				}
			}
		})
	}
}

func TestSignedRankSaleConfigurationAuthorityAndRollback(t *testing.T) {
	for _, scenario := range []string{"owner", "outsider", "restricted_owner", "invalid_second_update"} {
		t.Run(scenario, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			key := secp256k1.GenPrivKey()
			address := sdk.AccAddress(key.PubKey().Address())
			actor := appendAppPlayer(t, bApp.StructsKeeper, ctx, address)
			other := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x74}, 20)))
			guild := types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner, guild.Creator = "0-914", 914, actor.Id, actor.Creator
			guild.EntryRank = 5
			if scenario == "outsider" {
				guild.Owner = other.Id
			}
			bApp.StructsKeeper.SetGuild(ctx, guild)
			if scenario == "restricted_owner" {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), types.PermAll&^types.PermAdmin)
			}
			messages := []sdk.Msg{&types.MsgGuildUpdateRankPurchase{Creator: address.String(), GuildId: guild.Id,
				BasePrice: 10, StepIncrease: 2, MinimumRank: 1, JoinByRankPurchaseEnabled: true}}
			if scenario == "invalid_second_update" {
				messages = append(messages, &types.MsgGuildUpdateRankPurchase{Creator: address.String(), GuildId: guild.Id,
					BasePrice: 99, MinimumRank: 6, JoinByRankPurchaseEnabled: true})
			}
			signer := bApp.AccountKeeper.GetAccount(ctx, address)
			config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil,
				ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
			require.NoError(t, err)
			encoded, err := config.TxEncoder()(tx)
			require.NoError(t, err)
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{encoded}})
			require.NoError(t, err)
			require.Len(t, block.TxResults, 1)
			result := block.TxResults[0]
			finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
			actual, found := bApp.StructsKeeper.GetGuild(finalCtx, guild.Id)
			require.True(t, found)
			if scenario == "owner" {
				require.Zero(t, result.Code, result.Log)
				require.Equal(t, uint64(10), actual.RankPurchaseBasePrice)
				require.Equal(t, uint64(2), actual.RankPurchaseStepIncrease)
				require.Equal(t, uint64(1), actual.RankPurchaseMinimumRank)
				require.True(t, actual.JoinByRankPurchaseEnabled)
			} else {
				require.NotZero(t, result.Code)
				if scenario == "invalid_second_update" {
					require.Contains(t, result.Log, "message index: 1")
				} else {
					require.Contains(t, result.Log, "permission")
				}
				require.Equal(t, guild, actual, "failed configuration must leave every guild field unchanged")
				for _, event := range result.Events {
					require.NotEqual(t, "structs.structs.EventGuild", event.Type)
				}
			}
		})
	}
}
