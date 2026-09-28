package app_test

import (
	"math/rand"
	"testing"
	"time"

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

// Rank-derived authority must stop authorizing actual bank debits as soon as
// membership, rank or its permission grant changes, including within one tx.
func TestSignedGuildRankLossStopsTreasurySpending(t *testing.T) {
	for _, route := range []string{"vault_send", "player_send"} {
		for _, transition := range []string{"departure", "kick", "demotion", "rank_revoke"} {
			t.Run(route+"/"+transition, func(t *testing.T) {
				bApp, ctx := setupReactorAppGenesis(t, 1, true)
				k := bApp.StructsKeeper
				actorKey, ownerKey := secp256k1.GenPrivKey(), secp256k1.GenPrivKey()
				actorAddress, ownerAddress := sdk.AccAddress(actorKey.PubKey().Address()), sdk.AccAddress(ownerKey.PubKey().Address())
				actor := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
				owner := appendAppPlayer(t, k, ctx, ownerAddress)
				peer := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
				require.NoError(t, k.SetPlayerIndexForAddress(ctx, actorAddress.String(), actor.Index))
				k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(actorAddress.String()), types.PermTokenTransfer|types.PermGuildMembership)
				guild, destination := types.CreateEmptyGuild(), types.CreateEmptyGuild()
				guild.Id, guild.Index, guild.Owner = "0-994", 994, owner.Id
				guild.EntryRank, guild.RankPurchaseMinimumRank, guild.RankPurchaseBasePrice = 5, 1, 1
				destination.Id, destination.Index, destination.Owner = "0-995", 995, owner.Id
				destination.EntryRank, destination.RankPurchaseMinimumRank, destination.JoinByRankPurchaseEnabled = 5, 1, true
				k.SetGuild(ctx, guild)
				k.SetGuild(ctx, destination)
				actor.GuildId, actor.GuildRank = guild.Id, 5
				k.SetPlayer(ctx, actor)
				k.SetGuildRankPermissionStoreOnly(ctx, guild.Id, guild.Id, types.PermTokenTransfer, 3)
				vaults, err := k.NewCurrentContext(ctx).EnsureVaults(actor.Id, guild.Id)
				require.NoError(t, err)
				buyer, treasury := vaults[0], vaults[1]
				denom := "uguild." + guild.Id
				fundRedelegationAccount(t, bApp, ctx, buyer, sdk.NewCoins(sdk.NewInt64Coin(denom, 10)))
				fundRedelegationAccount(t, bApp, ctx, treasury, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100)))
				recipient, err := types.VaultAddress(peer.Id)
				require.NoError(t, err)
				if route == "player_send" {
					recipient, err = sdk.AccAddressFromBech32(peer.PrimaryAddress)
					require.NoError(t, err)
				}
				send := func(creator string) sdk.Msg {
					amount := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 7))
					if route == "player_send" {
						return &types.MsgPlayerSend{Creator: creator, FromAddress: treasury.String(), ToAddress: recipient.String(), Amount: amount}
					}
					return &types.MsgVaultSend{Creator: creator, OwnerId: guild.Id, RecipientOwnerId: peer.Id, Amount: amount}
				}
				config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
				current, height := ctx, int64(0)
				deliver := func(keys []cryptotypes.PrivKey, messages ...sdk.Msg) *abci.ExecTxResult {
					t.Helper()
					var numbers, sequences []uint64
					for _, key := range keys {
						account := bApp.AccountKeeper.GetAccount(current, sdk.AccAddress(key.PubKey().Address()))
						numbers = append(numbers, account.GetAccountNumber())
						sequences = append(sequences, account.GetSequence())
					}
					if height > 0 {
						_, err := bApp.Commit()
						require.NoError(t, err)
					}
					height++
					tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(height)), config, messages, nil,
						ante.DefaultFreeGasCap, jailGateChainID, numbers, sequences, keys...)
					require.NoError(t, err)
					wire, err := config.TxEncoder()(tx)
					require.NoError(t, err)
					when := ctx.BlockTime().Add(time.Duration(height-1) * time.Second)
					block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: when, Txs: [][]byte{wire}})
					require.NoError(t, err)
					require.Len(t, block.TxResults, 1)
					result := block.TxResults[0]
					require.NotContains(t, result.Log, "panic")
					require.LessOrEqual(t, result.GasUsed, int64(ante.DefaultFreeGasCap))
					current = bApp.NewContext(false).WithBlockTime(when)
					return result
				}
				actorKeys, ownerKeys := []cryptotypes.PrivKey{actorKey}, []cryptotypes.PrivKey{ownerKey}
				assertFunds := func(treasuryAmount, recipientAmount int64) {
					t.Helper()
					require.Equal(t, treasuryAmount, bApp.BankKeeper.GetBalance(current, treasury, "ualpha").Amount.Int64())
					require.Equal(t, recipientAmount, bApp.BankKeeper.GetBalance(current, recipient, "ualpha").Amount.Int64())
				}
				assertRefused := func(result *abci.ExecTxResult) {
					t.Helper()
					require.NotZero(t, result.Code)
					require.Contains(t, result.Log, "permission")
					for _, event := range result.Events {
						require.NotContains(t, event.Type, "structs.structs.Event", "failed messages cannot publish rolled-back state")
						require.NotContains(t, []string{"transfer", "coin_spent", "coin_received"}, event.Type)
					}
				}
				// A capable signing key at entry rank still has no treasury grant.
				assertRefused(deliver(actorKeys, send(actorAddress.String())))
				assertFunds(100, 0)
				result := deliver(actorKeys,
					&types.MsgGuildRankPurchase{Creator: actorAddress.String(), GuildId: guild.Id, ExpectedRank: 5, TargetRank: 3, MaxPrice: 2},
					send(actorAddress.String()))
				require.Zero(t, result.Code, result.Log)
				assertFunds(93, 7)
				require.Equal(t, int64(8), bApp.BankKeeper.GetBalance(current, buyer, denom).Amount.Int64())
				require.Equal(t, int64(2), bApp.BankKeeper.GetBalance(current, treasury, denom).Amount.Int64())
				var change sdk.Msg
				changeKeys, batchKeys := ownerKeys, []cryptotypes.PrivKey{ownerKey, actorKey}
				switch transition {
				case "departure":
					change = &types.MsgGuildMembershipJoinByRankPurchase{Creator: actorAddress.String(), GuildId: destination.Id, ExpectedGuildId: guild.Id, TargetRank: 5}
					changeKeys, batchKeys = actorKeys, actorKeys
				case "kick":
					change = &types.MsgGuildMembershipKick{Creator: ownerAddress.String(), GuildId: guild.Id, PlayerId: actor.Id}
				case "demotion":
					change = &types.MsgPlayerUpdateGuildRank{Creator: ownerAddress.String(), GuildId: guild.Id, PlayerId: actor.Id, GuildRank: 4}
				case "rank_revoke":
					change = &types.MsgPermissionGuildRankRevoke{Creator: ownerAddress.String(), GuildId: guild.Id, ObjectId: guild.Id, Permission: uint64(types.PermTokenTransfer)}
				}
				beforePlayer, found := k.GetPlayer(current, actor.Id)
				require.True(t, found)
				beforeRanks, beforePermissions := k.GetAllGuildRankPermissionExport(current), k.GetAllPermissionExport(current)
				result = deliver(batchKeys, change, send(actorAddress.String()))
				assertRefused(result)
				require.Contains(t, result.Log, "message index: 1", "authority loss must execute before spending is refused")
				assertFunds(93, 7)
				afterPlayer, found := k.GetPlayer(current, actor.Id)
				require.True(t, found)
				require.Equal(t, beforePlayer, afterPlayer, "the failed batch must roll back the preceding membership/rank transition")
				require.Equal(t, beforeRanks, k.GetAllGuildRankPermissionExport(current))
				require.Equal(t, beforePermissions, k.GetAllPermissionExport(current))
				result = deliver(changeKeys, change)
				require.Zero(t, result.Code, result.Log)
				assertRefused(deliver(actorKeys, send(actorAddress.String())))
				assertFunds(93, 7)
				// The current owner still controls the same treasury: refusal
				// above is authority loss, not a frozen account or failed bank path.
				result = deliver(ownerKeys, send(ownerAddress.String()))
				require.Zero(t, result.Code, result.Log)
				assertFunds(86, 14)
				ownerID, associated := k.GetVaultOwner(current, treasury)
				require.True(t, associated)
				require.Equal(t, guild.Id, ownerID)
				require.Zero(t, k.GetPlayerIndexFromAddress(current, treasury.String()))
				require.Equal(t, int64(8), bApp.BankKeeper.GetBalance(current, buyer, denom).Amount.Int64())
				require.Equal(t, int64(2), bApp.BankKeeper.GetBalance(current, treasury, denom).Amount.Int64())
				_, err = bApp.Commit()
				require.NoError(t, err)
			})
		}
	}
}
