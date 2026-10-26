package app_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
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

func TestSignedProviderCreationAuthorityAndValidation(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		for _, scenario := range []string{"direct", "station_only", "owner_only", "station_owner", "guild_owner", "station_rank", "station_demoted", "station_departed", "guild_rank", "guild_demoted", "guild_departed", "key_ceiling", "missing_station", "wrong_station_kind", "missing_owner", "wrong_owner_kind", "player_gift", "zero_rate", "negative_rate", "invalid_denom", "empty_denom", "invalid_policy", "invalid_capacity", "invalid_duration", "invalid_penalty", "later_failure"} {
			if strings.HasPrefix(scenario, "guild_") && !guildOwned || scenario == "player_gift" && guildOwned {
				continue
			}
			t.Run(fmt.Sprintf("guild=%t/%s", guildOwned, scenario), func(t *testing.T) {
				bApp, ctx := setupReactorAppGenesis(t, 1, true)
				k := bApp.StructsKeeper
				newPlayer := func() types.Player {
					return appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
				}
				actor, owner, sourceOwner := newPlayer(), newPlayer(), newPlayer()
				key := secp256k1.GenPrivKey()
				address := sdk.AccAddress(key.PubKey().Address())
				require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), actor.Index))
				bits := types.PermSourceAllocation
				guild, sourceGuild := types.CreateEmptyGuild(), types.CreateEmptyGuild()
				guild.Id, guild.Index, guild.Owner = "0-994", 994, owner.Id
				sourceGuild.Id, sourceGuild.Index, sourceGuild.Owner = "0-995", 995, sourceOwner.Id
				if scenario == "guild_owner" {
					guild.Owner = actor.Id
				}
				if scenario == "station_owner" {
					sourceGuild.Owner = actor.Id
				}
				k.SetGuild(ctx, guild)
				k.SetGuild(ctx, sourceGuild)
				station := types.Substation{Id: "4-994", Owner: sourceGuild.Id}
				k.SetSubstation(ctx, station)
				requestedOwner, expectedOwner := "", actor.Id
				if guildOwned {
					requestedOwner, expectedOwner = guild.Id, guild.Id
				}
				stationGrant, guildGrant, success := true, true, true
				switch scenario {
				case "station_only":
					guildGrant, success = false, !guildOwned
				case "owner_only":
					stationGrant, success = false, false
				case "station_owner":
					stationGrant = false
				case "guild_owner":
					guildGrant = false
				case "station_rank", "station_demoted", "station_departed":
					stationGrant = false
					actor.GuildId, actor.GuildRank = sourceGuild.Id, 3
					k.SetGuildRankPermissionStoreOnly(ctx, station.Id, sourceGuild.Id, types.PermSourceAllocation, 3)
				case "guild_rank", "guild_demoted", "guild_departed":
					guildGrant = false
					actor.GuildId, actor.GuildRank = guild.Id, 3
					k.SetGuildRankPermissionStoreOnly(ctx, guild.Id, guild.Id, types.PermSourceAllocation, 3)
				case "key_ceiling":
					bits, success = types.PermAll&^types.PermSourceAllocation, false
				case "missing_owner":
					requestedOwner, success = "0-999999", false
				case "wrong_owner_kind":
					requestedOwner, success = station.Id, false
				case "player_gift":
					requestedOwner, expectedOwner = owner.Id, owner.Id
				}
				if strings.HasSuffix(scenario, "demoted") {
					actor.GuildRank, success = 4, false
				}
				if strings.HasSuffix(scenario, "departed") {
					actor.GuildId, success = "", false
				}
				k.SetPlayer(ctx, actor)
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
				if stationGrant {
					k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(station.Id, actor.Id), types.PermSourceAllocation)
				}
				if guildGrant {
					k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(guild.Id, actor.Id), types.PermSourceAllocation)
				}
				message := &types.MsgProviderCreate{Creator: address.String(), OwnerId: requestedOwner, SubstationId: station.Id,
					Rate: sdk.NewInt64Coin("ualpha", 3), AccessPolicy: types.ProviderAccessPolicy_openMarket,
					CapacityMinimum: 1, CapacityMaximum: 100, DurationMinimum: 2, DurationMaximum: 200,
					ProviderCancellationPenalty: math.LegacyNewDecWithPrec(1, 1), ConsumerCancellationPenalty: math.LegacyNewDecWithPrec(2, 1)}
				switch scenario {
				case "missing_station":
					message.SubstationId, success = "4-999999", false
				case "wrong_station_kind":
					message.SubstationId, success = actor.Id, false
				case "zero_rate":
					message.Rate.Amount = math.ZeroInt()
				case "negative_rate":
					message.Rate.Amount, success = math.NewInt(-1), false
				case "invalid_denom":
					message.Rate.Denom, success = "!", false
				case "empty_denom":
					message.Rate.Denom, success = "", false
				case "invalid_policy":
					message.AccessPolicy, success = 500, false
				case "invalid_capacity":
					message.CapacityMinimum, success = 101, false
				case "invalid_duration":
					message.DurationMinimum, success = 201, false
				case "invalid_penalty":
					message.ProviderCancellationPenalty, success = math.LegacyNewDec(2), false
				}
				messages := []sdk.Msg{message}
				if scenario == "later_failure" {
					invalid := *message
					invalid.SubstationId = "4-999999"
					messages, success = append(messages, &invalid), false
				}
				beforePermissions, beforeRanks := k.GetAllPermissionExport(ctx), k.GetAllGuildRankPermissionExport(ctx)
				beforeGrid, beforeCount := k.GetAllGridExport(ctx), k.GetProviderCount(ctx)
				providerID := keeperlib.GetObjectID(types.ObjectType_provider, beforeCount)
				collateral, earnings := keeperlib.GetProviderCollateralPoolLocation(providerID), keeperlib.GetProviderEarningsPoolLocation(providerID)
				require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, collateral))
				require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, earnings))
				balances := bApp.BankKeeper.GetSupply(ctx, "ualpha")
				signer := bApp.AccountKeeper.GetAccount(ctx, address)
				config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
				tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil, ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
				require.NoError(t, err)
				wire, err := config.TxEncoder()(tx)
				require.NoError(t, err)
				block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
				require.NoError(t, err)
				require.Len(t, block.TxResults, 1)
				result := block.TxResults[0]
				require.NotContains(t, result.Log, "panic")
				require.LessOrEqual(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
				current := bApp.NewContext(false)
				providers := k.GetAllProvider(current)
				if success {
					require.Zero(t, result.Code, result.Log)
					require.Len(t, providers, 1)
					expected := types.Provider{Id: providerID, Index: beforeCount, Owner: expectedOwner, Creator: address.String(), SubstationId: station.Id,
						Rate: message.Rate, AccessPolicy: message.AccessPolicy, CapacityMinimum: 1, CapacityMaximum: 100, DurationMinimum: 2, DurationMaximum: 200,
						ProviderCancellationPenalty: message.ProviderCancellationPenalty, ConsumerCancellationPenalty: message.ConsumerCancellationPenalty}
					require.Equal(t, expected, providers[0])
					require.Equal(t, beforeCount+1, k.GetProviderCount(current))
					for _, id := range []string{actor.Id, owner.Id, guild.Id} {
						want := types.Permissionless
						if !guildOwned && id == expectedOwner {
							want = types.PermProviderAll
						}
						require.Equal(t, want, k.GetPermissionsByBytes(current, keeperlib.GetObjectPermissionIDBytes(providerID, id)))
					}
					cc := k.NewCurrentContext(current)
					require.Equal(t, uint64(1), cc.GetProvider(providerID).GetCheckpointBlock())
					require.Zero(t, cc.GetProvider(providerID).GetAgreementLoad())
				} else {
					require.NotZero(t, result.Code, "malformed input or missing authority must not create a provider")
					if scenario == "later_failure" {
						require.Contains(t, result.Log, "message index: 1")
					}
					require.Empty(t, providers)
					require.Equal(t, beforeCount, k.GetProviderCount(current))
					require.Equal(t, beforePermissions, k.GetAllPermissionExport(current))
					require.Equal(t, beforeGrid, k.GetAllGridExport(current))
				}
				for i, pool := range []sdk.AccAddress{collateral, earnings} {
					id, kind, found := k.GetProviderPoolAddress(current, pool.String())
					require.Equal(t, success, found)
					if success {
						require.Equal(t, providerID, id)
						require.Equal(t, keeperlib.ProviderPoolKind(i+1), kind)
						require.NotNil(t, bApp.AccountKeeper.GetAccount(current, pool))
					} else {
						require.Nil(t, bApp.AccountKeeper.GetAccount(current, pool))
					}
					require.True(t, bApp.BankKeeper.GetAllBalances(current, pool).IsZero())
				}
				providerEvents, addressEvents := 0, 0
				for _, event := range result.Events {
					switch event.Type {
					case "structs.structs.EventProvider":
						decoded, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						require.Len(t, providers, 1)
						require.Equal(t, providers[0], *decoded.(*types.EventProvider).Provider)
						providerEvents++
					case "structs.structs.EventProviderAddress":
						decoded, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						require.Equal(t, &types.EventProviderAddressDetail{ProviderId: providerID, CollateralPool: collateral.String(), EarningPool: earnings.String()}, decoded.(*types.EventProviderAddress).EventProviderAddressDetail)
						addressEvents++
					case "transfer", "coin_spent", "coin_received", "structs.structs.EventVaultAddress":
						t.Fatalf("provider creation moved assets or provisioned a vault: %s", event.Type)
					}
				}
				wantEvents := 0
				if success {
					wantEvents = 1
				}
				require.Equal(t, wantEvents, providerEvents)
				require.Equal(t, wantEvents, addressEvents)
				require.Equal(t, beforeRanks, k.GetAllGuildRankPermissionExport(current))
				require.Equal(t, balances, bApp.BankKeeper.GetSupply(current, "ualpha"))
				actualStation, found := k.GetSubstation(current, station.Id)
				require.True(t, found)
				require.Equal(t, station, actualStation)
			})
		}
	}
}

func TestSignedMalformedLegacyProviderRefusesAgreementAndCanBeDeleted(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		for _, badRate := range []sdk.Coin{{Denom: "ualpha", Amount: math.NewInt(-1)}, {Denom: "!", Amount: math.OneInt()}} {
			t.Run(fmt.Sprintf("guild=%t/rate=%s", guildOwned, badRate), func(t *testing.T) {
				bApp, ctx := setupReactorAppGenesis(t, 1, true)
				k := bApp.StructsKeeper
				key := secp256k1.GenPrivKey()
				address := sdk.AccAddress(key.PubKey().Address())
				actor := appendAppPlayer(t, k, ctx, address)
				owner := actor.Id
				if guildOwned {
					guild := types.CreateEmptyGuild()
					guild.Id, guild.Index, guild.Owner = "0-996", 996, actor.Id
					k.SetGuild(ctx, guild)
					owner = guild.Id
				}
				station := types.Substation{Id: "4-996", Owner: owner}
				k.SetSubstation(ctx, station)
				k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, station.Id), 100)
				cc := k.NewCurrentContext(ctx)
				provider := cc.NewProvider(types.Provider{Owner: owner, Creator: address.String(), SubstationId: station.Id,
					Rate: badRate, AccessPolicy: types.ProviderAccessPolicy_openMarket, CapacityMinimum: 1, CapacityMaximum: 100, DurationMinimum: 1, DurationMaximum: 100,
					ProviderCancellationPenalty: math.LegacyZeroDec(), ConsumerCancellationPenalty: math.LegacyZeroDec()})
				cc.CommitAll()
				collateral, earnings := provider.GetCollateralPoolLocation(), provider.GetEarningsPoolLocation()
				collateralCoins := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 11), sdk.NewInt64Coin("uasset", 7))
				earningsCoins := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 13))
				fundRedelegationAccount(t, bApp, ctx, collateral, collateralCoins)
				fundRedelegationAccount(t, bApp, ctx, earnings, earningsCoins)
				vault, err := types.VaultAddress(owner)
				require.NoError(t, err)
				signer := bApp.AccountKeeper.GetAccount(ctx, address)
				config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
				for i, message := range []sdk.Msg{
					&types.MsgAgreementOpen{Creator: address.String(), ProviderId: provider.ID(), Capacity: 1, Duration: 10},
					&types.MsgProviderDelete{Creator: address.String(), ProviderId: provider.ID()},
				} {
					tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, []sdk.Msg{message}, nil, ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence() + uint64(i)}, key)
					require.NoError(t, err)
					wire, err := config.TxEncoder()(tx)
					require.NoError(t, err)
					block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: int64(i + 1), Time: ctx.BlockTime().Add(time.Duration(i) * time.Second), Txs: [][]byte{wire}})
					require.NoError(t, err)
					require.Len(t, block.TxResults, 1)
					result := block.TxResults[0]
					require.NotContains(t, result.Log, "panic")
					current := bApp.NewContext(false)
					require.Empty(t, k.GetAllAgreement(current))
					if i == 0 {
						require.NotZero(t, result.Code)
						require.Contains(t, result.Log, "invalid_coin")
						require.Equal(t, collateralCoins, bApp.BankKeeper.GetAllBalances(current, collateral))
						require.Equal(t, earningsCoins, bApp.BankKeeper.GetAllBalances(current, earnings))
						require.Nil(t, bApp.AccountKeeper.GetAccount(current, vault))
						for _, event := range result.Events {
							require.NotContains(t, event.Type, "structs.structs.Event")
							require.NotEqual(t, "transfer", event.Type)
						}
					} else {
						require.Zero(t, result.Code, result.Log)
						require.Empty(t, k.GetAllProvider(current))
						require.Equal(t, collateralCoins.Add(earningsCoins...), bApp.BankKeeper.GetAllBalances(current, vault))
						for _, pool := range []sdk.AccAddress{collateral, earnings} {
							require.True(t, bApp.BankKeeper.GetAllBalances(current, pool).IsZero())
							_, _, found := k.GetProviderPoolAddress(current, pool.String())
							require.False(t, found)
						}
					}
					_, err = bApp.Commit()
					require.NoError(t, err)
				}
			})
		}
	}
}
