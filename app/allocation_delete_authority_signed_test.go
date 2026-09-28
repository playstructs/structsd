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

// Allocation deletion has two independent authority paths. Ante must not
// disable the object's Delete path by requiring SourceAllocation on every key.
func TestSignedAllocationDeleteAlternativeAuthority(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		for _, scenario := range []string{"source_only", "delete_only", "source_owner", "allocation_owner", "rank_delete", "rank_demoted", "rank_departed", "source_key_delete_grant", "delete_key_source_grant", "both_grants_wrong_key", "neither_grant", "static_allocation", "missing_allocation", "later_failure"} {
			t.Run(fmt.Sprintf("guild=%t/%s", guildOwned, scenario), func(t *testing.T) {
				bApp, ctx := setupJailGateApp(t)
				k := bApp.StructsKeeper
				key := secp256k1.GenPrivKey()
				address := sdk.AccAddress(key.PubKey().Address())
				// Keep the recovery primary at PermAll and sign with a genuinely
				// separate delegated key, as a scoped production wallet would.
				actor := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
				require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), actor.Index))
				source := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
				owner := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
				if scenario == "source_owner" {
					source = actor
				}
				guild := types.CreateEmptyGuild()
				guild.Id, guild.Index, guild.Owner, guild.Creator = "0-957", 957, owner.Id, owner.Creator
				controller := owner.Id
				if scenario == "allocation_owner" {
					controller, guild.Owner = actor.Id, actor.Id
				}
				if guildOwned {
					controller = guild.Id
				}
				k.SetGuild(ctx, guild)
				const destination = "4-957"
				k.SetSubstation(ctx, types.Substation{Id: destination, Owner: owner.Id})
				k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, source.Id), 1000)
				allocationType := types.AllocationType_dynamic
				if scenario == "static_allocation" {
					allocationType = types.AllocationType_static
				}
				cc := k.NewCurrentContext(ctx)
				allocation, err := cc.NewAllocation(allocationType, source.Id, destination, source.Creator, controller, 100)
				require.NoError(t, err)
				id := allocation.ID()
				cc.CommitAll()
				bits, sourceGrant, deleteGrant := types.PermDelete, false, true
				wantError := ""
				switch scenario {
				case "source_only":
					bits, sourceGrant, deleteGrant = types.PermSourceAllocation, true, false
				case "source_owner":
					bits, deleteGrant = types.PermSourceAllocation, false
				case "allocation_owner":
					deleteGrant = false
				case "rank_delete", "rank_demoted", "rank_departed":
					deleteGrant = false
					actor.GuildId, actor.GuildRank = guild.Id, 3
					if scenario == "rank_demoted" {
						actor.GuildRank, wantError = 4, "permission"
					}
					if scenario == "rank_departed" {
						actor.GuildId, wantError = "", "permission"
					}
					k.SetPlayer(ctx, actor)
					k.SetGuildRankPermissionStoreOnly(ctx, id, guild.Id, types.PermDelete, 3)
				case "source_key_delete_grant":
					bits, wantError = types.PermSourceAllocation, "permission"
				case "delete_key_source_grant":
					sourceGrant, deleteGrant, wantError = true, false, "permission"
				case "both_grants_wrong_key":
					bits, sourceGrant, wantError = types.PermPlay, true, "permission"
				case "neither_grant":
					bits, deleteGrant, wantError = types.PermSourceAllocation|types.PermDelete, false, "permission"
				case "static_allocation":
					wantError = "allocation type (static vs dynamic)"
				case "missing_allocation":
					wantError = "not found"
				case "later_failure":
					wantError = "message index: 1"
				}
				if sourceGrant {
					k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(source.Id, actor.Id), types.PermSourceAllocation)
				}
				if deleteGrant {
					k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(id, actor.Id), types.PermDelete)
				}
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
				msg := &types.MsgAllocationDelete{Creator: address.String(), AllocationId: id}
				if scenario == "missing_allocation" {
					msg.AllocationId = "6-999999"
				}
				messages := []sdk.Msg{msg}
				if scenario == "later_failure" {
					messages = append(messages, &types.MsgAllocationDelete{Creator: address.String(), AllocationId: id})
				}
				before, found := k.GetAllocation(ctx, id)
				require.True(t, found)
				beforeGrid, beforePermissions := k.GetAllGridExport(ctx), k.GetAllPermissionExport(ctx)
				beforeAssets, beforeCount := bankAndAuthStateRows(bApp, ctx), k.GetAllocationCount(ctx)
				beforeRank := k.ReadGuildRankRegister(ctx, id, guild.Id)
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
				powerID := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, id)
				stored, found := k.GetAllocation(current, id)
				if wantError != "" {
					require.NotZero(t, result.Code)
					require.Contains(t, result.Log, wantError)
					require.NotContains(t, result.Log, "panic")
					require.True(t, found)
					require.Equal(t, before, stored)
					require.Equal(t, beforeGrid, k.GetAllGridExport(current))
					require.Equal(t, beforePermissions, k.GetAllPermissionExport(current))
					require.Equal(t, beforeRank, k.ReadGuildRankRegister(current, id, guild.Id))
					require.Equal(t, []string{id}, k.GetAllAllocationIdBySourceIndex(current, source.Id))
					require.Equal(t, []string{id}, k.GetAllAllocationIdByDestinationIndex(current, destination))
				} else {
					require.Zero(t, result.Code, result.Log)
					require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
					require.False(t, found)
					require.Nil(t, current.KVStore(bApp.GetKey(types.StoreKey)).Get([]byte(types.GridAttributeKey+powerID)))
					require.Empty(t, k.GetPermissionsByObject(current, id))
					var remaining []*types.PermissionRecord
					for _, permission := range beforePermissions {
						if !strings.HasPrefix(permission.PermissionId, id+"@") {
							remaining = append(remaining, permission)
						}
					}
					require.Equal(t, remaining, k.GetAllPermissionExport(current), "deletion cannot clear grants on other objects or addresses")
					require.Equal(t, [types.PermissionBitCount]uint64{}, k.ReadGuildRankRegister(current, id, guild.Id))
					require.Empty(t, k.GetAllAllocationIdBySourceIndex(current, source.Id))
					require.Empty(t, k.GetAllAllocationIdByDestinationIndex(current, destination))
					for object, attribute := range map[string]types.GridAttributeType{source.Id: types.GridAttributeType_load, destination: types.GridAttributeType_capacity} {
						require.Zero(t, k.GetGridAttribute(current, keeperlib.GetGridAttributeIDByObjectId(attribute, object)))
					}
				}
				require.Equal(t, beforeCount, k.GetAllocationCount(current), "deletion never reuses allocation identities")
				deletes, powerClears := 0, 0
				for _, event := range result.Events {
					require.NotContains(t, []string{"coin_spent", "coin_received", "transfer"}, event.Type)
					if wantError != "" {
						require.NotContains(t, []string{"structs.structs.EventDelete", "structs.structs.EventAllocation", "structs.structs.EventPermission", "structs.structs.EventGuildRankPermission"}, event.Type)
					}
					if event.Type == "structs.structs.EventDelete" {
						parsed, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						require.Equal(t, id, parsed.(*types.EventDelete).ObjectId)
						deletes++
					}
					if event.Type == "structs.structs.EventGrid" {
						parsed, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						row := parsed.(*types.EventGrid).GridRecord
						if row.AttributeId == powerID {
							require.Empty(t, wantError, "a rolled-back deletion cannot clear indexed power")
							require.Zero(t, row.Value)
							powerClears++
						}
					}
				}
				if wantError == "" {
					require.Equal(t, 1, deletes)
					require.Equal(t, 1, powerClears)
				}
				afterAssets := bankAndAuthStateRows(bApp, current)
				signerKey, err := collections.EncodeKeyWithPrefix(authtypes.AddressStoreKeyPrefix.Bytes(), sdk.AccAddressKey, address)
				require.NoError(t, err)
				signerRow := authtypes.StoreKey + "/" + string(signerKey)
				delete(beforeAssets, signerRow)
				delete(afterAssets, signerRow)
				require.Equal(t, beforeAssets, afterAssets)
				afterSigner := bApp.AccountKeeper.GetAccount(current, address)
				require.Equal(t, signer.GetAccountNumber(), afterSigner.GetAccountNumber())
				require.Equal(t, signer.GetSequence()+1, afterSigner.GetSequence(), "dynamic authority is enforced in the handler")
			})
		}
	}
}
