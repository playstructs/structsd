package app_test

import (
	"bytes"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

func TestVaultBalanceMigrationMovesAllDenomsAndLeavesUnassociatedAccounts(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
	outsider := sdk.AccAddress(bytes.Repeat([]byte{0x31}, 20))
	guild := types.CreateEmptyGuild()
	guild.Id = "0-991"
	guild.Index = 991
	guild.Owner = playerID
	bApp.StructsKeeper.SetGuild(ctx, guild)
	keepers := &upgrades.Keepers{StructsKeeper: bApp.StructsKeeper}
	require.NoError(t, v22.MigrateProvisionVaults(ctx, keepers))
	vault, err := types.VaultAddress(playerID)
	require.NoError(t, err)
	denom := "uguild." + guild.Id
	first := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 10), sdk.NewInt64Coin(denom, 20))
	second := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 30), sdk.NewInt64Coin("otherasset", 40))
	existing := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 5))
	other := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 7))
	for _, funding := range []struct {
		address sdk.AccAddress
		coins   sdk.Coins
	}{{primary, first}, {secondary, second}, {vault, existing}, {outsider, other}} {
		require.NoError(t, bApp.BankKeeper.MintCoins(ctx, types.ModuleName, funding.coins))
		require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, funding.address, funding.coins))
	}
	// A freeze must not strand some denominations on old signing accounts.
	bApp.BankKeeper.SetSendEnabled(ctx, "otherasset", false)
	beforeSupply := map[string]sdk.Coin{}
	for _, d := range []string{"ualpha", denom, "otherasset"} {
		beforeSupply[d] = bApp.BankKeeper.GetSupply(ctx, d)
	}
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, v22.MigrateVaultBankBalances(ctx, keepers))
	// The real bank keeper must tell indexers about every denomination moved,
	// with both endpoints, rather than only changing balances in state.
	transfers := map[string]string{}
	spent := map[string]string{}
	credits := []string{}
	for _, event := range ctx.EventManager().Events() {
		attributes := map[string]string{}
		for _, attribute := range event.Attributes { attributes[attribute.Key] = attribute.Value }
		switch event.Type {
		case banktypes.EventTypeTransfer:
			if attributes[banktypes.AttributeKeyRecipient] == vault.String() { transfers[attributes[sdk.AttributeKeySender]] = attributes[sdk.AttributeKeyAmount] }
		case banktypes.EventTypeCoinSpent:
			spent[attributes[banktypes.AttributeKeySpender]] = attributes[sdk.AttributeKeyAmount]
		case banktypes.EventTypeCoinReceived:
			if attributes[banktypes.AttributeKeyReceiver] == vault.String() { credits = append(credits, attributes[sdk.AttributeKeyAmount]) }
		}
	}
	require.Equal(t, map[string]string{primary.String(): first.String(), secondary.String(): second.String()}, transfers)
	require.Equal(t, first.String(), spent[primary.String()])
	require.Equal(t, second.String(), spent[secondary.String()])
	require.ElementsMatch(t, []string{first.String(), second.String()}, credits)
	require.Equal(t, first.Add(second...).Add(existing...), bApp.BankKeeper.GetAllBalances(ctx, vault))
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, primary).IsZero())
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, secondary).IsZero())
	require.Equal(t, other, bApp.BankKeeper.GetAllBalances(ctx, outsider))
	guildVault, err := types.VaultAddress(guild.Id)
	require.NoError(t, err)
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, guildVault).IsZero(), "owner's personal assets are not guild treasury")
	for d, supply := range beforeSupply {
		require.Equal(t, supply, bApp.BankKeeper.GetSupply(ctx, d))
	}
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, v22.MigrateVaultBankBalances(ctx, keepers))
	require.Empty(t, ctx.EventManager().Events(), "rerun must not advertise duplicate asset movements")
	require.Equal(t, first.Add(second...).Add(existing...), bApp.BankKeeper.GetAllBalances(ctx, vault))
}

func TestVaultBalanceMigrationLeavesNativeVestingIntact(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
	keepers := &upgrades.Keepers{StructsKeeper: bApp.StructsKeeper, StakingKeeper: bApp.StakingKeeper}
	require.NoError(t, v22.MigrateProvisionVaults(ctx, keepers))
	funds := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 10))
	for _, address := range []sdk.AccAddress{primary, secondary} {
		require.NoError(t, bApp.BankKeeper.MintCoins(ctx, types.ModuleName, funds))
		require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, address, funds))
	}
	account := bApp.AccountKeeper.GetAccount(ctx, secondary)
	base := authtypes.NewBaseAccount(account.GetAddress(), account.GetPubKey(), account.GetAccountNumber(), account.GetSequence())
	vesting, err := vestingtypes.NewDelayedVestingAccount(base, funds, ctx.BlockTime().Add(time.Hour).Unix())
	require.NoError(t, err)
	bApp.AccountKeeper.SetAccount(ctx, vesting)
	require.NoError(t, v22.MigrateVaultBankBalances(ctx, keepers))
	require.Equal(t, funds, bApp.BankKeeper.GetAllBalances(ctx, secondary))
	require.Equal(t, vesting, bApp.AccountKeeper.GetAccount(ctx, secondary))
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, primary).IsZero())
	vault, err := types.VaultAddress(playerID)
	require.NoError(t, err)
	require.Equal(t, funds, bApp.BankKeeper.GetAllBalances(ctx, vault))
}
