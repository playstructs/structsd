package app_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/collections"
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

// UGC has two independent authority paths: Update on the object, or
// GuildUGCUpdate on its owning guild (the owner's guild for player assets).
// Both must follow current standing and the exact signing key's own bits.
func TestSignedSubstationUGCAuthority(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		for _, field := range []string{"name", "pfp"} {
			for _, scenario := range []string{"owner", "direct", "direct_after_departure", "rank", "demoted", "departed", "guild_ugc", "guild_rank", "guild_demoted", "guild_departed", "guild_owner", "wrong_guild", "update_key_ceiling", "ugc_key_ceiling", "neither", "missing", "invalid", "later_failure", "former_owner"} {
				if scenario == "former_owner" && !guildOwned {
					continue
				}
				t.Run(fmt.Sprintf("guild=%t/%s/%s", guildOwned, field, scenario), func(t *testing.T) {
					bApp, ctx := setupReactorAppGenesis(t, 1, true)
					k := bApp.StructsKeeper
					newPlayer := func() types.Player {
						return appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
					}
					actor, owner, other := newPlayer(), newPlayer(), newPlayer()
					if scenario == "owner" || scenario == "former_owner" {
						actor = owner
					}
					guild, unrelated := types.CreateEmptyGuild(), types.CreateEmptyGuild()
					guild.Id, guild.Index, guild.Owner = "0-990", 990, owner.Id
					unrelated.Id, unrelated.Index, unrelated.Owner = "0-991", 991, other.Id
					if !guildOwned {
						guild.Owner = other.Id
					}
					if scenario == "guild_owner" {
						guild.Owner = actor.Id
					}
					if scenario == "former_owner" {
						guild.Owner = other.Id
					}
					k.SetGuild(ctx, guild)
					k.SetGuild(ctx, unrelated)
					owner.GuildId = guild.Id
					if guildOwned {
						owner.GuildId = unrelated.Id
					}
					k.SetPlayer(ctx, owner)
					if actor.Id == owner.Id {
						actor = owner
					}
					station := types.Substation{Id: "4-990", Owner: owner.Id, Name: "Old Station", Pfp: "https://example.com/old.png"}
					if guildOwned {
						station.Owner = guild.Id
					}
					k.SetSubstation(ctx, station)
					key := secp256k1.GenPrivKey()
					address := sdk.AccAddress(key.PubKey().Address())
					require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), actor.Index))
					bits, success := types.PermUpdate, true
					grant := func(id string, permission types.Permission) {
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(id, actor.Id), permission)
					}
					switch scenario {
					case "direct", "direct_after_departure", "invalid", "later_failure", "missing", "update_key_ceiling":
						grant(station.Id, types.PermUpdate)
                        if scenario == "direct_after_departure" {
                            grant(station.Id, types.PermUpdate|types.PermAdmin)
                            bits |= types.PermAdmin
                            actor.GuildId, actor.GuildRank = "", 0
							k.SetGuildRankPermissionStoreOnly(ctx, station.Id, guild.Id, types.PermUpdate, 3)
						}
						if scenario == "update_key_ceiling" {
							bits, success = types.PermGuildUGCUpdate, false
						}
					case "rank", "demoted", "departed":
						actor.GuildId, actor.GuildRank = guild.Id, 3
						k.SetGuildRankPermissionStoreOnly(ctx, station.Id, guild.Id, types.PermUpdate, 3)
						if scenario == "demoted" {
							actor.GuildRank, success = 4, false
						}
						if scenario == "departed" {
							actor.GuildId, success = "", false
						}
					case "guild_ugc", "ugc_key_ceiling":
						bits = types.PermGuildUGCUpdate
						grant(guild.Id, types.PermGuildUGCUpdate)
						if scenario == "ugc_key_ceiling" {
							bits, success = types.PermUpdate, false
						}
					case "guild_rank", "guild_demoted", "guild_departed":
						bits = types.PermGuildUGCUpdate
						actor.GuildId, actor.GuildRank = guild.Id, 3
						k.SetGuildRankPermissionStoreOnly(ctx, guild.Id, guild.Id, types.PermGuildUGCUpdate, 3)
						if scenario == "guild_demoted" {
							actor.GuildRank, success = 4, false
						}
						if scenario == "guild_departed" {
							actor.GuildId, success = "", false
						}
					case "guild_owner":
						bits = types.PermGuildUGCUpdate
					case "wrong_guild":
						bits, success = types.PermGuildUGCUpdate, false
						grant(unrelated.Id, types.PermGuildUGCUpdate)
					case "neither", "former_owner":
						success = false
					}
					k.SetPlayer(ctx, actor)
					k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
					id, value := station.Id, "New Station"
					if field == "pfp" {
						value = "https://example.com/new.png"
					}
					if scenario == "invalid" {
						value, success = strings.Repeat("x", types.MaxPfpLength+1), false
					}
					if scenario == "missing" {
						id, success = "4-999999", false
					}
					message := func(id string) sdk.Msg {
						if field == "name" {
							return &types.MsgSubstationUpdateName{Creator: address.String(), SubstationId: id, Name: value}
						}
						return &types.MsgSubstationUpdatePfp{Creator: address.String(), SubstationId: id, Pfp: value}
					}
					messages := []sdk.Msg{message(id)}
					if scenario == "later_failure" {
						messages, success = append(messages, message("4-999999")), false
					}
					beforePermissions, beforeRanks := k.GetAllPermissionExport(ctx), k.GetAllGuildRankPermissionExport(ctx)
					beforeGrid, beforeGuilds, beforeAssets := k.GetAllGridExport(ctx), k.GetAllGuild(ctx), bankAndAuthStateRows(bApp, ctx)
					signer := bApp.AccountKeeper.GetAccount(ctx, address)
					config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
					deliver := func(height int64, messages ...sdk.Msg) *abci.ExecTxResult {
						tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(height)), config, messages, nil, ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence() + uint64(height-1)}, key)
						require.NoError(t, err)
						wire, err := config.TxEncoder()(tx)
						require.NoError(t, err)
						block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: ctx.BlockTime().Add(time.Duration(height-1) * time.Second), Txs: [][]byte{wire}})
						require.NoError(t, err)
						require.Len(t, block.TxResults, 1)
						return block.TxResults[0]
					}
					result := deliver(1, messages...)
					if success {
						require.Zero(t, result.Code, result.Log)
					} else {
						require.NotZero(t, result.Code)
						require.NotContains(t, result.Log, "panic")
						if scenario == "later_failure" {
							require.Contains(t, result.Log, "message index: 1")
						}
					}
					current := bApp.NewContext(false)
					expected := station
					if success && field == "name" {
						expected.Name = value
					}
					if success && field == "pfp" {
						expected.Pfp = value
					}
					actual, found := k.GetSubstation(current, station.Id)
					require.True(t, found)
					require.Equal(t, expected, actual)
					require.Equal(t, beforePermissions, k.GetAllPermissionExport(current))
					require.Equal(t, beforeRanks, k.GetAllGuildRankPermissionExport(current))
					require.Equal(t, beforeGrid, k.GetAllGridExport(current))
					require.Equal(t, beforeGuilds, k.GetAllGuild(current))
					signerKey, err := collections.EncodeKeyWithPrefix(authtypes.AddressStoreKeyPrefix.Bytes(), sdk.AccAddressKey, address)
					require.NoError(t, err)
					delete(beforeAssets, authtypes.StoreKey+"/"+string(signerKey))
					afterAssets := bankAndAuthStateRows(bApp, current)
					delete(afterAssets, authtypes.StoreKey+"/"+string(signerKey))
					require.Equal(t, beforeAssets, afterAssets)
					require.Equal(t, signer.GetSequence()+1, bApp.AccountKeeper.GetAccount(current, address).GetSequence())
					require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
					updates, moderation := 0, 0
					for _, event := range result.Events {
						require.NotContains(t, []string{"transfer", "coin_spent", "coin_received", "structs.structs.EventGrid", "structs.structs.EventPermission"}, event.Type)
						if event.Type == "structs.structs.EventSubstation" {
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							require.Equal(t, expected, *parsed.(*types.EventSubstation).Substation)
							updates++
						}
						if event.Type == types.EventTypeUGCModerated {
							attributes := map[string]string{}
							for _, attribute := range event.Attributes {
								require.NotContains(t, attributes, attribute.Key)
								attributes[attribute.Key] = attribute.Value
							}
							old := station.Name
							if field == "pfp" {
								old = station.Pfp
							}
							for name, wanted := range map[string]string{types.AttributeKeyUGCActorPlayerId: actor.Id, types.AttributeKeyUGCActorAddress: address.String(), types.AttributeKeyUGCTargetObjectId: station.Id, types.AttributeKeyUGCTargetOwnerId: station.Owner, types.AttributeKeyUGCField: field, types.AttributeKeyUGCOldValue: old, types.AttributeKeyUGCNewValue: value} {
								require.Equal(t, wanted, attributes[name])
							}
							moderation++
						}
					}
					if success {
						require.Equal(t, 1, updates)
						if station.Owner == actor.Id {
							require.Zero(t, moderation)
						} else {
							require.Equal(t, 1, moderation)
						}
					} else {
						require.Zero(t, updates)
						require.Zero(t, moderation)
					}
					if scenario == "direct_after_departure" {
						_, err = bApp.Commit()
						require.NoError(t, err)
						revoke := &types.MsgPermissionRevokeOnObject{Creator: address.String(), ObjectId: station.Id, PlayerId: actor.Id, Permissions: uint64(types.PermUpdate)}
						failed := deliver(2, revoke, message(station.Id))
						require.NotZero(t, failed.Code)
						require.Contains(t, failed.Log, "message index: 1")
						require.Equal(t, beforePermissions, k.GetAllPermissionExport(bApp.NewContext(false)), "failed batch restores direct grant")
						_, err = bApp.Commit()
						require.NoError(t, err)
						require.Zero(t, deliver(3, revoke).Code)
						_, err = bApp.Commit()
						require.NoError(t, err)
						require.NotZero(t, deliver(4, message(station.Id)).Code, "explicit revocation removes independent authority")
						actual, found = k.GetSubstation(bApp.NewContext(false), station.Id)
						require.True(t, found)
						require.Equal(t, expected, actual)
					}
				})
			}
		}
	}
}
