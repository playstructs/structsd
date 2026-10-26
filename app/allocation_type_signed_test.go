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

func TestSignedAllocationCreateTypeValidation(t *testing.T) {
	for _, guildSource := range []bool{false, true} {
		for _, guildController := range []bool{false, true} {
			for _, kind := range []types.AllocationType{0, 1, 2, 3, -1, 4, 500, 2147483647} {
				t.Run(fmt.Sprintf("guildSource=%t/guildController=%t/type=%d", guildSource, guildController, kind), func(t *testing.T) {
					bApp, ctx := setupJailGateApp(t)
					k := bApp.StructsKeeper
					actor := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
					recipient := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
					key := secp256k1.GenPrivKey()
					address := sdk.AccAddress(key.PubKey().Address())
					require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), actor.Index))
					k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), types.PermSourceAllocation|types.PermAdmin)
					source, controller := actor.Id, recipient.Id
					for i, owner := range []types.Player{actor, recipient} {
						guild := types.CreateEmptyGuild()
						guild.Index, guild.Id, guild.Owner, guild.Creator = uint64(963+i), fmt.Sprintf("0-%d", 963+i), owner.Id, owner.Creator
						k.SetGuild(ctx, guild)
						if i == 0 && guildSource {
							source = guild.Id
						}
						if i == 1 && guildController {
							controller = guild.Id
						}
					}
					k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, source), 1000)
					beforeCount, beforeGrid := k.GetAllocationCount(ctx), k.GetAllGridExport(ctx)
					beforeAssets := bankAndAuthStateRows(bApp, ctx)
					permissionMap := func(rows []*types.PermissionRecord) map[string]uint64 {
						result := map[string]uint64{}
						for _, row := range rows {
							result[row.PermissionId] = row.Value
						}
						return result
					}
					expectedPermissions := permissionMap(k.GetAllPermissionExport(ctx))
					id := keeperlib.GetObjectID(types.ObjectType_allocation, beforeCount)
					msg := &types.MsgAllocationCreate{Creator: address.String(), SourceObjectId: source, Controller: controller, AllocationType: kind, Power: 100}
					signer := bApp.AccountKeeper.GetAccount(ctx, address)
					config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
					tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, []sdk.Msg{msg}, nil,
						ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
					require.NoError(t, err)
					wire, err := config.TxEncoder()(tx)
					require.NoError(t, err)
					block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
					require.NoError(t, err)
					require.Len(t, block.TxResults, 1)
					result := block.TxResults[0]
					current := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
					stored, found := k.GetAllocation(current, id)
					valid := kind >= types.AllocationType_static && kind <= types.AllocationType_providerAgreement
					if !valid {
						t.Logf("type=%d: code=%d allocationExists=%t counter=%d->%d", kind, result.Code, found, beforeCount, k.GetAllocationCount(current))
						require.NotZero(t, result.Code, "a protobuf enum is not wire validation")
						require.Contains(t, result.Log, "invalid_type")
						require.NotContains(t, result.Log, "panic")
						require.False(t, found)
						require.Equal(t, beforeCount, k.GetAllocationCount(current))
						require.Equal(t, beforeGrid, k.GetAllGridExport(current))
						require.Empty(t, k.GetAllAllocationIdBySourceIndex(current, source))
					} else {
						require.Zero(t, result.Code, result.Log)
						require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
						require.True(t, found)
						require.Equal(t, types.Allocation{Id: id, Index: beforeCount, Type: kind, SourceObjectId: source, Controller: controller, Creator: address.String()}, stored)
						require.Equal(t, beforeCount+1, k.GetAllocationCount(current))
						require.Equal(t, []string{id}, k.GetAllAllocationIdBySourceIndex(current, source))
						power := uint64(100)
						if kind == types.AllocationType_automated {
							power = 1000
						}
						require.Equal(t, power, k.GetGridAttribute(current, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, id)))
						require.Equal(t, power, k.GetGridAttribute(current, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, source)))
						if !guildController {
							expectedPermissions[string(keeperlib.GetObjectPermissionIDBytes(id, recipient.Id))] = uint64(types.PermAllocationConnection | types.PermAdmin)
							if kind == types.AllocationType_dynamic {
								expectedPermissions[string(keeperlib.GetObjectPermissionIDBytes(id, actor.Id))] = uint64(types.PermUpdate | types.PermDelete)
							} else if kind == types.AllocationType_automated {
								expectedPermissions[string(keeperlib.GetObjectPermissionIDBytes(id, actor.Id))] = uint64(types.PermDelete)
							}
						}
					}
					auto, autoFound := k.GetAutoResizeAllocationBySource(current, source)
					require.Equal(t, valid && kind == types.AllocationType_automated, autoFound)
					if autoFound {
						require.Equal(t, id, auto)
					}
					require.Equal(t, expectedPermissions, permissionMap(k.GetAllPermissionExport(current)))
					require.Empty(t, k.GetAllGuildRankPermissionExport(current))
					events := 0
					for _, event := range result.Events {
						require.NotContains(t, []string{"coin_spent", "coin_received", "transfer", "structs.structs.EventDelete"}, event.Type)
						if !valid {
							require.NotContains(t, []string{"structs.structs.EventAllocation", "structs.structs.EventGrid", "structs.structs.EventPermission"}, event.Type)
						}
						if event.Type == "structs.structs.EventAllocation" {
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							require.Equal(t, stored, *parsed.(*types.EventAllocation).Allocation)
							events++
						}
					}
					if valid {
						require.Equal(t, 1, events)
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
					require.Equal(t, signer.GetSequence()+1, afterSigner.GetSequence())
				})
			}
		}
	}
}

// A constructor caller may commit its CurrentContext after handling an error.
// Transaction rollback alone would conceal a prematurely reserved id or cache.
func TestAllocationUnknownTypeConstructorLeavesNoWrites(t *testing.T) {
	for _, kind := range []types.AllocationType{-2147483648, -1, 4, 500, 2147483647} {
		t.Run(fmt.Sprint(int32(kind)), func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			k := bApp.StructsKeeper
			actor := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
			k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, actor.Id), 1000)
			// Existing malformed legacy state is not rewritten by a refused
			// creation. Repair/import policy is separate from new allocation input.
			k.SetAllocationOnly(ctx, types.Allocation{Id: "6-999", Index: 999, Type: 500, SourceObjectId: actor.Id, Controller: actor.Id})
			before, events := applicationAssetStateRows(bApp, ctx), ctx.EventManager().ABCIEvents()
			cc := k.NewCurrentContext(ctx)
			_, err := cc.NewAllocation(kind, actor.Id, "", actor.Creator, actor.Id, 100)
			require.ErrorContains(t, err, "invalid_type")
			cc.CommitAll()
			require.Equal(t, before, applicationAssetStateRows(bApp, ctx))
			require.Equal(t, events, ctx.EventManager().ABCIEvents())
			// Also continue in a context after a refused creation, before its
			// one permitted CommitAll, with no skipped id or phantom reservation.
			cc = k.NewCurrentContext(ctx)
			_, err = cc.NewAllocation(kind, actor.Id, "", actor.Creator, actor.Id, 100)
			require.ErrorContains(t, err, "invalid_type")
			count := k.GetAllocationCount(ctx)
			created, err := cc.NewAllocation(types.AllocationType_dynamic, actor.Id, "", actor.Creator, actor.Id, 100)
			require.NoError(t, err)
			cc.CommitAll()
			require.Equal(t, keeperlib.GetObjectID(types.ObjectType_allocation, count), created.ID())
			require.Equal(t, count+1, k.GetAllocationCount(ctx))
			require.Equal(t, uint64(100), k.GetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, actor.Id)))
			require.Equal(t, []string{created.ID()}, k.GetAllAllocationIdBySourceIndex(ctx, actor.Id))
		})
	}
}
