package app_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
)

func TestVaultOwnershipInvariantAndExportRejectSigningOrMissingAccounts(t *testing.T) {
	for _, corruption := range []string{"missing_account", "public_key", "signing_index"} {
		t.Run(corruption, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			_, _, playerID := newTransferPlayer(t, bApp, ctx)
			check := keeperlib.VaultOwnershipConsistencyInvariant(bApp.StructsKeeper)
			_, broken := check(ctx)
			require.False(t, broken, "owners need not have provisioned vaults before migration")
			vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(playerID)
			require.NoError(t, err)
			vault := vaults[0]
			owners, err := bApp.StructsKeeper.GetVaultOwnersForExport(ctx)
			require.NoError(t, err)
			require.Contains(t, owners, playerID)
			_, broken = check(ctx)
			require.False(t, broken)
			account := bApp.AccountKeeper.GetAccount(ctx, vault)
			switch corruption {
			case "missing_account":
				bApp.AccountKeeper.RemoveAccount(ctx, account)
			case "public_key":
				require.NoError(t, account.SetPubKey(ed25519.GenPrivKey().PubKey()))
				bApp.AccountKeeper.SetAccount(ctx, account)
			case "signing_index":
				player, found := bApp.StructsKeeper.GetPlayer(ctx, playerID)
				require.True(t, found)
				require.NoError(t, bApp.StructsKeeper.SetPlayerIndexForAddress(ctx, vault.String(), player.Index))
			}
			_, err = bApp.StructsKeeper.GetVaultOwnersForExport(ctx)
			require.Error(t, err)
			require.Contains(t, err.Error(), playerID)
			message, broken := check(ctx)
			require.True(t, broken)
			require.Contains(t, message, playerID)
		})
	}
}
