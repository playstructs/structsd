package app_test

import (
	"fmt"
	"math/rand"
	"strings"
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

// Supply can be redirected by allocation authority. Either allocation authority
// or the current recipient may disconnect it; neither the source nor a different
// recipient gains that authority merely by owning an adjacent grid object.
func TestSignedSubstationAllocationAuthority(t *testing.T) {
	for _, guildAllocation := range []bool{false, true} {
		for _, guildStation := range []bool{false, true} {
			for _, action := range []string{"connect", "disconnect"} {
				for _, scenario := range []string{
					"allocation_grant", "station_grant", "both", "neither", "source_grant", "wrong_station",
					"allocation_owner", "station_owner", "allocation_rank", "station_rank",
					"allocation_rank_demoted", "station_rank_demoted", "allocation_rank_departed", "station_rank_departed",
					"key_ceiling", "missing_allocation", "missing_destination", "same_destination", "source_destination", "later_failure",
				} {
					if action == "disconnect" && (scenario == "missing_destination" || scenario == "same_destination" || scenario == "source_destination") {
						continue
					}
					t.Run(fmt.Sprintf("allocationGuild=%t/stationGuild=%t/%s/%s", guildAllocation, guildStation, action, scenario), func(t *testing.T) {
						bApp, ctx := setupJailGateApp(t)
						k := bApp.StructsKeeper
						newPlayer := func() types.Player {
							return appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
						}
						actor, allocationOwner, stationOwner, sourceOwner := newPlayer(), newPlayer(), newPlayer(), newPlayer()
						key := secp256k1.GenPrivKey()
						address := sdk.AccAddress(key.PubKey().Address())
						require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), actor.Index))
						// The recovery primary retains all bits; the transaction uses
						// a separate key holding just the operation's required bit.
						if scenario == "allocation_owner" {
							allocationOwner = actor
						}
						if scenario == "station_owner" {
							stationOwner = actor
						}
						allocationGuild, stationGuild := types.CreateEmptyGuild(), types.CreateEmptyGuild()
						allocationGuild.Id, allocationGuild.Index, allocationGuild.Owner, allocationGuild.Creator = "0-960", 960, allocationOwner.Id, allocationOwner.Creator
						stationGuild.Id, stationGuild.Index, stationGuild.Owner, stationGuild.Creator = "0-961", 961, stationOwner.Id, stationOwner.Creator
						k.SetGuild(ctx, allocationGuild)
						k.SetGuild(ctx, stationGuild)
						controller, recipientOwner := allocationOwner.Id, stationOwner.Id
						if guildAllocation {
							controller = allocationGuild.Id
						}
						if guildStation {
							recipientOwner = stationGuild.Id
						}
						const source, oldStation, newStation = "4-960", "4-961", "4-962"
						k.SetSubstation(ctx, types.Substation{Id: source, Owner: sourceOwner.Id})
						for _, id := range []string{oldStation, newStation} {
							k.SetSubstation(ctx, types.Substation{Id: id, Owner: recipientOwner})
						}
						k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceOwner.Id), 1000)
						cc := k.NewCurrentContext(ctx)
						_, err := cc.NewAllocation(types.AllocationType_dynamic, sourceOwner.Id, source, sourceOwner.Creator, sourceOwner.Id, 1000)
						require.NoError(t, err)
						allocation, err := cc.NewAllocation(types.AllocationType_dynamic, source, oldStation, sourceOwner.Creator, controller, 100)
						require.NoError(t, err)
						id := allocation.ID()
						cc.CommitAll()
						allocationAllowed, stationAllowed := false, false
						grant := func(object string, bit types.Permission) {
							k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(object, actor.Id), bit)
						}
						switch scenario {
						case "allocation_grant", "key_ceiling", "missing_allocation", "missing_destination", "same_destination", "source_destination", "later_failure":
							grant(id, types.PermAllocationConnection)
							allocationAllowed = true
						case "station_grant":
							grant(oldStation, types.PermAllocationConnection)
							stationAllowed = true
						case "both":
							grant(id, types.PermAllocationConnection)
							grant(oldStation, types.PermAllocationConnection)
							allocationAllowed, stationAllowed = true, true
						case "source_grant":
							grant(source, types.PermSourceAllocation|types.PermAllocationConnection)
						case "wrong_station":
							grant(newStation, types.PermAllocationConnection)
						case "allocation_owner":
							allocationAllowed = true
						case "station_owner":
							stationAllowed = true
						}
						if strings.Contains(scenario, "_rank") {
							object, guild := id, allocationGuild.Id
							if strings.HasPrefix(scenario, "station_") {
								object, guild = oldStation, stationGuild.Id
							}
							actor.GuildId, actor.GuildRank = guild, 3
							if strings.HasSuffix(scenario, "_demoted") {
								actor.GuildRank = 4
							}
							if strings.HasSuffix(scenario, "_departed") {
								actor.GuildId = ""
							}
							k.SetPlayer(ctx, actor)
							k.SetGuildRankPermissionStoreOnly(ctx, object, guild, types.PermAllocationConnection, 3)
							allocationAllowed, stationAllowed = scenario == "allocation_rank", scenario == "station_rank"
						}
						bits := types.PermAllocationConnection
						if scenario == "key_ceiling" {
							bits = types.PermPlay
						}
						k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
						messageID, destination := id, newStation
						wantError := ""
						if !(allocationAllowed || (action == "disconnect" && stationAllowed)) || scenario == "key_ceiling" {
							wantError = "permission"
						}
						switch scenario {
						case "missing_allocation":
							messageID, wantError = "6-999999", "not found"
						case "missing_destination":
							destination, wantError = "4-999999", "not found"
						case "same_destination":
							destination, wantError = oldStation, "cannot change to same destination"
						case "source_destination":
							destination, wantError = source, "source_destination_match"
						case "later_failure":
							wantError = "message index: 1"
						}
						var msg sdk.Msg = &types.MsgSubstationAllocationConnect{Creator: address.String(), AllocationId: messageID, DestinationId: destination}
						if action == "disconnect" {
							msg = &types.MsgSubstationAllocationDisconnect{Creator: address.String(), AllocationId: messageID}
						}
						messages := []sdk.Msg{msg}
						if scenario == "later_failure" {
							messages = append(messages, &types.MsgSubstationAllocationDisconnect{Creator: address.String(), AllocationId: "6-999999"})
						}
						before, found := k.GetAllocation(ctx, id)
						require.True(t, found)
						beforeGrid, beforePermissions := k.GetAllGridExport(ctx), k.GetAllPermissionExport(ctx)
						beforeRanks := k.GetAllGuildRankPermissionExport(ctx)
						beforeAssets, beforeCount := bankAndAuthStateRows(bApp, ctx), k.GetAllocationCount(ctx)
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
						expected := before
						if wantError != "" {
							require.NotZero(t, result.Code)
							require.Contains(t, result.Log, wantError)
							require.NotContains(t, result.Log, "panic")
							require.Equal(t, beforeGrid, k.GetAllGridExport(current))
						} else {
							require.Zero(t, result.Code, result.Log)
							require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
							expected.DestinationId = newStation
							if action == "disconnect" {
								expected.DestinationId = ""
							}
						}
						stored, found := k.GetAllocation(current, id)
						require.True(t, found)
						require.Equal(t, expected, stored)
						require.Equal(t, beforeCount, k.GetAllocationCount(current))
						require.Equal(t, beforePermissions, k.GetAllPermissionExport(current))
						require.Equal(t, beforeRanks, k.GetAllGuildRankPermissionExport(current))
						require.Equal(t, []string{id}, k.GetAllAllocationIdBySourceIndex(current, source))
						for _, station := range []string{oldStation, newStation} {
							capacity := uint64(0)
							if expected.DestinationId == station {
								capacity = 100
								require.Equal(t, []string{id}, k.GetAllAllocationIdByDestinationIndex(current, station))
							} else {
								require.Empty(t, k.GetAllAllocationIdByDestinationIndex(current, station))
							}
							require.Equal(t, capacity, k.GetGridAttribute(current, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, station)))
						}
						require.Equal(t, uint64(100), k.GetGridAttribute(current, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, id)))
						require.Equal(t, uint64(100), k.GetGridAttribute(current, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, source)))
						allocationEvents := 0
						for _, event := range result.Events {
							require.NotContains(t, []string{"coin_spent", "coin_received", "transfer", "structs.structs.EventDelete", "structs.structs.EventPermission", "structs.structs.EventGuildRankPermission"}, event.Type)
							if wantError != "" {
								require.NotContains(t, []string{"structs.structs.EventAllocation", "structs.structs.EventGrid"}, event.Type)
							}
							if event.Type == "structs.structs.EventAllocation" {
								parsed, err := sdk.ParseTypedEvent(event)
								require.NoError(t, err)
								require.Equal(t, stored, *parsed.(*types.EventAllocation).Allocation)
								allocationEvents++
							}
							if event.Type == "structs.structs.EventGrid" {
								parsed, err := sdk.ParseTypedEvent(event)
								require.NoError(t, err)
								row := parsed.(*types.EventGrid).GridRecord
								require.Equal(t, k.GetGridAttribute(current, row.AttributeId), row.Value, "indexer replacement agrees with committed grid")
							}
						}
						if wantError == "" {
							require.Equal(t, 1, allocationEvents)
						}
						afterAssets := bankAndAuthStateRows(bApp, current)
						signerKey, err := collections.EncodeKeyWithPrefix(authtypes.AddressStoreKeyPrefix.Bytes(), sdk.AccAddressKey, address)
						require.NoError(t, err)
						signerRow := authtypes.StoreKey + "/" + string(signerKey)
						delete(beforeAssets, signerRow)
						delete(afterAssets, signerRow)
						require.Equal(t, beforeAssets, afterAssets)
						afterSigner := bApp.AccountKeeper.GetAccount(current, address)
						wantSequence := signer.GetSequence() + 1
						if scenario == "key_ceiling" {
							wantSequence-- // fixed ante bit refuses before sequence changes
						}
						require.Equal(t, signer.GetAccountNumber(), afterSigner.GetAccountNumber())
						require.Equal(t, wantSequence, afterSigner.GetSequence())
					})
				}
			}
		}
	}
}
