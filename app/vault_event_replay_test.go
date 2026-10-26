package app_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestVaultEventsAdoptExistingAccountAndReplayWithoutProvisioning(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	_, _, playerID := newTransferPlayer(t, bApp, ctx)
	guild := types.CreateEmptyGuild()
	guild.Id, guild.Index, guild.Owner = "0-993", 993, playerID
	bApp.StructsKeeper.SetGuild(ctx, guild)
	vault, err := types.VaultAddress(playerID)
	require.NoError(t, err)
	account := bApp.AccountKeeper.NewAccountWithAddress(ctx, vault)
	bApp.AccountKeeper.SetAccount(ctx, account)
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	_, err = bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(playerID, guild.Id)
	require.NoError(t, err)
	provisioned := migrationEvents[*types.EventVaultAddress](t, ctx, "EventVaultAddress")
	require.Len(t, provisioned, 2)
	require.True(t, provisioned[0].EventVaultAddressDetail.AccountCreated)
	require.Equal(t, guild.Id, provisioned[0].EventVaultAddressDetail.OwnerId)
	require.False(t, provisioned[1].EventVaultAddressDetail.AccountCreated)
	require.Equal(t, account.GetAccountNumber(), provisioned[1].EventVaultAddressDetail.AccountNumber)
	ctx = ctx.WithBlockHeight(1).WithEventManager(sdk.NewEventManager())
	bApp.StructsKeeper.EventAllGenesis(ctx)
	replayed := migrationEvents[*types.EventVaultAddress](t, ctx, "EventVaultAddress")
	require.Len(t, replayed, 2)
	for i, event := range replayed {
		detail := event.EventVaultAddressDetail
		original := provisioned[i].EventVaultAddressDetail
		require.Equal(t, original.OwnerId, detail.OwnerId)
		require.Equal(t, original.VaultAddress, detail.VaultAddress)
		require.Equal(t, original.AccountNumber, detail.AccountNumber)
		require.False(t, detail.AccountCreated)
		require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, detail.VaultAddress))
	}
	for _, event := range migrationEvents[*types.EventAddressAssociation](t, ctx, "EventAddressAssociation") {
		for _, vaultEvent := range replayed {
			require.NotEqual(t, vaultEvent.EventVaultAddressDetail.VaultAddress, event.AddressAssociation.Address)
		}
	}
	ctx = ctx.WithBlockHeight(2).WithEventManager(sdk.NewEventManager())
	bApp.StructsKeeper.EventAllGenesis(ctx)
	require.Empty(t, ctx.EventManager().Events(), "full replay must remain restricted to block one")
}
