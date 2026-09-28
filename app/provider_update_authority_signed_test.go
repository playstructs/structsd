package app_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Updating an offering consumes provider authority, not its source's authority
// or an unrelated grant on the owning guild. The signing key remains a ceiling.
func TestSignedProviderUpdateAuthority(t *testing.T) {
	actions := []struct {
		name           string
		value, invalid uint64
		invalidReason  string
		message        func(string, string, uint64) sdk.Msg
		apply          func(*types.Provider, uint64)
	}{
		{"access", uint64(types.ProviderAccessPolicy_guildMarket), 500, "invalid_policy",
			func(creator, id string, value uint64) sdk.Msg {
				return &types.MsgProviderUpdateAccessPolicy{Creator: creator, ProviderId: id, AccessPolicy: types.ProviderAccessPolicy(value)}
			},
			func(p *types.Provider, value uint64) { p.AccessPolicy = types.ProviderAccessPolicy(value) }},
		{"capacity_minimum", 200, 1001, "exceeds_maximum",
			func(creator, id string, value uint64) sdk.Msg {
				return &types.MsgProviderUpdateCapacityMinimum{Creator: creator, ProviderId: id, NewMinimumCapacity: value}
			},
			func(p *types.Provider, value uint64) { p.CapacityMinimum = value }},
		{"capacity_maximum", 2000, 99, "cannot be lower than minimum",
			func(creator, id string, value uint64) sdk.Msg {
				return &types.MsgProviderUpdateCapacityMaximum{Creator: creator, ProviderId: id, NewMaximumCapacity: value}
			},
			func(p *types.Provider, value uint64) { p.CapacityMaximum = value }},
		{"duration_minimum", 20, 101, "exceeds_maximum",
			func(creator, id string, value uint64) sdk.Msg {
				return &types.MsgProviderUpdateDurationMinimum{Creator: creator, ProviderId: id, NewMinimumDuration: value}
			},
			func(p *types.Provider, value uint64) { p.DurationMinimum = value }},
		{"duration_maximum", 200, 9, "cannot be lower than minimum",
			func(creator, id string, value uint64) sdk.Msg {
				return &types.MsgProviderUpdateDurationMaximum{Creator: creator, ProviderId: id, NewMaximumDuration: value}
			},
			func(p *types.Provider, value uint64) { p.DurationMaximum = value }},
	}
	for _, action := range actions {
		for _, guildOwned := range []bool{false, true} {
			for _, scenario := range []string{"owner", "delegate", "rank", "demoted", "departed", "member_only", "station_grant", "guild_grant", "neither", "key_ceiling", "owner_key_ceiling", "rank_key_ceiling", "missing", "wrong_kind", "invalid_value", "later_failure"} {
				t.Run(fmt.Sprintf("%s/guild=%t/%s", action.name, guildOwned, scenario), func(t *testing.T) {
					bApp, ctx := setupReactorAppGenesis(t, 1, true)
					k := bApp.StructsKeeper
					newPlayer := func() types.Player {
						return appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
					}
					actor, owner := newPlayer(), newPlayer()
					if scenario == "owner" || scenario == "owner_key_ceiling" {
						owner = actor
					}
					key := secp256k1.GenPrivKey()
					address := sdk.AccAddress(key.PubKey().Address())
					require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), actor.Index))
					guild := types.CreateEmptyGuild()
					guild.Id, guild.Index, guild.Owner, guild.Creator = "0-978", 978, owner.Id, owner.Creator
					k.SetGuild(ctx, guild)
					ownerID := owner.Id
					if guildOwned {
						ownerID = guild.Id
					}
					const station = "4-978"
					k.SetSubstation(ctx, types.Substation{Id: station, Owner: ownerID})
					cc := k.NewCurrentContext(ctx)
					provider := cc.NewProvider(types.Provider{Owner: ownerID, Creator: owner.Creator, SubstationId: station,
						Rate: sdk.NewInt64Coin("ualpha", 3), AccessPolicy: types.ProviderAccessPolicy_openMarket,
						CapacityMinimum: 100, CapacityMaximum: 1000, DurationMinimum: 10, DurationMaximum: 100,
						ProviderCancellationPenalty: math.LegacyMustNewDecFromStr("0.25"), ConsumerCancellationPenalty: math.LegacyMustNewDecFromStr("0.5")})
					cc.CommitAll()
					original := provider.GetProvider()
					fundRedelegationAccount(t, bApp, ctx, provider.GetCollateralPoolLocation(), sdk.NewCoins(sdk.NewInt64Coin("ualpha", 11)))
					fundRedelegationAccount(t, bApp, ctx, provider.GetEarningsPoolLocation(), sdk.NewCoins(sdk.NewInt64Coin("ualpha", 7)))
					bits, grant, wantError := types.PermUpdate, false, ""
					switch scenario {
					case "owner", "owner_key_ceiling":
						require.Empty(t, actor.GuildId, "guild ownership does not require membership")
					case "delegate", "key_ceiling", "missing", "wrong_kind", "invalid_value", "later_failure":
						grant = true
					case "rank", "demoted", "departed", "rank_key_ceiling":
						actor.GuildId, actor.GuildRank = guild.Id, 3
						if scenario == "demoted" {
							actor.GuildRank, wantError = 4, "permission"
						}
						if scenario == "departed" {
							actor.GuildId, wantError = "", "permission"
						}
						k.SetPlayer(ctx, actor)
						k.SetGuildRankPermissionStoreOnly(ctx, provider.ID(), guild.Id, types.PermUpdate, 3)
					case "member_only":
						actor.GuildId, actor.GuildRank = guild.Id, 1
						k.SetPlayer(ctx, actor)
						wantError = "permission"
					case "station_grant", "guild_grant":
						objectID := station
						if scenario == "guild_grant" {
							objectID = guild.Id
						}
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(objectID, actor.Id), types.PermUpdate)
						wantError = "permission"
					case "neither":
						wantError = "permission"
					}
					if grant {
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(provider.ID(), actor.Id), types.PermUpdate)
					}
					if strings.HasSuffix(scenario, "key_ceiling") {
						bits, wantError = types.PermAll&^types.PermUpdate, "address lacks required permission bits"
					}
					k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
					id, value := provider.ID(), action.value
					switch scenario {
					case "missing":
						id, wantError = "10-999999", "not found"
					case "wrong_kind":
						id, wantError = actor.Id, "not found"
					case "invalid_value":
						value, wantError = action.invalid, action.invalidReason
					case "later_failure":
						wantError = "message index: 1"
					}
					messages := []sdk.Msg{action.message(address.String(), id, value)}
					if scenario == "later_failure" {
						messages = append(messages, action.message(address.String(), "10-999999", value))
					}
					beforeGrid, beforePermissions := k.GetAllGridExport(ctx), k.GetAllPermissionExport(ctx)
					beforeRanks, beforeCount := k.GetAllGuildRankPermissionExport(ctx), k.GetProviderCount(ctx)
					beforeAssets := bankAndAuthStateRows(bApp, ctx)
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
					expected := original
					if wantError == "" {
						require.Zero(t, result.Code, result.Log)
						require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
						action.apply(&expected, value)
					} else {
						require.NotZero(t, result.Code)
						require.Contains(t, result.Log, wantError)
						require.NotContains(t, result.Log, "panic")
					}
					actual, found := k.GetProvider(current, original.Id)
					require.True(t, found)
					require.Equal(t, expected, actual)
					require.Equal(t, beforeCount, k.GetProviderCount(current))
					require.Equal(t, beforeGrid, k.GetAllGridExport(current))
					require.Equal(t, beforePermissions, k.GetAllPermissionExport(current))
					require.Equal(t, beforeRanks, k.GetAllGuildRankPermissionExport(current))
					events := 0
					for _, event := range result.Events {
						if event.Type == "structs.structs.EventProvider" {
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							require.Equal(t, &expected, parsed.(*types.EventProvider).Provider)
							events++
						} else if strings.HasPrefix(event.Type, "structs.structs.Event") {
							require.Equal(t, "structs.structs.EventAddressActivity", event.Type)
						}
						require.NotContains(t, []string{"coin_spent", "coin_received", "transfer"}, event.Type)
					}
					if wantError == "" {
						require.Equal(t, 1, events)
					} else {
						require.Zero(t, events)
					}
					// Ante consumes sequence after a handler-level refusal, but
					// no provider update can move coins or provision an account.
					afterAssets := bankAndAuthStateRows(bApp, current)
					signerKey, err := collections.EncodeKeyWithPrefix(authtypes.AddressStoreKeyPrefix.Bytes(), sdk.AccAddressKey, address)
					require.NoError(t, err)
					signerRow := authtypes.StoreKey + "/" + string(signerKey)
					delete(beforeAssets, signerRow)
					delete(afterAssets, signerRow)
					require.Equal(t, beforeAssets, afterAssets)
					wantSequence := signer.GetSequence() + 1
					if strings.HasSuffix(scenario, "key_ceiling") {
						wantSequence--
					}
					afterSigner := bApp.AccountKeeper.GetAccount(current, address)
					require.Equal(t, signer.GetAccountNumber(), afterSigner.GetAccountNumber())
					require.Equal(t, wantSequence, afterSigner.GetSequence())
				})
			}
		}
	}
}

func TestSignedProviderUpdateFollowsGuildOwnerWithinBatch(t *testing.T) {
	for _, scenario := range []string{"new_owner", "old_owner", "restricted_new_owner"} {
		t.Run(scenario, func(t *testing.T) {
			bApp, ctx := setupReactorAppGenesis(t, 1, true)
			k := bApp.StructsKeeper
			oldOwner := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
			newOwner := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
			require.Empty(t, newOwner.GuildId)
			oldKey, newKey := secp256k1.GenPrivKey(), secp256k1.GenPrivKey()
			oldAddress, newAddress := sdk.AccAddress(oldKey.PubKey().Address()), sdk.AccAddress(newKey.PubKey().Address())
			require.NoError(t, k.SetPlayerIndexForAddress(ctx, oldAddress.String(), oldOwner.Index))
			require.NoError(t, k.SetPlayerIndexForAddress(ctx, newAddress.String(), newOwner.Index))
			k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(oldAddress.String()), types.PermAdmin)
			k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(newAddress.String()), types.PermUpdate)
			guild := types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner, guild.Creator = "0-978", 978, oldOwner.Id, oldOwner.Creator
			k.SetGuild(ctx, guild)
			k.SetSubstation(ctx, types.Substation{Id: "4-978", Owner: guild.Id})
			cc := k.NewCurrentContext(ctx)
			provider := cc.NewProvider(types.Provider{Owner: guild.Id, Creator: oldOwner.Creator, SubstationId: "4-978",
				Rate: sdk.NewInt64Coin("ualpha", 3), AccessPolicy: types.ProviderAccessPolicy_openMarket,
				CapacityMinimum: 100, CapacityMaximum: 1000, DurationMinimum: 10, DurationMaximum: 100,
				ProviderCancellationPenalty: math.LegacyZeroDec(), ConsumerCancellationPenalty: math.LegacyZeroDec()})
			cc.CommitAll()
			original := provider.GetProvider()
			updater := newAddress.String()
			keys := []cryptotypes.PrivKey{oldKey, newKey}
			if scenario == "old_owner" {
				updater, keys = oldAddress.String(), keys[:1]
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(oldAddress.String()), types.PermAdmin|types.PermUpdate)
			}
			if scenario == "restricted_new_owner" {
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(newAddress.String()), types.PermAll&^types.PermUpdate)
			}
			messages := []sdk.Msg{
				&types.MsgGuildUpdateOwnerId{Creator: oldAddress.String(), GuildId: guild.Id, Owner: newOwner.Id},
				&types.MsgProviderUpdateAccessPolicy{Creator: updater, ProviderId: provider.ID(), AccessPolicy: types.ProviderAccessPolicy_closedMarket},
			}
			beforeGrid, beforePermissions := k.GetAllGridExport(ctx), k.GetAllPermissionExport(ctx)
			beforeRanks, beforeAssets := k.GetAllGuildRankPermissionExport(ctx), bankAndAuthStateRows(bApp, ctx)
			var numbers, sequences []uint64
			for _, key := range keys {
				signer := bApp.AccountKeeper.GetAccount(ctx, sdk.AccAddress(key.PubKey().Address()))
				numbers, sequences = append(numbers, signer.GetAccountNumber()), append(sequences, signer.GetSequence())
			}
			config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil,
				ante.DefaultFreeGasCap, jailGateChainID, numbers, sequences, keys...)
			require.NoError(t, err)
			wire, err := config.TxEncoder()(tx)
			require.NoError(t, err)
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
			require.NoError(t, err)
			require.Len(t, block.TxResults, 1)
			result := block.TxResults[0]
			current := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
			expected := original
			expectedGuild := guild
			if scenario == "new_owner" {
				require.Zero(t, result.Code, result.Log)
				expected.AccessPolicy = types.ProviderAccessPolicy_closedMarket
				expectedGuild.Owner = newOwner.Id
				require.Equal(t, types.PermGuildAll, k.GetPermissionsByBytes(current, keeperlib.GetObjectPermissionIDBytes(guild.Id, newOwner.Id)))
			} else {
				require.NotZero(t, result.Code)
				if scenario == "old_owner" {
					require.Contains(t, result.Log, "message index: 1")
				} else {
					require.Contains(t, result.Log, "address lacks required permission bits")
				}
				require.Contains(t, result.Log, "permission")
				require.NotContains(t, result.Log, "panic")
				require.Equal(t, beforePermissions, k.GetAllPermissionExport(current))
				for _, event := range result.Events {
					require.False(t, strings.HasPrefix(event.Type, "structs.structs.Event"), event.Type)
				}
			}
			actual, found := k.GetProvider(current, original.Id)
			require.True(t, found)
			require.Equal(t, expected, actual)
			actualGuild, found := k.GetGuild(current, guild.Id)
			require.True(t, found)
			require.Equal(t, expectedGuild, actualGuild)
			expectedPermissions, actualPermissions := map[string]uint64{}, map[string]uint64{}
			for _, row := range beforePermissions {
				expectedPermissions[row.PermissionId] = row.Value
			}
			if scenario == "new_owner" {
				expectedPermissions[string(keeperlib.GetObjectPermissionIDBytes(guild.Id, newOwner.Id))] = uint64(types.PermGuildAll)
			}
			for _, row := range k.GetAllPermissionExport(current) {
				actualPermissions[row.PermissionId] = row.Value
			}
			require.Equal(t, expectedPermissions, actualPermissions)
			require.Equal(t, beforeGrid, k.GetAllGridExport(current))
			require.Equal(t, beforeRanks, k.GetAllGuildRankPermissionExport(current))
			for _, owner := range []types.Player{oldOwner, newOwner} {
				require.Equal(t, types.Permissionless, k.GetPermissionsByBytes(current, keeperlib.GetObjectPermissionIDBytes(provider.ID(), owner.Id)), "ownership must not be copied into child grants")
			}
			events := 0
			for _, event := range result.Events {
				require.NotContains(t, []string{"coin_spent", "coin_received", "transfer"}, event.Type)
				if event.Type == "structs.structs.EventProvider" {
					parsed, err := sdk.ParseTypedEvent(event)
					require.NoError(t, err)
					require.Equal(t, &expected, parsed.(*types.EventProvider).Provider)
					events++
				}
			}
			if scenario == "new_owner" {
				require.Equal(t, 1, events)
			} else {
				require.Zero(t, events)
			}
			afterAssets := bankAndAuthStateRows(bApp, current)
			for i, key := range keys {
				address := sdk.AccAddress(key.PubKey().Address())
				signerKey, err := collections.EncodeKeyWithPrefix(authtypes.AddressStoreKeyPrefix.Bytes(), sdk.AccAddressKey, address)
				require.NoError(t, err)
				delete(beforeAssets, authtypes.StoreKey+"/"+string(signerKey))
				delete(afterAssets, authtypes.StoreKey+"/"+string(signerKey))
				wantSequence := sequences[i] + 1
				if scenario == "restricted_new_owner" {
					wantSequence--
				}
				afterSigner := bApp.AccountKeeper.GetAccount(current, address)
				require.Equal(t, numbers[i], afterSigner.GetAccountNumber())
				require.Equal(t, wantSequence, afterSigner.GetSequence())
			}
			require.Equal(t, beforeAssets, afterAssets)
		})
	}
}
