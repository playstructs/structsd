package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestVaultSendPlayerFundsGuildAndOwnerDistributesTreasury(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	fundRankVault(t, f, f.member, 100)
	coins := sdk.NewCoins(sdk.NewInt64Coin("uguild."+f.guild.Id, 40))
	_, err := f.ms.VaultSend(f.ctx, &types.MsgVaultSend{Creator: f.member.Creator, OwnerId: f.member.Id, RecipientOwnerId: f.guild.Id, Amount: coins})
	require.NoError(t, err)
	require.Equal(t, int64(60), rankVaultBalance(t, f, f.member.Id))
	require.Equal(t, int64(40), rankVaultBalance(t, f, f.guild.Id))
	// Guild owner authority works independently of membership. The recipient's
	// vault is provisioned by this send and never registered as a signing key.
	_, err = f.ms.VaultSend(f.ctx, &types.MsgVaultSend{Creator: f.buyer.Creator, OwnerId: f.guild.Id, RecipientOwnerId: f.buyer.Id, Amount: coins})
	require.NoError(t, err)
	require.Zero(t, rankVaultBalance(t, f, f.guild.Id))
	require.Equal(t, int64(40), rankVaultBalance(t, f, f.buyer.Id))
	addr, err := types.VaultAddress(f.buyer.Id)
	require.NoError(t, err)
	require.Zero(t, f.k.GetPlayerIndexFromAddress(f.ctx, addr.String()))
	require.True(t, f.tokenBalance(f.buyer).IsZero())
}

func TestVaultSendRejectsUnauthorizedAndInvalidTransfers(t *testing.T) {
	for _, scenario := range []string{"other_player", "guild_member", "weak_key", "missing_owner", "missing_recipient", "wrong_namespace", "empty", "negative", "duplicate_denom", "insufficient"} {
		t.Run(scenario, func(t *testing.T) {
			f := newNonMemberOwnerFixture(t)
			fundRankVault(t, f, f.member, 100)
			msg := &types.MsgVaultSend{Creator: f.member.Creator, OwnerId: f.member.Id, RecipientOwnerId: f.guild.Id, Amount: sdk.NewCoins(sdk.NewInt64Coin("uguild."+f.guild.Id, 40))}
			switch scenario {
			case "other_player":
				msg.Creator = f.buyer.Creator
			case "guild_member":
				msg.OwnerId = f.guild.Id
			case "weak_key":
				cc := f.k.NewCurrentContext(f.ctx)
				cc.ClearPermissions(keeperlib.GetAddressPermissionIDBytes(f.member.Creator))
				cc.CommitAll()
			case "missing_owner":
				msg.OwnerId = "1-99999999"
			case "missing_recipient":
				msg.RecipientOwnerId = "0-99999999"
			case "wrong_namespace":
				msg.RecipientOwnerId = "3-1"
			case "empty":
				msg.Amount = nil
			case "negative":
				msg.Amount[0].Amount = msg.Amount[0].Amount.Neg()
			case "duplicate_denom":
				msg.Amount = append(msg.Amount, msg.Amount[0])
			case "insufficient":
				msg.Amount[0].Amount = msg.Amount[0].Amount.MulRaw(100)
			}
			_, err := f.ms.VaultSend(f.ctx, msg)
			require.Error(t, err)
			require.Equal(t, int64(100), rankVaultBalance(t, f, f.member.Id))
			require.Zero(t, rankVaultBalance(t, f, f.guild.Id))
		})
	}
}

func TestGuildVaultRankAuthorityTracksDemotionAndSigningKeyCeiling(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	fundRankVault(t, f, f.member, 100)
	coins := func(n int64) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin("uguild."+f.guild.Id, n)) }
	_, err := f.ms.VaultSend(f.ctx, &types.MsgVaultSend{Creator: f.member.Creator, OwnerId: f.member.Id, RecipientOwnerId: f.guild.Id, Amount: coins(100)})
	require.NoError(t, err)
	rank := f.member.GuildRank
	_, err = f.ms.PermissionGuildRankSet(f.ctx, &types.MsgPermissionGuildRankSet{Creator: f.buyer.Creator, ObjectId: f.guild.Id, GuildId: f.guild.Id, Permission: uint64(types.PermTokenTransfer), Rank: rank})
	require.NoError(t, err)
	spend := &types.MsgVaultSend{Creator: f.member.Creator, OwnerId: f.guild.Id, RecipientOwnerId: f.member.Id, Amount: coins(10)}
	_, err = f.ms.VaultSend(f.ctx, spend)
	require.NoError(t, err)
	require.Equal(t, int64(90), rankVaultBalance(t, f, f.guild.Id))
	// The current non-member owner can demote a treasury delegate. A fresh
	// transaction must resolve current rank rather than retain the old grant.
	_, err = f.ms.PlayerUpdateGuildRank(f.ctx, &types.MsgPlayerUpdateGuildRank{Creator: f.buyer.Creator, GuildId: f.guild.Id, PlayerId: f.member.Id, GuildRank: rank + 1})
	require.NoError(t, err)
	_, err = f.ms.VaultSend(f.ctx, spend)
	require.Error(t, err)
	require.Equal(t, int64(90), rankVaultBalance(t, f, f.guild.Id))
	require.Equal(t, int64(10), rankVaultBalance(t, f, f.member.Id))
	_, err = f.ms.PlayerUpdateGuildRank(f.ctx, &types.MsgPlayerUpdateGuildRank{Creator: f.buyer.Creator, GuildId: f.guild.Id, PlayerId: f.member.Id, GuildRank: rank})
	require.NoError(t, err)
	testPermissionRemove(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(f.member.Creator), types.PermTokenTransfer)
	_, err = f.ms.VaultSend(f.ctx, spend)
	require.Error(t, err, "restored guild rank cannot exceed the signing key's asset bits")
	require.Equal(t, int64(90), rankVaultBalance(t, f, f.guild.Id))
	testPermissionAdd(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(f.member.Creator), types.PermTokenTransfer)
	_, err = f.ms.VaultSend(f.ctx, spend)
	require.NoError(t, err)
	require.Equal(t, int64(80), rankVaultBalance(t, f, f.guild.Id))
	// Removing the rank grant revokes spending without walking child objects or
	// changing the vault's owner association.
	_, err = f.ms.PermissionGuildRankRevoke(f.ctx, &types.MsgPermissionGuildRankRevoke{Creator: f.buyer.Creator, ObjectId: f.guild.Id, GuildId: f.guild.Id, Permission: uint64(types.PermTokenTransfer)})
	require.NoError(t, err)
	_, err = f.ms.VaultSend(f.ctx, spend)
	require.Error(t, err)
	require.Equal(t, int64(80), rankVaultBalance(t, f, f.guild.Id))
}

func TestGuildVaultRankAuthorityEndsWithMembership(t *testing.T) {
	for _, departure := range []string{"kick", "purchase_join_another_guild"} {
		t.Run(departure, func(t *testing.T) {
			f := newNonMemberOwnerFixture(t)
			fundRankVault(t, f, f.member, 100)
			coins := func(n int64) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin("uguild."+f.guild.Id, n)) }
			_, err := f.ms.VaultSend(f.ctx, &types.MsgVaultSend{Creator: f.member.Creator, OwnerId: f.member.Id, RecipientOwnerId: f.guild.Id, Amount: coins(100)})
			require.NoError(t, err)
			_, err = f.ms.PermissionGuildRankSet(f.ctx, &types.MsgPermissionGuildRankSet{Creator: f.buyer.Creator, ObjectId: f.guild.Id, GuildId: f.guild.Id, Permission: uint64(types.PermTokenTransfer), Rank: f.member.GuildRank})
			require.NoError(t, err)
			spend := &types.MsgVaultSend{Creator: f.member.Creator, OwnerId: f.guild.Id, RecipientOwnerId: f.member.Id, Amount: coins(10)}
			_, err = f.ms.VaultSend(f.ctx, spend)
			require.NoError(t, err, "the rank grant must authorize spending before departure")
			expectedGuild := ""
			if departure == "kick" {
				_, err = f.ms.GuildMembershipKick(f.ctx, &types.MsgGuildMembershipKick{Creator: f.buyer.Creator, GuildId: f.guild.Id, PlayerId: f.member.Id})
				require.NoError(t, err)
			} else {
				destination := testCreateGuild(f.k, f.ctx)
				expectedGuild = destination.Guild.Id
				_, err = f.ms.GuildUpdateRankPurchase(f.ctx, &types.MsgGuildUpdateRankPurchase{Creator: destination.GuildOwner.Creator, GuildId: expectedGuild, BasePrice: 10, MinimumRank: 2, JoinByRankPurchaseEnabled: true})
				require.NoError(t, err)
				_, err = f.ms.GuildMembershipJoinByRankPurchase(f.ctx, &types.MsgGuildMembershipJoinByRankPurchase{Creator: f.member.Creator, GuildId: expectedGuild, ExpectedGuildId: f.guild.Id, TargetRank: destination.Guild.EntryRank})
				require.NoError(t, err)
			}
			player, found := f.k.GetPlayer(f.ctx, f.member.Id)
			require.True(t, found)
			require.Equal(t, expectedGuild, player.GuildId)
			_, err = f.ms.VaultSend(f.ctx, spend)
			require.Error(t, err, "the old guild rank must not authorize a former member")
			require.Equal(t, int64(90), rankVaultBalance(t, f, f.guild.Id))
			require.Equal(t, int64(10), rankVaultBalance(t, f, f.member.Id))
			// Rejoining at entry rank must not resurrect the departed rank's
			// authority. Keep the old rank grant installed throughout this test.
			configureRankSales(t, f, true)
			_, err = f.ms.GuildMembershipJoinByRankPurchase(f.ctx, &types.MsgGuildMembershipJoinByRankPurchase{Creator: f.member.Creator, GuildId: f.guild.Id, ExpectedGuildId: expectedGuild, TargetRank: f.guild.EntryRank})
			require.NoError(t, err)
			_, err = f.ms.VaultSend(f.ctx, spend)
			require.Error(t, err, "entry-rank admission must not restore the prior treasury rank")
			require.Equal(t, int64(90), rankVaultBalance(t, f, f.guild.Id))
			// The installed grant still works after an explicit promotion; the
			// denial above must come from current membership/rank resolution.
			_, err = f.ms.PlayerUpdateGuildRank(f.ctx, &types.MsgPlayerUpdateGuildRank{Creator: f.buyer.Creator, GuildId: f.guild.Id, PlayerId: f.member.Id, GuildRank: f.member.GuildRank})
			require.NoError(t, err)
			_, err = f.ms.VaultSend(f.ctx, spend)
			require.NoError(t, err)
			require.Equal(t, int64(80), rankVaultBalance(t, f, f.guild.Id))
		})
	}
}
