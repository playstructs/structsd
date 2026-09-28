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

// Deleting a station controls its connections and allocations, but migrating
// its residents also requires authority on the receiving station. Exercise the
// actual signed transaction, including rollback after the teardown has run.
func TestSignedSubstationDeletionAuthorityAndCleanup(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		for _, migration := range []string{"disconnect", "self", "other"} {
			for _, scenario := range []string{"direct", "delete_only", "destination_only", "neither", "owner", "rank", "demoted", "departed", "delete_key_ceiling", "connection_key_ceiling", "missing_destination", "missing_station", "later_failure", "repeat_delete"} {
				if migration == "disconnect" && (scenario == "delete_only" || scenario == "destination_only" || scenario == "connection_key_ceiling" || scenario == "missing_destination") {
					continue
				}
				if migration == "self" && scenario == "missing_destination" {
					continue
				}
				t.Run(fmt.Sprintf("guild=%t/%s/%s", guildOwned, migration, scenario), func(t *testing.T) {
					bApp, ctx := setupReactorAppGenesis(t, 1, true)
					k := bApp.StructsKeeper
					newPlayer := func() types.Player {
						return appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
					}
					actor, owner, destinationOwner, root, automatedRoot := newPlayer(), newPlayer(), newPlayer(), newPlayer(), newPlayer()
					first, second, resident := newPlayer(), newPlayer(), newPlayer()
					if scenario == "owner" {
						owner, destinationOwner = actor, actor
					}
					guild := types.CreateEmptyGuild()
					guild.Id, guild.Index, guild.Owner, guild.Creator = "0-987", 987, owner.Id, owner.Creator
					k.SetGuild(ctx, guild)
					stationOwner := owner.Id
					if guildOwned {
						stationOwner = guild.Id
					}
					const oldStation, targetStation, tailStation = "4-987", "4-988", "4-989"
					k.SetSubstation(ctx, types.Substation{Id: oldStation, Owner: stationOwner})
					k.SetSubstation(ctx, types.Substation{Id: targetStation, Owner: destinationOwner.Id})
					k.SetSubstation(ctx, types.Substation{Id: tailStation, Owner: root.Id})
					gridID := keeperlib.GetGridAttributeIDByObjectId
					k.SetGridAttribute(ctx, gridID(types.GridAttributeType_capacity, root.Id), 1000)
					k.SetGridAttribute(ctx, gridID(types.GridAttributeType_capacity, automatedRoot.Id), 40)
					k.SetGridAttribute(ctx, gridID(types.GridAttributeType_capacity, targetStation), 300)
					cc := k.NewCurrentContext(ctx)
					incoming, err := cc.NewAllocation(types.AllocationType_dynamic, root.Id, oldStation, root.Creator, root.Id, 120)
					require.NoError(t, err)
					automated, err := cc.NewAllocation(types.AllocationType_automated, automatedRoot.Id, oldStation, automatedRoot.Creator, automatedRoot.Id, 0)
					require.NoError(t, err)
					outgoing, err := cc.NewAllocation(types.AllocationType_dynamic, oldStation, tailStation, owner.Creator, stationOwner, 40)
					require.NoError(t, err)
					cc.GetPlayer(first.Id).MigrateSubstation(oldStation)
					cc.GetPlayer(second.Id).MigrateSubstation(oldStation)
					cc.GetPlayer(resident.Id).MigrateSubstation(targetStation)
					cc.CommitAll()
					deleted := map[string]bool{oldStation: true, incoming.ID(): true, automated.ID(): true, outgoing.ID(): true}
					// Unrelated grants on every deleted object must be cleared;
					// surviving object grants must remain byte-for-byte intact.
					for _, id := range []string{oldStation, targetStation, incoming.ID(), automated.ID(), outgoing.ID()} {
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(id, resident.Id), types.PermUpdate)
						k.SetGuildRankPermissionStoreOnly(ctx, id, guild.Id, types.PermUpdate, 2)
					}
					key := secp256k1.GenPrivKey()
					address := sdk.AccAddress(key.PubKey().Address())
					require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), actor.Index))
					destination := ""
					if migration == "self" {
						destination = oldStation
					}
					if migration == "other" {
						destination = targetStation
					}
					grantDelete, grantConnection, wantError := true, true, ""
					switch scenario {
					case "delete_only":
						grantConnection, wantError = false, "permission"
					case "destination_only":
						grantDelete, wantError = false, "permission"
					case "neither":
						grantDelete, grantConnection, wantError = false, false, "permission"
					case "owner":
						grantDelete, grantConnection = false, false
					case "rank", "demoted", "departed":
						grantDelete, grantConnection = false, false
						actor.GuildId, actor.GuildRank = guild.Id, 3
						if scenario == "demoted" {
							actor.GuildRank, wantError = 4, "permission"
						}
						if scenario == "departed" {
							actor.GuildId, wantError = "", "permission"
						}
						k.SetPlayer(ctx, actor)
						k.SetGuildRankPermissionStoreOnly(ctx, oldStation, guild.Id, types.PermDelete, 3)
						if destination != "" {
							k.SetGuildRankPermissionStoreOnly(ctx, destination, guild.Id, types.PermSubstationConnection, 3)
						}
					}
					if grantDelete {
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(oldStation, actor.Id), types.PermDelete)
					}
					if grantConnection && destination != "" {
						cc = k.NewCurrentContext(ctx)
						cc.PermissionAdd(keeperlib.GetObjectPermissionIDBytes(destination, actor.Id), types.PermSubstationConnection)
						cc.CommitAll()
					}
					bits := types.PermDelete | types.PermSubstationConnection
					if migration == "disconnect" {
						bits = types.PermDelete
					}
					if scenario == "delete_key_ceiling" {
						bits, wantError = bits&^types.PermDelete, "address lacks required permission bits"
					}
					if scenario == "connection_key_ceiling" {
						bits, wantError = bits&^types.PermSubstationConnection, "permission"
					}
					k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
					id := oldStation
					if scenario == "missing_station" {
						id, wantError = "4-999999", "not found"
					}
					if scenario == "missing_destination" {
						destination, wantError = "4-999999", "not found"
					}
					messages := []sdk.Msg{&types.MsgSubstationDelete{Creator: address.String(), SubstationId: id, MigrationSubstationId: destination}}
					if scenario == "later_failure" || scenario == "repeat_delete" {
						nextID := "4-999999"
						if scenario == "repeat_delete" {
							nextID = oldStation
						}
						messages = append(messages, &types.MsgSubstationDelete{Creator: address.String(), SubstationId: nextID})
						wantError = "message index: 1"
					}
					beforeStations, beforeAllocations, beforePlayers := k.GetAllSubstation(ctx), k.GetAllAllocation(ctx), k.GetAllPlayer(ctx)
					beforePermissions, beforeRanks, beforeGrid := k.GetAllPermissionExport(ctx), k.GetAllGuildRankPermissionExport(ctx), k.GetAllGridExport(ctx)
					beforeAssets, count := bankAndAuthStateRows(bApp, ctx), k.GetSubstationCount(ctx)
					signer := bApp.AccountKeeper.GetAccount(ctx, address)
					config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
					deliver := func(height int64, sequence uint64) *abci.ExecTxResult {
						tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(height)), config, messages, nil, ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{sequence}, key)
						require.NoError(t, err)
						wire, err := config.TxEncoder()(tx)
						require.NoError(t, err)
						block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: ctx.BlockTime().Add(time.Duration(height-1) * time.Second), Txs: [][]byte{wire}})
						require.NoError(t, err)
						require.Len(t, block.TxResults, 1)
						return block.TxResults[0]
					}
					result := deliver(1, signer.GetSequence())
					current := bApp.NewContext(false)
					success := wantError == ""
					if success {
						require.Zero(t, result.Code, result.Log)
					} else {
						require.NotZero(t, result.Code)
						require.Contains(t, result.Log, wantError)
						require.NotContains(t, result.Log, "panic")
					}
					require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
					expectedGrid, gridEvents := map[string]uint64{}, map[string]uint64{}
					for _, row := range beforeGrid {
						expectedGrid[row.AttributeId] = row.Value
					}
					expectedPlayers := append([]types.Player{}, beforePlayers...)
					expectedStations := []types.Substation{}
					for _, row := range beforeStations {
						if !success || row.Id != oldStation {
							expectedStations = append(expectedStations, row)
						}
					}
					expectedAllocations := []types.Allocation{}
					for _, row := range beforeAllocations {
						if !success || !deleted[row.Id] {
							expectedAllocations = append(expectedAllocations, row)
						}
					}
					expectedPermissions, permissionEvents := map[string]uint64{}, map[string]uint64{}
					for _, row := range beforePermissions {
						if success && deleted[strings.SplitN(row.PermissionId, "@", 2)[0]] {
							permissionEvents[row.PermissionId] = 0
						} else {
							expectedPermissions[row.PermissionId] = row.Value
						}
					}
					expectedRanks, rankEvents := []*types.GuildRankPermissionRecord{}, []*types.GuildRankPermissionRecord{}
					for _, row := range beforeRanks {
						if success && deleted[row.ObjectId] {
							copy := *row
							copy.Rank = 0
							rankEvents = append(rankEvents, &copy)
						} else {
							expectedRanks = append(expectedRanks, row)
						}
					}
					playerEvents := map[string]types.Player{}
					if success {
						for i := range expectedPlayers {
							if expectedPlayers[i].Id == first.Id || expectedPlayers[i].Id == second.Id {
								expectedPlayers[i].SubstationId = ""
								if migration == "other" {
									expectedPlayers[i].SubstationId = targetStation
								}
								playerEvents[expectedPlayers[i].Id] = expectedPlayers[i]
							}
						}
						for _, kind := range []types.GridAttributeType{types.GridAttributeType_capacity, types.GridAttributeType_load, types.GridAttributeType_connectionCapacity, types.GridAttributeType_connectionCount} {
							attribute := gridID(kind, oldStation)
							delete(expectedGrid, attribute)
							gridEvents[attribute] = 0
						}
						for _, allocation := range []string{incoming.ID(), automated.ID(), outgoing.ID()} {
							attribute := gridID(types.GridAttributeType_power, allocation)
							delete(expectedGrid, attribute)
							gridEvents[attribute] = 0
						}
						for _, rootID := range []string{root.Id, automatedRoot.Id} {
							gridEvents[gridID(types.GridAttributeType_load, rootID)] = 0
						}
						gridEvents[gridID(types.GridAttributeType_capacity, tailStation)] = 0
						gridEvents[gridID(types.GridAttributeType_connectionCapacity, tailStation)] = 0
						if migration == "other" {
							gridEvents[gridID(types.GridAttributeType_connectionCount, targetStation)] = 3
							gridEvents[gridID(types.GridAttributeType_connectionCapacity, targetStation)] = 90
						}
						for attribute, value := range gridEvents {
							if _, exists := expectedGrid[attribute]; exists {
								expectedGrid[attribute] = value
							}
						}
					}
					checkState := func(c sdk.Context) {
						require.Equal(t, count, k.GetSubstationCount(c))
						require.ElementsMatch(t, expectedStations, k.GetAllSubstation(c))
						require.ElementsMatch(t, expectedAllocations, k.GetAllAllocation(c))
						require.ElementsMatch(t, expectedPlayers, k.GetAllPlayer(c))
						require.ElementsMatch(t, expectedRanks, k.GetAllGuildRankPermissionExport(c))
						actualPermissions, actualGrid := map[string]uint64{}, map[string]uint64{}
						for _, row := range k.GetAllPermissionExport(c) {
							actualPermissions[row.PermissionId] = row.Value
						}
						for _, row := range k.GetAllGridExport(c) {
							actualGrid[row.AttributeId] = row.Value
						}
						require.Equal(t, expectedPermissions, actualPermissions)
						require.Equal(t, expectedGrid, actualGrid)
						for _, station := range []string{oldStation, targetStation, tailStation} {
							members, arrivals := []string{}, []string{}
							for _, row := range expectedPlayers {
								if row.SubstationId == station {
									members = append(members, row.Id)
								}
							}
							for _, row := range expectedAllocations {
								if row.DestinationId == station {
									arrivals = append(arrivals, row.Id)
								}
							}
							require.ElementsMatch(t, members, k.GetAllPlayerIdBySubstationIndex(c, station))
							require.ElementsMatch(t, arrivals, k.GetAllAllocationIdByDestinationIndex(c, station))
						}
						for _, source := range []string{root.Id, automatedRoot.Id, oldStation} {
							ids := []string{}
							for _, row := range expectedAllocations {
								if row.SourceObjectId == source {
									ids = append(ids, row.Id)
								}
							}
							require.ElementsMatch(t, ids, k.GetAllAllocationIdBySourceIndex(c, source))
						}
						hook, found := k.GetAutoResizeAllocationBySource(c, automatedRoot.Id)
						require.Equal(t, !success, found)
						if found {
							require.Equal(t, automated.ID(), hook)
						}
					}
					checkState(current)
					actualGridEvents, actualPermissionEvents := map[string]uint64{}, map[string]uint64{}
					actualRankEvents := []*types.GuildRankPermissionRecord{}
					actualPlayerEvents, actualDeletes := map[string]types.Player{}, map[string]bool{}
					for _, event := range result.Events {
						require.NotContains(t, []string{"transfer", "coin_spent", "coin_received", "coinbase", "burn", "structs.structs.EventVaultAddress", "structs.structs.EventSubstation", "structs.structs.EventAllocation"}, event.Type)
						if !strings.HasPrefix(event.Type, "structs.structs.Event") {
							continue
						}
						parsed, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						switch row := parsed.(type) {
						case *types.EventDelete:
							require.False(t, actualDeletes[row.ObjectId])
							actualDeletes[row.ObjectId] = true
						case *types.EventGrid:
							_, duplicate := actualGridEvents[row.GridRecord.AttributeId]
							require.False(t, duplicate)
							actualGridEvents[row.GridRecord.AttributeId] = row.GridRecord.Value
						case *types.EventPermission:
							_, duplicate := actualPermissionEvents[row.PermissionRecord.PermissionId]
							require.False(t, duplicate)
							actualPermissionEvents[row.PermissionRecord.PermissionId] = row.PermissionRecord.Value
						case *types.EventGuildRankPermission:
							actualRankEvents = append(actualRankEvents, row.GuildRankPermissionRecord)
						case *types.EventPlayer:
							_, duplicate := actualPlayerEvents[row.Player.Id]
							require.False(t, duplicate)
							actualPlayerEvents[row.Player.Id] = *row.Player
						}
					}
					expectedDeletes := map[string]bool{}
					if success {
						expectedDeletes = deleted
					}
					require.Equal(t, expectedDeletes, actualDeletes)
					require.Equal(t, gridEvents, actualGridEvents)
					require.Equal(t, permissionEvents, actualPermissionEvents)
					require.ElementsMatch(t, rankEvents, actualRankEvents)
					require.Equal(t, playerEvents, actualPlayerEvents)
					signerKey, err := collections.EncodeKeyWithPrefix(authtypes.AddressStoreKeyPrefix.Bytes(), sdk.AccAddressKey, address)
					require.NoError(t, err)
					delete(beforeAssets, authtypes.StoreKey+"/"+string(signerKey))
					afterAssets := bankAndAuthStateRows(bApp, current)
					delete(afterAssets, authtypes.StoreKey+"/"+string(signerKey))
					require.Equal(t, beforeAssets, afterAssets)
					sequence := signer.GetSequence() + 1
					if scenario == "delete_key_ceiling" {
						sequence--
					}
					require.Equal(t, sequence, bApp.AccountKeeper.GetAccount(current, address).GetSequence())
					if scenario == "direct" {
						_, err = bApp.Commit()
						require.NoError(t, err)
						retry := deliver(2, sequence)
						require.NotZero(t, retry.Code)
						require.Contains(t, retry.Log, "not found")
						checkState(bApp.NewContext(false))
						require.Equal(t, sequence+1, bApp.AccountKeeper.GetAccount(bApp.NewContext(false), address).GetSequence())
						retryAssets := bankAndAuthStateRows(bApp, bApp.NewContext(false))
						delete(retryAssets, authtypes.StoreKey+"/"+string(signerKey))
						require.Equal(t, beforeAssets, retryAssets)
						for _, event := range retry.Events {
							require.NotContains(t, []string{"structs.structs.EventDelete", "structs.structs.EventGrid", "structs.structs.EventPermission", "structs.structs.EventGuildRankPermission", "structs.structs.EventPlayer", "transfer"}, event.Type)
						}
					}
				})
			}
		}
	}
}
