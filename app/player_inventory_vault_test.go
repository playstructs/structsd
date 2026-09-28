package app_test

import (
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestPlayerInventoryReadsVaultWithoutProvisioning(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	primary := sdk.AccAddress("inventory-primary-01")
	player := appendAppPlayer(t, bApp.StructsKeeper, ctx, primary)
	fundRedelegationAccount(t, bApp, ctx, primary, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 73)))
	vault, err := types.VaultAddress(player.Id)
	require.NoError(t, err)
	require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, vault))

	check := func(want int64) {
		t.Helper()
		beforeAccount := bApp.AccountKeeper.GetAccount(ctx, vault)
		beforeEvents := len(ctx.EventManager().Events())
		result, err := bApp.StructsKeeper.Player(ctx, &types.QueryGetPlayerRequest{Id: player.Id})
		require.NoError(t, err)
		require.Equal(t, sdk.NewInt64Coin("ualpha", want), result.PlayerInventory.Rocks)
		require.Equal(t, primary.String(), result.Player.PrimaryAddress)
		require.Equal(t, int64(73), bApp.BankKeeper.GetBalance(ctx, primary, "ualpha").Amount.Int64())
		require.Equal(t, beforeAccount, bApp.AccountKeeper.GetAccount(ctx, vault))
		require.Len(t, ctx.EventManager().Events(), beforeEvents)
		_, found := bApp.StructsKeeper.GetVaultOwner(ctx, vault)
		require.False(t, found, "read-only inventory cannot provision a vault association")
		require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, vault.String()))
	}
	check(0)
	fundRedelegationAccount(t, bApp, ctx, vault, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100)))
	check(100)
	// Inventory must retain the existing spendability contract, not expose
	// locked vault funds as if the player could spend them.
	base := bApp.AccountKeeper.GetAccount(ctx, vault).(*authtypes.BaseAccount)
	locked, err := vestingtypes.NewDelayedVestingAccount(base, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 60)), ctx.BlockTime().Add(time.Hour).Unix())
	require.NoError(t, err)
	bApp.AccountKeeper.SetAccount(ctx, locked)
	check(40)
}
