package app_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestVaultSendRealBankProvisioningAndRollback(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x61}, 20)))
	peer := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(bytes.Repeat([]byte{0x62}, 20)))
	guild := types.CreateEmptyGuild()
	guild.Id = "0-911"
	guild.Index = 911
	guild.Owner = owner.Id
	guild.Creator = owner.Creator
	bApp.StructsKeeper.SetGuild(ctx, guild)
	source, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(owner.Id)
	require.NoError(t, err)
	denom := "uguild." + guild.Id
	coins := sdk.NewCoins(sdk.NewInt64Coin(denom, 100))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, source[0], coins))
	guildAddress, err := types.VaultAddress(guild.Id)
	require.NoError(t, err)
	require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, guildAddress))
	msg := &types.MsgVaultSend{Creator: owner.Creator, OwnerId: owner.Id, RecipientOwnerId: guild.Id, Amount: coins}
	bApp.BankKeeper.SetSendEnabled(ctx, denom, false)
	_, err = ms.VaultSend(ctx, msg)
	require.Error(t, err, "vault transfers must honor governance denom freezes")
	require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, guildAddress))
	require.Equal(t, coins[0], bApp.BankKeeper.GetBalance(ctx, source[0], denom))
	bApp.BankKeeper.SetSendEnabled(ctx, denom, true)
	_, err = ms.VaultSend(ctx, msg)
	require.NoError(t, err)
	require.Equal(t, coins[0], bApp.BankKeeper.GetBalance(ctx, guildAddress, denom))
	require.NotNil(t, bApp.AccountKeeper.GetAccount(ctx, guildAddress))
	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, guildAddress.String()))
	// A bank failure occurs after recipient provisioning. Discard the SDK message
	// cache, as transaction execution does on error, and verify both stores.
	peerAddress, err := types.VaultAddress(peer.Id)
	require.NoError(t, err)
	failedCtx, _ := ctx.CacheContext()
	msg.OwnerId = guild.Id
	msg.RecipientOwnerId = peer.Id
	msg.Amount = sdk.NewCoins(sdk.NewInt64Coin(denom, 101))
	_, err = ms.VaultSend(failedCtx, msg)
	require.Error(t, err)
	require.NotNil(t, bApp.AccountKeeper.GetAccount(failedCtx, peerAddress), "failure must exercise post-provisioning rollback")
	require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, peerAddress))
	_, indexed := bApp.StructsKeeper.GetVaultOwner(ctx, peerAddress)
	require.False(t, indexed)
	require.Equal(t, coins[0], bApp.BankKeeper.GetBalance(ctx, guildAddress, denom))
	msg.Amount = coins
	_, err = ms.VaultSend(ctx, msg)
	require.NoError(t, err)
	require.Equal(t, coins[0], bApp.BankKeeper.GetBalance(ctx, peerAddress, denom))
	require.True(t, bApp.BankKeeper.GetBalance(ctx, guildAddress, denom).IsZero())
}
