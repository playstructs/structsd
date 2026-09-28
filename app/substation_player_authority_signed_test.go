package app_test

import (
	"fmt"
	"math/rand"
	"testing"

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

// Joining needs both parties' authority; either party may disconnect. Guild
// ownership and rank affect the station gate, never the target player's gate.
func TestSignedSubstationPlayerAuthority(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		for _, action := range []string{"connect", "migrate", "disconnect"} {
			for _, scenario := range []string{"both", "station_only", "player_only", "neither", "key_ceiling", "owner", "rank", "rank_demoted", "rank_departed", "late_missing_player", "late_player_denied", "later_message_failure", "duplicate_player"} {
				if (scenario == "late_missing_player" || scenario == "late_player_denied" || scenario == "duplicate_player") && action != "migrate" {
					continue
				}
				t.Run(fmt.Sprintf("guild=%t/%s/%s", guildOwned, action, scenario), func(t *testing.T) {
					bApp, ctx := setupJailGateApp(t)
					k := bApp.StructsKeeper
					key := secp256k1.GenPrivKey()
					address := sdk.AccAddress(key.PubKey().Address())
					actor := appendAppPlayer(t, k, ctx, address)
					newPlayer := func() types.Player {
						return appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
					}
					owner, first, second, resident := newPlayer(), newPlayer(), newPlayer(), newPlayer()
					guild := types.CreateEmptyGuild()
					guild.Id, guild.Index, guild.Owner, guild.Creator = "0-954", 954, owner.Id, owner.Creator
					stationOwner := owner.Id
					if scenario == "owner" {
						guild.Owner, stationOwner = actor.Id, actor.Id
						// Ownership works even when the actor is not a guild member.
					}
					if guildOwned {
						stationOwner = guild.Id
					}
					k.SetGuild(ctx, guild)
					const oldStation, newStation = "4-954", "4-955"
					for _, id := range []string{oldStation, newStation} {
						k.SetSubstation(ctx, types.Substation{Id: id, Owner: stationOwner})
					}
					k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, oldStation), 120)
					k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, newStation), 300)
					cc := k.NewCurrentContext(ctx)
					cc.GetPlayer(first.Id).MigrateSubstation(oldStation)
					cc.GetPlayer(second.Id).MigrateSubstation(oldStation)
					cc.GetPlayer(resident.Id).MigrateSubstation(newStation)
					cc.CommitAll()
					station := newStation
					if action == "disconnect" {
						station = oldStation
					}
					stationGrant, playerGrant := true, true
					switch scenario {
					case "station_only":
						playerGrant = false
					case "player_only":
						stationGrant = false
					case "neither":
						stationGrant, playerGrant = false, false
					case "owner", "rank", "rank_demoted", "rank_departed":
						stationGrant = false
						if action == "disconnect" {
							playerGrant = false // isolate the station's owner/rank fallback
						}
					}
					if scenario == "rank" || scenario == "rank_demoted" || scenario == "rank_departed" {
						actor.GuildId, actor.GuildRank = guild.Id, 3
						if scenario == "rank_demoted" {
							actor.GuildRank = 4
						}
						if scenario == "rank_departed" {
							actor.GuildId = ""
						}
						k.SetPlayer(ctx, actor)
						k.SetGuildRankPermissionStoreOnly(ctx, station, guild.Id, types.PermSubstationConnection, 3)
					}
					if stationGrant {
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(station, actor.Id), types.PermSubstationConnection)
					}
					if playerGrant {
						for _, id := range []string{first.Id, second.Id} {
							if scenario == "late_player_denied" && id == second.Id {
								continue
							}
							k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(id, actor.Id), types.PermSubstationConnection)
						}
					}
					bits := types.PermSubstationConnection
					if scenario == "key_ceiling" {
						bits = types.PermPlay
					}
					k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
					players := []string{first.Id, second.Id}
					if scenario == "late_missing_player" {
						players = append(players, "1-999999")
					}
					if scenario == "duplicate_player" {
						players = []string{first.Id, first.Id, second.Id}
					}
					var msg sdk.Msg
					switch action {
					case "connect":
						msg = &types.MsgSubstationPlayerConnect{Creator: address.String(), SubstationId: newStation, PlayerId: first.Id}
					case "migrate":
						msg = &types.MsgSubstationPlayerMigrate{Creator: address.String(), SubstationId: newStation, PlayerId: players}
					case "disconnect":
						msg = &types.MsgSubstationPlayerDisconnect{Creator: address.String(), PlayerId: first.Id}
					}
					messages := []sdk.Msg{msg}
					if scenario == "later_message_failure" {
						messages = append(messages, &types.MsgSubstationPlayerDisconnect{Creator: address.String(), PlayerId: "1-999999"})
					}
					stationAllowed := stationGrant || scenario == "owner" || scenario == "rank"
					allowed := stationAllowed && playerGrant
					if action == "disconnect" {
						allowed = stationAllowed || playerGrant
					}
					wantError := ""
					if !allowed || scenario == "key_ceiling" || scenario == "late_player_denied" {
						wantError = "permission"
					}
					if scenario == "late_missing_player" {
						wantError = "not found"
					}
					if scenario == "later_message_failure" {
						wantError = "message index: 1"
					}
					beforeGrid, beforePermissions := k.GetAllGridExport(ctx), k.GetAllPermissionExport(ctx)
					beforeAssets := bankAndAuthStateRows(bApp, ctx)
					beforePlayers := map[string]types.Player{}
					for _, player := range []types.Player{actor, owner, first, second, resident} {
						stored, found := k.GetPlayer(ctx, player.Id)
						require.True(t, found)
						beforePlayers[player.Id] = stored
					}
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
					expected := map[string]string{first.Id: oldStation, second.Id: oldStation, resident.Id: newStation}
					if wantError != "" {
						require.NotZero(t, result.Code)
						require.Contains(t, result.Log, wantError)
						require.NotContains(t, result.Log, "panic")
						require.Equal(t, beforeGrid, k.GetAllGridExport(current))
					} else {
						require.Zero(t, result.Code, result.Log)
						require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
						expected[first.Id] = newStation
						if action == "disconnect" {
							expected[first.Id] = ""
						}
						if action == "migrate" {
							expected[second.Id] = newStation
						}
					}
					require.Equal(t, beforePermissions, k.GetAllPermissionExport(current))
					for _, id := range []string{oldStation, newStation} {
						var members []string
						for playerID, stationID := range expected {
							if stationID == id {
								members = append(members, playerID)
							}
						}
						require.ElementsMatch(t, members, k.GetAllPlayerIdBySubstationIndex(current, id))
						stationCache := k.NewCurrentContext(current).GetSubstation(id)
						require.Equal(t, uint64(len(members)), stationCache.GetConnectionCount())
						capacity := uint64(120)
						if id == newStation {
							capacity = 300
						}
						require.Equal(t, (capacity*9/10)/uint64(max(1, len(members))), k.GetGridAttribute(current, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, id)))
					}
					playerEvents := map[string]int{}
					for _, event := range result.Events {
						require.NotContains(t, []string{"coin_spent", "coin_received", "transfer", "structs.structs.EventPermission", "structs.structs.EventGuildRankPermission"}, event.Type)
						if event.Type != "structs.structs.EventPlayer" {
							continue
						}
						require.Empty(t, wantError, "rolled-back connections must not reach indexers")
						parsed, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						player := parsed.(*types.EventPlayer).Player
						require.Contains(t, expected, player.Id)
						stored, found := k.GetPlayer(current, player.Id)
						require.True(t, found)
						require.Equal(t, stored, *player)
						playerEvents[player.Id]++
					}
					for id, station := range expected {
						player, found := k.GetPlayer(current, id)
						require.True(t, found)
						require.Equal(t, station, player.SubstationId)
						expectedPlayer := beforePlayers[id]
						expectedPlayer.SubstationId = station
						require.Equal(t, expectedPlayer, player, "connection changes no other player fields")
						count := 0
						if wantError == "" && (id == first.Id || (action == "migrate" && id == second.Id)) {
							count = 1
						}
						require.Equal(t, count, playerEvents[id], "one final replacement per changed player, including duplicate input ids")
					}
					for _, id := range []string{actor.Id, owner.Id} {
						stored, found := k.GetPlayer(current, id)
						require.True(t, found)
						require.Equal(t, beforePlayers[id], stored)
					}
					// Signed delivery consumes sequence even when a later handler
					// refuses. Exclude only that expected auth-row change; all
					// bank rows and every other account must remain identical.
					afterAssets := bankAndAuthStateRows(bApp, current)
					signerKey, err := collections.EncodeKeyWithPrefix(authtypes.AddressStoreKeyPrefix.Bytes(), sdk.AccAddressKey, address)
					require.NoError(t, err)
					signerRow := authtypes.StoreKey + "/" + string(signerKey)
					delete(beforeAssets, signerRow)
					delete(afterAssets, signerRow)
					require.Equal(t, beforeAssets, afterAssets)
					wantSequence := signer.GetSequence() + 1
					if scenario == "key_ceiling" {
						wantSequence = signer.GetSequence()
					}
					afterSigner := bApp.AccountKeeper.GetAccount(current, address)
					require.Equal(t, signer.GetAccountNumber(), afterSigner.GetAccountNumber())
					require.Equal(t, wantSequence, afterSigner.GetSequence())
				})
			}
		}
	}
}
