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

func TestSignedSubstationCreationAuthorityAndAllocationReuse(t *testing.T) {
	for _, guildAllocation := range []bool{false, true} {
		for _, ownerKind := range []string{"default", "player", "guild"} {
			for _, scenario := range []string{"direct", "allocation_only", "owner_only", "neither", "allocation_owner", "guild_owner", "allocation_rank", "allocation_demoted", "allocation_departed", "guild_rank", "guild_demoted", "guild_departed", "source_only", "key_ceiling", "already_connected", "missing_allocation", "missing_owner", "wrong_owner_kind", "later_failure", "repeat_allocation"} {
				if strings.HasPrefix(scenario, "guild_") && ownerKind != "guild" {
					continue
				}
				if (scenario == "missing_owner" || scenario == "wrong_owner_kind") && ownerKind == "default" {
					continue
				}
				t.Run(fmt.Sprintf("allocationGuild=%t/owner=%s/%s", guildAllocation, ownerKind, scenario), func(t *testing.T) {
					bApp, ctx := setupReactorAppGenesis(t, 1, true)
					k := bApp.StructsKeeper
					newPlayer := func() types.Player {
						return appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
					}
					actor, allocationOwner, recipient, source := newPlayer(), newPlayer(), newPlayer(), newPlayer()
					if scenario == "allocation_owner" {
						allocationOwner = actor
					}
					if scenario == "guild_owner" {
						recipient = actor
					}
					key := secp256k1.GenPrivKey()
					address := sdk.AccAddress(key.PubKey().Address())
					require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), actor.Index))
					controlGuild, receivingGuild := types.CreateEmptyGuild(), types.CreateEmptyGuild()
					controlGuild.Id, controlGuild.Index, controlGuild.Owner, controlGuild.Creator = "0-984", 984, allocationOwner.Id, allocationOwner.Creator
					receivingGuild.Id, receivingGuild.Index, receivingGuild.Owner, receivingGuild.Creator = "0-985", 985, recipient.Id, recipient.Creator
					k.SetGuild(ctx, controlGuild)
					k.SetGuild(ctx, receivingGuild)
					controller := allocationOwner.Id
					if guildAllocation {
						controller = controlGuild.Id
					}
					owner, requestedOwner := actor.Id, ""
					if ownerKind == "player" {
						owner, requestedOwner = recipient.Id, recipient.Id
					}
					if ownerKind == "guild" {
						owner, requestedOwner = receivingGuild.Id, receivingGuild.Id
					}
					k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, source.Id), 1000)
					cc := k.NewCurrentContext(ctx)
					allocation, err := cc.NewAllocation(types.AllocationType_dynamic, source.Id, "", source.Creator, controller, 100)
					require.NoError(t, err)
					cc.CommitAll()
					grantAllocation, grantOwner := true, true
					wantError := ""
					switch scenario {
					case "allocation_only":
						grantOwner = false
						if ownerKind == "guild" {
							wantError = "permission"
						}
					case "owner_only":
						grantAllocation = false
						wantError = "permission"
					case "neither":
						grantAllocation, grantOwner = false, false
						wantError = "permission"
					case "allocation_owner":
						grantAllocation = false
					case "guild_owner":
						grantOwner = false
					case "allocation_rank", "allocation_demoted", "allocation_departed":
						grantAllocation = false
						actor.GuildId, actor.GuildRank = controlGuild.Id, 3
						if scenario == "allocation_demoted" {
							actor.GuildRank = 4
							wantError = "permission"
						}
						if scenario == "allocation_departed" {
							actor.GuildId = ""
							wantError = "permission"
						}
						k.SetPlayer(ctx, actor)
						k.SetGuildRankPermissionStoreOnly(ctx, allocation.ID(), controlGuild.Id, types.PermAllocationConnection, 3)
					case "guild_rank", "guild_demoted", "guild_departed":
						grantOwner = false
						actor.GuildId, actor.GuildRank = receivingGuild.Id, 3
						if scenario == "guild_demoted" {
							actor.GuildRank = 4
							wantError = "permission"
						}
						if scenario == "guild_departed" {
							actor.GuildId = ""
							wantError = "permission"
						}
						k.SetPlayer(ctx, actor)
						k.SetGuildRankPermissionStoreOnly(ctx, receivingGuild.Id, receivingGuild.Id, types.PermAllocationConnection, 3)
					case "source_only":
						grantAllocation = false
						wantError = "permission"
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(source.Id, actor.Id), types.PermAllocationConnection)
					}
					if grantAllocation {
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(allocation.ID(), actor.Id), types.PermAllocationConnection)
					}
					if grantOwner && ownerKind == "guild" {
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(owner, actor.Id), types.PermAllocationConnection)
					}
					bits := types.PermAllocationConnection
					if scenario == "key_ceiling" {
						bits = types.PermAll &^ types.PermAllocationConnection
						wantError = "address lacks required permission bits"
					}
					k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
					allocationID := allocation.ID()
					switch scenario {
					case "missing_allocation":
						allocationID = "6-999999"
						wantError = "not found"
					case "missing_owner":
						requestedOwner = "1-999999"
						if ownerKind == "guild" {
							requestedOwner = "0-999999"
						}
						wantError = "not found"
					case "wrong_owner_kind":
						requestedOwner = allocation.ID()
						wantError = "not found"
					case "already_connected":
						k.SetSubstation(ctx, types.Substation{Id: "4-986", Owner: source.Id})
						cc = k.NewCurrentContext(ctx)
						a, found := cc.GetAllocation(allocation.ID())
						require.True(t, found)
						require.NoError(t, a.SetDestination("4-986"))
						cc.CommitAll()
						wantError = "already feeds substation"
					case "later_failure":
						wantError = "message index: 1"
					case "repeat_allocation":
						wantError = "message index: 1"
					}
					original, found := k.GetAllocation(ctx, allocation.ID())
					require.True(t, found)
					beforeStations, beforeGrid := k.GetAllSubstation(ctx), k.GetAllGridExport(ctx)
					beforePermissions, beforeRanks := k.GetAllPermissionExport(ctx), k.GetAllGuildRankPermissionExport(ctx)
					beforeAssets := bankAndAuthStateRows(bApp, ctx)
					count := k.GetSubstationCount(ctx)
					expectedID := keeperlib.GetObjectID(types.ObjectType_substation, count)
					messages := []sdk.Msg{&types.MsgSubstationCreate{Creator: address.String(), Owner: requestedOwner, AllocationId: allocationID}}
					if scenario == "later_failure" {
						messages = append(messages, &types.MsgSubstationCreate{Creator: address.String(), Owner: requestedOwner, AllocationId: "6-999999"})
					}
					if scenario == "repeat_allocation" {
						messages = append(messages, &types.MsgSubstationCreate{Creator: address.String(), Owner: requestedOwner, AllocationId: allocationID})
					}
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
					current := bApp.NewContext(false)
					success := wantError == ""
					if success {
						require.Zero(t, result.Code, result.Log)
						require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
					} else {
						require.NotZero(t, result.Code)
						require.Contains(t, result.Log, wantError)
						require.NotContains(t, result.Log, "panic")
					}
					expectedStations := append([]types.Substation{}, beforeStations...)
					expectedAllocation := original
					expectedPermissions := map[string]uint64{}
					for _, row := range beforePermissions {
						expectedPermissions[row.PermissionId] = row.Value
					}
					expectedGrid := map[string]uint64{}
					for _, row := range beforeGrid {
						expectedGrid[row.AttributeId] = row.Value
					}
					expectedGridEvents := map[string]uint64{}
					station := types.Substation{Id: expectedID, Owner: owner, Creator: address.String()}
					if success {
						expectedStations = append(expectedStations, station)
						expectedAllocation.DestinationId = expectedID
						count++
						for _, kind := range []types.GridAttributeType{types.GridAttributeType_capacity, types.GridAttributeType_connectionCapacity} {
							id := keeperlib.GetGridAttributeIDByObjectId(kind, expectedID)
							expectedGrid[id] = 100
							if kind == types.GridAttributeType_connectionCapacity {
								expectedGrid[id] = 90
							}
							expectedGridEvents[id] = expectedGrid[id]
						}
						if ownerKind != "guild" {
							expectedPermissions[string(keeperlib.GetObjectPermissionIDBytes(expectedID, owner))] = uint64(types.PermSubstationAll)
						}
					}
					require.Equal(t, count, k.GetSubstationCount(current))
					require.ElementsMatch(t, expectedStations, k.GetAllSubstation(current))
					actual, found := k.GetAllocation(current, allocation.ID())
					require.True(t, found)
					require.Equal(t, expectedAllocation, actual)
					actualPermissions := map[string]uint64{}
					for _, row := range k.GetAllPermissionExport(current) {
						actualPermissions[row.PermissionId] = row.Value
					}
					require.Equal(t, expectedPermissions, actualPermissions)
					require.ElementsMatch(t, beforeRanks, k.GetAllGuildRankPermissionExport(current))
					actualGrid := map[string]uint64{}
					for _, row := range k.GetAllGridExport(current) {
						actualGrid[row.AttributeId] = row.Value
					}
					require.Equal(t, expectedGrid, actualGrid)
					require.ElementsMatch(t, []string{allocation.ID()}, k.GetAllAllocationIdBySourceIndex(current, source.Id))
					if success {
						require.ElementsMatch(t, []string{allocation.ID()}, k.GetAllAllocationIdByDestinationIndex(current, expectedID))
					} else {
						require.Empty(t, k.GetAllAllocationIdByDestinationIndex(current, expectedID))
					}
					if original.DestinationId != "" {
						require.ElementsMatch(t, []string{allocation.ID()}, k.GetAllAllocationIdByDestinationIndex(current, original.DestinationId))
					}
					require.Empty(t, k.GetAllPlayerIdBySubstationIndex(current, expectedID), "creation never moves a player implicitly")
					stations, allocations, permissions := 0, 0, 0
					gridEvents := map[string]uint64{}
					for _, event := range result.Events {
						require.NotContains(t, []string{"transfer", "coin_spent", "coin_received", "coinbase", "burn", "structs.structs.EventVaultAddress", "structs.structs.EventGuildRankPermission"}, event.Type)
						if !success {
							require.NotContains(t, []string{"structs.structs.EventSubstation", "structs.structs.EventAllocation", "structs.structs.EventGrid", "structs.structs.EventPermission"}, event.Type)
						}
						switch event.Type {
						case "structs.structs.EventSubstation":
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							require.Equal(t, station, *parsed.(*types.EventSubstation).Substation)
							stations++
						case "structs.structs.EventAllocation":
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							require.Equal(t, expectedAllocation, *parsed.(*types.EventAllocation).Allocation)
							allocations++
						case "structs.structs.EventGrid":
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							row := parsed.(*types.EventGrid).GridRecord
							_, duplicate := gridEvents[row.AttributeId]
							require.False(t, duplicate)
							gridEvents[row.AttributeId] = row.Value
						case "structs.structs.EventPermission":
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							row := parsed.(*types.EventPermission).PermissionRecord
							require.Equal(t, string(keeperlib.GetObjectPermissionIDBytes(expectedID, owner)), row.PermissionId)
							require.Equal(t, uint64(types.PermSubstationAll), row.Value)
							permissions++
						}
					}
					require.Equal(t, expectedGridEvents, gridEvents)
					if success {
						require.Equal(t, 1, stations)
						require.Equal(t, 1, allocations)
						if ownerKind == "guild" {
							require.Zero(t, permissions)
						} else {
							require.Equal(t, 1, permissions)
						}
					} else {
						require.Zero(t, stations)
						require.Zero(t, allocations)
						require.Zero(t, permissions)
					}
					afterAssets := bankAndAuthStateRows(bApp, current)
					signerKey, err := collections.EncodeKeyWithPrefix(authtypes.AddressStoreKeyPrefix.Bytes(), sdk.AccAddressKey, address)
					require.NoError(t, err)
					delete(beforeAssets, authtypes.StoreKey+"/"+string(signerKey))
					delete(afterAssets, authtypes.StoreKey+"/"+string(signerKey))
					require.Equal(t, beforeAssets, afterAssets)
					expectedSequence := signer.GetSequence() + 1
					if scenario == "key_ceiling" {
						expectedSequence--
					}
					require.Equal(t, expectedSequence, bApp.AccountKeeper.GetAccount(current, address).GetSequence())
					if scenario == "direct" {
						// A successful committed creation cannot be replayed to
						// manufacture another empty station with the same input.
						_, err = bApp.Commit()
						require.NoError(t, err)
						retry, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(2)), config, messages, nil, ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{expectedSequence}, key)
						require.NoError(t, err)
						retryWire, err := config.TxEncoder()(retry)
						require.NoError(t, err)
						next, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: ctx.BlockTime().Add(time.Second), Txs: [][]byte{retryWire}})
						require.NoError(t, err)
						require.Len(t, next.TxResults, 1)
						require.NotZero(t, next.TxResults[0].Code)
						require.Contains(t, next.TxResults[0].Log, "already feeds substation")
						continued := bApp.NewContext(false)
						require.Equal(t, count, k.GetSubstationCount(continued))
						require.ElementsMatch(t, expectedStations, k.GetAllSubstation(continued))
						continuedAllocation, found := k.GetAllocation(continued, allocation.ID())
						require.True(t, found)
						require.Equal(t, expectedAllocation, continuedAllocation)
						continuedPermissions := map[string]uint64{}
						for _, row := range k.GetAllPermissionExport(continued) {
							continuedPermissions[row.PermissionId] = row.Value
						}
						require.Equal(t, expectedPermissions, continuedPermissions)
						continuedGrid := map[string]uint64{}
						for _, row := range k.GetAllGridExport(continued) {
							continuedGrid[row.AttributeId] = row.Value
						}
						require.Equal(t, expectedGrid, continuedGrid)
						require.ElementsMatch(t, []string{allocation.ID()}, k.GetAllAllocationIdByDestinationIndex(continued, expectedID))
						require.Equal(t, expectedSequence+1, bApp.AccountKeeper.GetAccount(continued, address).GetSequence())
						for _, event := range next.TxResults[0].Events {
							require.NotContains(t, []string{"structs.structs.EventSubstation", "structs.structs.EventAllocation", "structs.structs.EventGrid", "structs.structs.EventPermission", "transfer", "coin_spent", "coin_received"}, event.Type)
						}
					}
				})
			}
		}
	}
}
