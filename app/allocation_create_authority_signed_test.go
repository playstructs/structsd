package app_test

import (
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

func TestSignedAllocationCreateSourceAndControllerAuthority(t *testing.T) {
	for _, sourceKind := range []string{"player", "guild", "player_station", "guild_station"} {
		for _, controllerKind := range []string{"default", "player", "guild"} {
			for _, scenario := range []string{"direct", "explicit_admin", "source_owner", "rank", "demoted", "departed", "member_only", "controller_grant", "key_ceiling", "zero", "over_capacity", "later_failure", "missing_controller", "noncanonical_controller", "wrong_controller_kind", "missing_source"} {
				t.Run(sourceKind+"/"+controllerKind+"/"+scenario, func(t *testing.T) {
					bApp, ctx := setupJailGateApp(t)
					k := bApp.StructsKeeper
					newPlayer := func() types.Player {
						return appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
					}
					actor, sourceOwner, recipient := newPlayer(), newPlayer(), newPlayer()
					if scenario == "source_owner" {
						sourceOwner = actor
					}
					key := secp256k1.GenPrivKey()
					address := sdk.AccAddress(key.PubKey().Address())
					require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), actor.Index))
					sourceGuild, controllerGuild := types.CreateEmptyGuild(), types.CreateEmptyGuild()
					sourceGuild.Id, sourceGuild.Index, sourceGuild.Owner, sourceGuild.Creator = "0-972", 972, sourceOwner.Id, sourceOwner.Creator
					controllerGuild.Id, controllerGuild.Index, controllerGuild.Owner, controllerGuild.Creator = "0-973", 973, recipient.Id, recipient.Creator
					k.SetGuild(ctx, sourceGuild)
					k.SetGuild(ctx, controllerGuild)
					source, sourceAssetOwner := sourceOwner.Id, sourceOwner.Id
					if strings.HasPrefix(sourceKind, "guild") {
						source, sourceAssetOwner = sourceGuild.Id, sourceGuild.Id
					}
					stationSource := strings.HasSuffix(sourceKind, "_station")
					if stationSource {
						source = "4-972"
						k.SetSubstation(ctx, types.Substation{Id: source, Owner: sourceAssetOwner})
						k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceOwner.Id), 1000)
						cc := k.NewCurrentContext(ctx)
						_, err := cc.NewAllocation(types.AllocationType_dynamic, sourceOwner.Id, source, sourceOwner.Creator, sourceOwner.Id, 1000)
						require.NoError(t, err)
						cc.CommitAll()
					} else {
						k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, source), 1000)
					}
					controller, requestedController := actor.Id, ""
                    guildSource := strings.HasPrefix(sourceKind, "guild")
                    if guildSource && controllerKind == "default" { controller = sourceGuild.Id }
					if controllerKind == "player" {
						controller, requestedController = recipient.Id, recipient.Id
					}
					if controllerKind == "guild" {
						controller, requestedController = controllerGuild.Id, controllerGuild.Id
					}
					grant, bits, wantError := true, types.PermSourceAllocation, ""
					switch scenario {
					case "source_owner":
						grant = false
					case "rank", "demoted", "departed", "member_only":
						grant = false
						actor.GuildId, actor.GuildRank = sourceGuild.Id, 3
						if scenario == "demoted" {
							actor.GuildRank, wantError = 4, "permission"
						}
						if scenario == "departed" {
							actor.GuildId, wantError = "", "permission"
						}
						k.SetPlayer(ctx, actor)
						if scenario == "member_only" {
							wantError = "permission"
						} else {
							k.SetGuildRankPermissionStoreOnly(ctx, source, sourceGuild.Id, types.PermSourceAllocation, 3)
						}
					case "controller_grant":
						grant, wantError = false, "permission"
                        // The guild is both source and default controller here:
                        // this grant is actual source authority, not a bypass.
                        if sourceKind == "guild" && controllerKind == "default" { wantError = "" }
                        k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(controller, actor.Id), types.PermSourceAllocation)
					case "key_ceiling":
						bits, wantError = types.PermUpdate, "permission"
					case "zero":
						wantError = "new_power_zero"
					case "over_capacity":
						wantError = "does not have capacity"
					case "later_failure":
						wantError = "message index: 1"
					case "missing_controller":
						requestedController, wantError = "1-999999", "not found"
						if controllerKind == "guild" {
							requestedController = "0-999999"
						}
					case "noncanonical_controller":
						requestedController, wantError = "1-01", "not found"
					case "wrong_controller_kind":
						requestedController, wantError = "4-972", "not found"
					case "missing_source":
						wantError = "permission"
						if sourceKind == "guild" {
							wantError = "not found"
						}
					}
                    // Exporting guild power to a different controller is admin
                    // authority. Keep validation/rollback fixtures authorized so
                    // they still exercise the intended later failure.
                    admin := scenario == "explicit_admin" || scenario == "source_owner" || scenario == "zero" || scenario == "over_capacity" || scenario == "later_failure"
                    if admin { bits |= types.PermAdmin }
                    if guildSource && controllerKind != "default" && (scenario == "direct" || scenario == "rank") { wantError = "admin permission" }
                    if scenario == "missing_source" && (stationSource || sourceKind == "guild") { wantError = "not found" }
					if grant {
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(source, actor.Id), types.PermSourceAllocation)
					}
                    if admin && grant { k.SetPermissionsByBytes(ctx,keeperlib.GetObjectPermissionIDBytes(source,actor.Id),types.PermSourceAllocation|types.PermAdmin) }
					k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
					power := uint64(100)
					if scenario == "zero" {
						power = 0
					}
					if scenario == "over_capacity" {
						power = 1001
					}
					msg := &types.MsgAllocationCreate{Creator: address.String(), SourceObjectId: source, Controller: requestedController, AllocationType: types.AllocationType_dynamic, Power: power}
					if scenario == "missing_source" {
						msg.SourceObjectId = strings.Split(source, "-")[0] + "-999999"
					}
					messages := []sdk.Msg{msg}
					if scenario == "later_failure" {
						messages = append(messages, &types.MsgAllocationCreate{Creator: address.String(), SourceObjectId: source, Controller: "1-999999", Power: 1})
					}
					beforeCount, beforeAllocations := k.GetAllocationCount(ctx), k.GetAllAllocation(ctx)
					id := keeperlib.GetObjectID(types.ObjectType_allocation, beforeCount)
					permissionMap := func(rows []*types.PermissionRecord) map[string]uint64 {
						result := map[string]uint64{}
						for _, row := range rows {
							result[row.PermissionId] = row.Value
						}
						return result
					}
					expectedPermissions := permissionMap(k.GetAllPermissionExport(ctx))
					beforeRanks, beforeGrid := k.GetAllGuildRankPermissionExport(ctx), k.GetAllGridExport(ctx)
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
					newPermissions := map[string]uint64{}
					if wantError != "" {
						require.NotZero(t, result.Code)
						require.Contains(t, result.Log, wantError)
						require.NotContains(t, result.Log, "panic")
						require.Equal(t, beforeCount, k.GetAllocationCount(current))
						require.Equal(t, beforeAllocations, k.GetAllAllocation(current))
						require.Equal(t, beforeGrid, k.GetAllGridExport(current))
						require.Empty(t, k.GetAllAllocationIdBySourceIndex(current, source))
					} else {
						require.Zero(t, result.Code, result.Log)
						require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
						expected := types.Allocation{Id: id, Index: beforeCount, SourceObjectId: source, Controller: controller, Creator: address.String(), Type: types.AllocationType_dynamic}
						require.Equal(t, append(beforeAllocations, expected), k.GetAllAllocation(current))
						require.Equal(t, beforeCount+1, k.GetAllocationCount(current))
						require.Equal(t, []string{id}, k.GetAllAllocationIdBySourceIndex(current, source))
						require.Equal(t, power, k.GetGridAttribute(current, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, id)))
						require.Equal(t, power, k.GetGridAttribute(current, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, source)))
						if stationSource {
							require.Equal(t, uint64(900)-power, k.GetGridAttribute(current, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, source)))
						}
						if !strings.HasPrefix(controller, "0-") {
							creatorBits := types.PermUpdate | types.PermDelete
							if controllerKind == "default" {
								creatorBits |= types.PermAllocationConnection | types.PermAdmin
							} else {
								newPermissions[string(keeperlib.GetObjectPermissionIDBytes(id, recipient.Id))] = uint64(types.PermAllocationConnection | types.PermAdmin)
							}
							newPermissions[string(keeperlib.GetObjectPermissionIDBytes(id, actor.Id))] = uint64(creatorBits)
						}
					}
					for permission, value := range newPermissions {
						expectedPermissions[permission] = value
					}
					require.Equal(t, expectedPermissions, permissionMap(k.GetAllPermissionExport(current)))
					require.Equal(t, beforeRanks, k.GetAllGuildRankPermissionExport(current))
					require.Empty(t, k.GetAllAllocationIdByDestinationIndex(current, ""))
					_, automated := k.GetAutoResizeAllocationBySource(current, source)
					require.False(t, automated)
					allocationEvents, permissionEvents := 0, map[string]uint64{}
					for _, event := range result.Events {
						require.NotContains(t, []string{"coin_spent", "coin_received", "transfer", "structs.structs.EventDelete", "structs.structs.EventGuildRankPermission"}, event.Type)
						if wantError != "" {
							require.NotContains(t, []string{"structs.structs.EventAllocation", "structs.structs.EventGrid", "structs.structs.EventPermission"}, event.Type)
						}
						if event.Type == "structs.structs.EventAllocation" {
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							stored, found := k.GetAllocation(current, id)
							require.True(t, found)
							require.Equal(t, stored, *parsed.(*types.EventAllocation).Allocation)
							allocationEvents++
						}
						if event.Type == "structs.structs.EventPermission" {
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							row := parsed.(*types.EventPermission).PermissionRecord
							require.NotContains(t, permissionEvents, row.PermissionId)
							permissionEvents[row.PermissionId] = row.Value
						}
						if event.Type == "structs.structs.EventGrid" {
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							row := parsed.(*types.EventGrid).GridRecord
							require.Equal(t, k.GetGridAttribute(current, row.AttributeId), row.Value)
						}
					}
					require.Equal(t, newPermissions, permissionEvents)
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
					wantSequence := signer.GetSequence() + 1
					if scenario == "key_ceiling" {
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
