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

// Drive transfer through ante and message delivery with a secondary Admin key.
// Ownership changes cannot erase unrelated grants or preserve implicit authority
// in a later message. Independent grants deliberately remain a separate policy.
func TestSignedAllocationTransferAuthority(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		for _, guildRecipient := range []bool{false, true} {
			for _, scenario := range []string{"owner", "direct", "rank", "demoted", "departed", "wrong_key", "no_authority", "missing_player", "missing_guild", "malformed_owner", "wrong_owner_type", "same_controller", "former_owner_second_message", "later_missing_allocation"} {
				t.Run(fmt.Sprintf("guild=%t/recipientGuild=%t/%s", guildOwned, guildRecipient, scenario), func(t *testing.T) {
					bApp, ctx := setupJailGateApp(t)
					k := bApp.StructsKeeper
					newPlayer := func() types.Player {
						return appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
					}
					actor, owner, recipient, source := newPlayer(), newPlayer(), newPlayer(), newPlayer()
					key := secp256k1.GenPrivKey()
					address := sdk.AccAddress(key.PubKey().Address())
					require.NoError(t, k.SetPlayerIndexForAddress(ctx, address.String(), actor.Index))
					implicitOwner := scenario == "owner" || scenario == "wrong_key" || scenario == "same_controller" || scenario == "former_owner_second_message" || scenario == "later_missing_allocation" || strings.Contains(scenario, "missing_") || scenario == "malformed_owner" || scenario == "wrong_owner_type"
					if implicitOwner {
						owner = actor
					}
					guild := types.CreateEmptyGuild()
					guild.Id, guild.Index, guild.Owner, guild.Creator = "0-958", 958, owner.Id, owner.Creator
					k.SetGuild(ctx, guild)
					otherGuild := types.CreateEmptyGuild()
					otherGuild.Id, otherGuild.Index, otherGuild.Owner, otherGuild.Creator = "0-959", 959, recipient.Id, recipient.Creator
					k.SetGuild(ctx, otherGuild)
					controller, target := owner.Id, recipient.Id
					if guildOwned {
						controller = guild.Id
					}
					if guildRecipient {
						target = otherGuild.Id
					}
					const destination = "4-958"
					k.SetSubstation(ctx, types.Substation{Id: destination, Owner: source.Id})
					k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, source.Id), 1000)
					cc := k.NewCurrentContext(ctx)
					allocation, err := cc.NewAllocation(types.AllocationType_dynamic, source.Id, destination, source.Creator, controller, 100)
					require.NoError(t, err)
					id := allocation.ID()
					cc.CommitAll()
					retained := types.PermUpdate | types.PermDelete
					if !guildOwned {
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(id, controller), retained|types.PermAllocationConnection)
					}
					k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(id, recipient.Id), retained)
					bits, refused := types.PermAdmin, false
					switch scenario {
					case "direct":
						k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(id, actor.Id), types.PermAdmin)
					case "rank", "demoted", "departed":
                        refused = true // Administration cannot come from rank.
						actor.GuildId, actor.GuildRank = guild.Id, 3
						if scenario == "demoted" {
							actor.GuildRank, refused = 4, true
						}
						if scenario == "departed" {
							actor.GuildId, refused = "", true
						}
						k.SetPlayer(ctx, actor)
						k.SetGuildRankPermissionStoreOnly(ctx, id, guild.Id, types.PermAdmin, 3)
					case "wrong_key":
						bits, refused = types.PermAllocationConnection, true
					case "no_authority":
						refused = true
					case "missing_player":
						target, refused = "1-999999", true
					case "missing_guild":
						target, refused = "0-999999", true
					case "malformed_owner":
						target, refused = "0-0959", true
					case "wrong_owner_type":
						target, refused = destination, true
					case "same_controller":
						target = controller
					case "former_owner_second_message", "later_missing_allocation":
						refused = true
					}
					k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(address.String()), bits)
					messages := []sdk.Msg{&types.MsgAllocationTransfer{Creator: address.String(), AllocationId: id, Controller: target}}
					if scenario == "former_owner_second_message" {
						messages = append(messages, &types.MsgAllocationTransfer{Creator: address.String(), AllocationId: id, Controller: controller})
					}
					if scenario == "later_missing_allocation" {
						messages = append(messages, &types.MsgAllocationTransfer{Creator: address.String(), AllocationId: "6-999999", Controller: controller})
					}
					before, found := k.GetAllocation(ctx, id)
					require.True(t, found)
					beforeGrid, beforeCount := k.GetAllGridExport(ctx), k.GetAllocationCount(ctx)
					beforePermissions := k.GetAllPermissionExport(ctx)
					beforeRanks := k.ReadGuildRankRegister(ctx, id, guild.Id)
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
					stored, found := k.GetAllocation(current, id)
					require.True(t, found)
					if refused {
						require.NotZero(t, result.Code, result.Log)
						require.NotContains(t, result.Log, "panic")
						if len(messages) == 2 {
							require.Contains(t, result.Log, "message index: 1")
						}
						require.Equal(t, before, stored)
						require.Equal(t, beforePermissions, k.GetAllPermissionExport(current))
					} else {
						require.Zero(t, result.Code, result.Log)
						require.Less(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
						before.Controller = target
						require.Equal(t, before, stored)
						// Compare every permission row, not only the recipient's row.
						expected := map[string]types.Permission{}
						for _, row := range beforePermissions {
							expected[row.PermissionId] = types.Permission(row.Value)
						}
						if !guildOwned {
							expected[string(keeperlib.GetObjectPermissionIDBytes(id, controller))] &^= types.PermAllocationConnection
						}
						if strings.HasPrefix(target, "1-") {
							expected[string(keeperlib.GetObjectPermissionIDBytes(id, target))] |= types.PermAllocationConnection
						}
						actual := map[string]types.Permission{}
						for _, row := range k.GetAllPermissionExport(current) {
							actual[row.PermissionId] = types.Permission(row.Value)
						}
						require.Equal(t, expected, actual)
					}
					require.Equal(t, beforeGrid, k.GetAllGridExport(current))
					require.Equal(t, beforeCount, k.GetAllocationCount(current))
					require.Equal(t, beforeRanks, k.ReadGuildRankRegister(current, id, guild.Id))
					require.Equal(t, []string{id}, k.GetAllAllocationIdBySourceIndex(current, source.Id))
					require.Equal(t, []string{id}, k.GetAllAllocationIdByDestinationIndex(current, destination))
					transfers := 0
					for _, event := range result.Events {
						require.NotContains(t, []string{"coin_spent", "coin_received", "transfer"}, event.Type)
						if refused {
							require.NotContains(t, []string{"structs.structs.EventAllocation", "structs.structs.EventPermission"}, event.Type)
						}
						if event.Type == "structs.structs.EventAllocation" {
							parsed, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							require.Equal(t, stored, *parsed.(*types.EventAllocation).Allocation)
							transfers++
						}
					}
					if !refused {
						require.Equal(t, 1, transfers)
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
					sequence := signer.GetSequence() + 1
					if scenario == "wrong_key" {
						sequence = signer.GetSequence() // Fixed Admin gate refuses before message execution.
					}
					require.Equal(t, sequence, afterSigner.GetSequence())
				})
			}
		}
	}
}
