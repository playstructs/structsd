package app_test

import (
	"math/rand"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestSignedGuildOwnershipTransferChangesVaultAuthorityWithinBatch(t *testing.T) {
	for _, scenario := range []string{"new_owner_spends", "old_owner_refused", "restricted_new_owner"} {
		t.Run(scenario, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			oldKey, newKey := secp256k1.GenPrivKey(), secp256k1.GenPrivKey()
			oldAddress, newAddress := sdk.AccAddress(oldKey.PubKey().Address()), sdk.AccAddress(newKey.PubKey().Address())
			oldOwner := appendAppPlayer(t, bApp.StructsKeeper, ctx, oldAddress)
			newOwner := appendAppPlayer(t, bApp.StructsKeeper, ctx, newAddress)
			require.Empty(t, newOwner.GuildId, "the recipient need not be a member to own the guild")
			guild := types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner, guild.Creator = "0-916", 916, oldOwner.Id, oldAddress.String()
			bApp.StructsKeeper.SetGuild(ctx, guild)
			vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(guild.Id)
			require.NoError(t, err)
			treasury := vaults[0]
			accountNumber := bApp.AccountKeeper.GetAccount(ctx, treasury).GetAccountNumber()
			fundRedelegationAccount(t, bApp, ctx, treasury, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100)))
			recipient, err := types.VaultAddress(newOwner.Id)
			require.NoError(t, err)
			spender := newAddress.String()
			keys := []cryptotypes.PrivKey{oldKey, newKey}
			if scenario == "old_owner_refused" {
				spender = oldAddress.String()
				keys = keys[:1]
			}
			if scenario == "restricted_new_owner" {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(newAddress.String()), types.PermAll&^types.PermTokenTransfer)
			}
			messages := []sdk.Msg{
				&types.MsgGuildUpdateOwnerId{Creator: oldAddress.String(), GuildId: guild.Id, Owner: newOwner.Id},
				&types.MsgVaultSend{Creator: spender, OwnerId: guild.Id, RecipientOwnerId: newOwner.Id, Amount: sdk.NewCoins(sdk.NewInt64Coin("ualpha", 7))},
			}
			var numbers, sequences []uint64
			for _, key := range keys {
				account := bApp.AccountKeeper.GetAccount(ctx, sdk.AccAddress(key.PubKey().Address()))
				numbers = append(numbers, account.GetAccountNumber())
				sequences = append(sequences, account.GetSequence())
			}
			config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil,
				ante.DefaultFreeGasCap, jailGateChainID, numbers, sequences, keys...)
			require.NoError(t, err)
			wire, err := config.TxEncoder()(tx)
			require.NoError(t, err)
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
			require.NoError(t, err)
			require.Len(t, block.TxResults, 1)
			result := block.TxResults[0]
			finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
			actual, found := bApp.StructsKeeper.GetGuild(finalCtx, guild.Id)
			require.True(t, found)
			if scenario == "new_owner_spends" {
				require.Zero(t, result.Code, result.Log)
				require.Equal(t, newOwner.Id, actual.Owner)
				require.Equal(t, int64(93), bApp.BankKeeper.GetBalance(finalCtx, treasury, "ualpha").Amount.Int64())
				require.Equal(t, int64(7), bApp.BankKeeper.GetBalance(finalCtx, recipient, "ualpha").Amount.Int64())
			} else {
				require.NotZero(t, result.Code)
				if scenario == "old_owner_refused" {
					require.Contains(t, result.Log, "message index: 1", "ownership transfer must execute before the old owner's spend is refused")
				} else {
					require.Contains(t, result.Log, "address lacks required permission bits", "the restricted key must be refused in ante")
				}
				require.Contains(t, result.Log, "permission")
				require.Equal(t, guild, actual, "failed spend must roll back the preceding ownership transfer")
				require.Equal(t, int64(100), bApp.BankKeeper.GetBalance(finalCtx, treasury, "ualpha").Amount.Int64())
				require.Nil(t, bApp.AccountKeeper.GetAccount(finalCtx, recipient))
				_, provisioned := bApp.StructsKeeper.GetVaultOwner(finalCtx, recipient)
				require.False(t, provisioned)
				for _, id := range []string{oldOwner.Id, newOwner.Id} {
					require.Equal(t, types.Permissionless, bApp.StructsKeeper.GetPermissionsByBytes(finalCtx, keeperlib.GetObjectPermissionIDBytes(guild.Id, id)),
						"failed transfer must not retain ownership permission-row edits")
				}
				for _, event := range result.Events {
					require.NotEqual(t, "transfer", event.Type)
					require.NotEqual(t, "structs.structs.EventGuild", event.Type)
					require.NotEqual(t, "structs.structs.EventVaultAddress", event.Type)
				}
			}
			ownerID, found := bApp.StructsKeeper.GetVaultOwner(finalCtx, treasury)
			require.True(t, found)
			require.Equal(t, guild.Id, ownerID, "the vault remains associated with the guild, not its human owner")
			require.Equal(t, accountNumber, bApp.AccountKeeper.GetAccount(finalCtx, treasury).GetAccountNumber())
			require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(finalCtx, treasury.String()))
		})
	}
}

func TestSignedGuildOwnershipTransferChangesChildPermissionAuthorityWithinBatch(t *testing.T) {
	for _, scenario := range []string{"new_owner_sets", "old_owner_refused", "new_owner_cannot_erase_unheld_bit"} {
		t.Run(scenario, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			oldKey, newKey := secp256k1.GenPrivKey(), secp256k1.GenPrivKey()
			oldAddress, newAddress := sdk.AccAddress(oldKey.PubKey().Address()), sdk.AccAddress(newKey.PubKey().Address())
			oldOwner := appendAppPlayer(t, bApp.StructsKeeper, ctx, oldAddress)
			newOwner := appendAppPlayer(t, bApp.StructsKeeper, ctx, newAddress)
			target := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress("child_grant_target01"))
			guild := types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner, guild.Creator = "0-917", 917, oldOwner.Id, oldAddress.String()
			bApp.StructsKeeper.SetGuild(ctx, guild)
			allocation, err := bApp.StructsKeeper.SetAllocationOnly(ctx, types.Allocation{Id: "6-917", Controller: guild.Id})
			require.NoError(t, err)
			targetID := keeperlib.GetObjectPermissionIDBytes(allocation.Id, target.Id)
			bApp.StructsKeeper.SetPermissionsByBytes(ctx, targetID, types.PermUpdate|types.PermDelete)
			creator := newAddress.String()
			keys := []cryptotypes.PrivKey{oldKey, newKey}
			if scenario == "old_owner_refused" {
				creator, keys = oldAddress.String(), keys[:1]
			}
			if scenario == "new_owner_cannot_erase_unheld_bit" {
				bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(newAddress.String()), types.PermAll&^types.PermDelete)
			}
			before := bApp.StructsKeeper.GetAllPermissionExport(ctx)
			messages := []sdk.Msg{
				&types.MsgGuildUpdateOwnerId{Creator: oldAddress.String(), GuildId: guild.Id, Owner: newOwner.Id},
				&types.MsgPermissionSetOnObject{Creator: creator, ObjectId: allocation.Id, PlayerId: target.Id, Permissions: uint64(types.PermUpdate)},
			}
			var numbers, sequences []uint64
			for _, key := range keys {
				account := bApp.AccountKeeper.GetAccount(ctx, sdk.AccAddress(key.PubKey().Address()))
				numbers = append(numbers, account.GetAccountNumber())
				sequences = append(sequences, account.GetSequence())
			}
			config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, messages, nil,
				ante.DefaultFreeGasCap, jailGateChainID, numbers, sequences, keys...)
			require.NoError(t, err)
			wire, err := config.TxEncoder()(tx)
			require.NoError(t, err)
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
			require.NoError(t, err)
			require.Len(t, block.TxResults, 1)
			result := block.TxResults[0]
			finalCtx := bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
			actual, found := bApp.StructsKeeper.GetGuild(finalCtx, guild.Id)
			require.True(t, found)
			storedAllocation, found := bApp.StructsKeeper.GetAllocation(finalCtx, allocation.Id)
			require.True(t, found)
			require.Equal(t, allocation, storedAllocation, "permission edits cannot change child ownership")
			if scenario == "new_owner_sets" {
				require.Zero(t, result.Code, result.Log)
				require.Equal(t, newOwner.Id, actual.Owner)
				require.Equal(t, types.PermUpdate, bApp.StructsKeeper.GetPermissionsByBytes(finalCtx, targetID))
				updates := 0
				for _, event := range result.Events {
					if event.Type != "structs.structs.EventPermission" {
						continue
					}
					decoded, err := sdk.ParseTypedEvent(event)
					require.NoError(t, err)
					record := decoded.(*types.EventPermission).PermissionRecord
					if record.PermissionId == string(targetID) {
						require.Equal(t, uint64(types.PermUpdate), record.Value)
						updates++
					}
				}
				require.Equal(t, 1, updates)
			} else {
				require.NotZero(t, result.Code)
				require.Contains(t, result.Log, "message index: 1", "the ownership transfer must execute before child permission refusal")
				require.Contains(t, result.Log, "permission")
				require.Equal(t, guild, actual, "failed child permission edit rolls back ownership transfer")
				require.Equal(t, before, bApp.StructsKeeper.GetAllPermissionExport(finalCtx))
				for _, event := range result.Events {
					require.NotEqual(t, "structs.structs.EventPermission", event.Type)
					require.NotEqual(t, "structs.structs.EventGuild", event.Type)
				}
			}
		})
	}
}
