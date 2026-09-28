package app_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestRankPurchaseRealBankFreezePaymentAndReplay(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x71}, 20)))
	buyer := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x72}, 20)))
	guild := types.CreateEmptyGuild()
	guild.Id, guild.Index, guild.Owner, guild.Creator = "0-912", 912, owner.Id, owner.Creator
	guild.EntryRank = 5
	bApp.StructsKeeper.SetGuild(ctx, guild)
	buyer.GuildId, buyer.GuildRank = guild.Id, 5
	bApp.StructsKeeper.SetPlayer(ctx, buyer)
	_, err := ms.GuildUpdateRankPurchase(ctx, &types.MsgGuildUpdateRankPurchase{
		Creator: owner.Creator, GuildId: guild.Id, BasePrice: 10, StepIncrease: 2, MinimumRank: 1,
	})
	require.NoError(t, err)
	vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(buyer.Id)
	require.NoError(t, err)
	denom := "uguild." + guild.Id
	coins := sdk.NewCoins(sdk.NewInt64Coin(denom, 100))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, vaults[0], coins))
	treasury, err := types.VaultAddress(guild.Id)
	require.NoError(t, err)
	msg := &types.MsgGuildRankPurchase{Creator: buyer.Creator, GuildId: guild.Id, ExpectedRank: 5, TargetRank: 3, MaxPrice: 22}
	assertRank := func(rank uint64) {
		player, found := bApp.StructsKeeper.GetPlayer(ctx, buyer.Id)
		require.True(t, found)
		require.Equal(t, rank, player.GuildRank)
	}
	bApp.BankKeeper.SetSendEnabled(ctx, denom, false)
	eventCount := len(ctx.EventManager().Events())
	_, err = ms.GuildRankPurchase(ctx, msg)
	require.Error(t, err)
	assertRank(5)
	require.Equal(t, int64(100), bApp.BankKeeper.GetBalance(ctx, vaults[0], denom).Amount.Int64())
	require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, treasury))
	require.Len(t, ctx.EventManager().Events(), eventCount)
	bApp.BankKeeper.SetSendEnabled(ctx, denom, true)
	response, err := ms.GuildRankPurchase(ctx, msg)
	require.NoError(t, err)
	require.Equal(t, uint64(22), response.Price)
	assertRank(3)
	require.Equal(t, int64(78), bApp.BankKeeper.GetBalance(ctx, vaults[0], denom).Amount.Int64())
	require.Equal(t, int64(22), bApp.BankKeeper.GetBalance(ctx, treasury, denom).Amount.Int64())
	require.True(t, bApp.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(owner.Creator), denom).IsZero())
	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, treasury.String()))
	eventCount = len(ctx.EventManager().Events())
	_, err = ms.GuildRankPurchase(ctx, msg)
	require.Error(t, err, "the old rank quote must not be paid twice")
	assertRank(3)
	require.Equal(t, int64(22), bApp.BankKeeper.GetBalance(ctx, treasury, denom).Amount.Int64())
	require.Len(t, ctx.EventManager().Events(), eventCount)
}
