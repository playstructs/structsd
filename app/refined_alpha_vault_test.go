package app_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestRefinedAlphaCreditsPlayerVaultWithRealBank(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	primary := sdk.AccAddress(bytes.Repeat([]byte{0x66}, 20))
	player := appendAppPlayer(t, bApp.StructsKeeper, ctx, primary)
	address, err := types.VaultAddress(player.Id)
	require.NoError(t, err)
	require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, address))
	for i := int64(1); i <= 2; i++ {
		cc := bApp.StructsKeeper.NewCurrentContext(ctx)
		owner, err := cc.GetExistingPlayer(player.Id)
		require.NoError(t, err)
		require.NoError(t, owner.DepositRefinedAlpha())
		cc.CommitAll()
		require.Equal(t, i*1000000, bApp.BankKeeper.GetBalance(ctx, address, "ualpha").Amount.Int64())
		require.Equal(t, i*1000000, bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount.Int64())
	}
	require.True(t, bApp.BankKeeper.GetBalance(ctx, primary, "ualpha").IsZero())
	account := bApp.AccountKeeper.GetAccount(ctx, address)
	require.NotNil(t, account)
	require.Nil(t, account.GetPubKey())
	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, address.String()))
	owner, indexed := bApp.StructsKeeper.GetVaultOwner(ctx, address)
	require.True(t, indexed)
	require.Equal(t, player.Id, owner)
}
