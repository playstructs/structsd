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

// A controller may redirect supply, but resizing consumes the source's power.
// Controller ownership/Update/SourceAllocation on the allocation are insufficient.
func TestSignedAllocationResizeSourceAuthority(t *testing.T) {
	for _, sourceKind := range []string{"player", "guild", "player_station", "guild_station"} {
		for _, guildController := range []bool{false, true} {
			for _, scenario := range []string{"direct", "source_owner", "rank", "demoted", "departed", "controller_owner", "allocation_update", "allocation_source", "neither", "key_ceiling", "static", "missing", "zero", "over_capacity", "decrease", "unchanged", "later_failure"} {
				t.Run(fmt.Sprintf("%s/controllerGuild=%t/%s", sourceKind, guildController, scenario), func(t *testing.T) {
					bApp, ctx := setupJailGateApp(t)
					k := bApp.StructsKeeper
					newPlayer := func() types.Player {
						return appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
					}
					actor, sourceOwner, controllerOwner := newPlayer(), newPlayer(), newPlayer()
					if scenario == "source_owner" {
						sourceOwner = actor
					}
					if scenario == "controller_owner" {
						controllerOwner = actor
					}
					key := secp256k1.GenPrivKey()
					address := sdk.AccAddress(key.PubKey().Address())
					require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), actor.Index))
					sourceGuild, controlGuild := types.CreateEmptyGuild(), types.CreateEmptyGuild()
					sourceGuild.Id, sourceGuild.Index, sourceGuild.Owner, sourceGuild.Creator = "0-965", 965, sourceOwner.Id, sourceOwner.Creator
					controlGuild.Id, controlGuild.Index, controlGuild.Owner, controlGuild.Creator = "0-966", 966, controllerOwner.Id, controllerOwner.Creator
					k.SetGuild(ctx, sourceGuild)
					k.SetGuild(ctx, controlGuild)
					source, sourceAssetOwner, controller := sourceOwner.Id, sourceOwner.Id, controllerOwner.Id
					if strings.HasPrefix(sourceKind, "guild") {
						source, sourceAssetOwner = sourceGuild.Id, sourceGuild.Id
					}
					if guildController {
						controller = controlGuild.Id
					}
					stationSource := strings.HasSuffix(sourceKind, "_station")
					cc := k.NewCurrentContext(ctx)
					if stationSource {
						source = "4-965"
						k.SetSubstation(ctx, types.Substation{Id: source, Owner: sourceAssetOwner})
						k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceOwner.Id), 1000)
						_, err := cc.NewAllocation(types.AllocationType_dynamic, sourceOwner.Id, source, sourceOwner.Creator, sourceOwner.Id, 1000)
						require.NoError(t, err)
					} else {
						k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, source), 1000)
					}
					const destination, otherDestination = "4-966", "4-967"
					for _, id := range []string{destination, otherDestination} {
						k.SetSubstation(ctx, types.Substation{Id: id, Owner: controller})
					}
					kind := types.AllocationType_dynamic
					if scenario == "static" {
						kind = types.AllocationType_static
					}
					allocation, err := cc.NewAllocation(kind, source, destination, sourceOwner.Creator, controller, 100)
					require.NoError(t, err)
					id := allocation.ID()
					sibling, err := cc.NewAllocation(types.AllocationType_dynamic, source, otherDestination, sourceOwner.Creator, controller, 50)
					require.NoError(t, err)
					cc.CommitAll()
					// Seed the correct initial share to isolate resizing from
					// creation; the creation path has its own regression below.
					if stationSource {
						k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, source), 750)
					}
					bits, grant, wantError := types.PermSourceAllocation, true, ""
					switch scenario {
					case "source_owner":
						grant = false
					case "rank", "demoted", "departed":
						grant = false
						actor.GuildId, actor.GuildRank = sourceGuild.Id, 3
						if scenario == "demoted" {
							actor.GuildRank, wantError = 4, "permission"
						}
						if scenario == "departed" {
							actor.GuildId, wantError = "", "permission"
						}
						k.SetPlayer(ctx, actor)
						k.SetGuildRankPermissionStoreOnly(ctx, source, sourceGuild.Id, types.PermSourceAllocation, 3)
					case "controller_owner", "allocation_update", "allocation_source", "neither":
						grant, wantError = false, "permission"
						if scenario == "allocation_update" {
							k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(id, actor.Id), types.PermUpdate)
						}
						if scenario == "allocation_source" {
							k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(id, actor.Id), types.PermSourceAllocation)
						}
					case "key_ceiling":
						bits, wantError = types.PermUpdate, "permission"
					case "static":
						wantError = "incorrect_type"
					case "missing":
						wantError = "not found"
					case "zero":
						wantError = "new_power_zero"
					case "over_capacity":
						wantError = "does not have capacity"
					case "later_failure":
						wantError = "message index: 1"
					}
					if grant {
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(source, actor.Id), types.PermSourceAllocation)
					}
					k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
					power := uint64(200)
					switch scenario {
					case "zero":
						power = 0
					case "over_capacity":
						power = 1000 // sibling's 50 is still reserved
					case "decrease":
						power = 50
					case "unchanged":
						power = 100
					}
					msg := &types.MsgAllocationUpdate{Creator: address.String(), AllocationId: id, Power: power}
					if scenario == "missing" {
						msg.AllocationId = "6-999999"
					}
					messages := []sdk.Msg{msg}
					if scenario == "later_failure" {
						messages = append(messages, &types.MsgAllocationUpdate{Creator: address.String(), AllocationId: "6-999999", Power: power})
					}
					gridMap := func(rows []*types.GridRecord) map[string]uint64 {
						result := map[string]uint64{}
						for _, row := range rows {
							result[row.AttributeId] = row.Value
						}
						return result
					}
					beforeAllocations, beforeCount := k.GetAllAllocation(ctx), k.GetAllocationCount(ctx)
					beforePermissions, beforeRanks := k.GetAllPermissionExport(ctx), k.GetAllGuildRankPermissionExport(ctx)
					expectedGrid, beforeAssets := gridMap(k.GetAllGridExport(ctx)), bankAndAuthStateRows(bApp, ctx)
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
					if wantError != "" {
						require.NotZero(t, result.Code)
						require.Contains(t, result.Log, wantError)
						require.NotContains(t, result.Log, "panic")
					} else {
						require.Zero(t, result.Code, result.Log)
						require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
						expectedGrid[keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, id)] = power
						expectedGrid[keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, source)] = power + 50
						expectedGrid[keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, destination)] = power
						expectedGrid[keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, destination)] = power*9/10
						if stationSource {
							expectedGrid[keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, source)] = 900 - power - 50
						}
					}
					require.Equal(t, expectedGrid, gridMap(k.GetAllGridExport(current)))
					require.Equal(t, beforeAllocations, k.GetAllAllocation(current))
					require.Equal(t, beforeCount, k.GetAllocationCount(current))
					require.Equal(t, beforePermissions, k.GetAllPermissionExport(current))
					require.Equal(t, beforeRanks, k.GetAllGuildRankPermissionExport(current))
					require.ElementsMatch(t, []string{id, sibling.ID()}, k.GetAllAllocationIdBySourceIndex(current, source))
					require.Equal(t, []string{id}, k.GetAllAllocationIdByDestinationIndex(current, destination))
					require.Equal(t, []string{sibling.ID()}, k.GetAllAllocationIdByDestinationIndex(current, otherDestination))
					for _, event := range result.Events {
						require.NotContains(t, []string{"coin_spent", "coin_received", "transfer", "structs.structs.EventAllocation", "structs.structs.EventDelete", "structs.structs.EventPermission", "structs.structs.EventGuildRankPermission"}, event.Type)
						if event.Type == "structs.structs.EventGrid" {
							require.Empty(t, wantError)
							require.NotEqual(t, "unchanged", scenario)
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							row := parsed.(*types.EventGrid).GridRecord
							require.Equal(t, expectedGrid[row.AttributeId], row.Value)
						}
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
