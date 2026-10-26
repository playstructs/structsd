package v0_22_0_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

func TestVaultProvisioningMigrationIsDeterministicAndDoesNotMoveAssets(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}
	address := sdk.AccAddress("old-player-address!!")
	k.SetPlayer(ctx, types.Player{Id: "1-2", Index: 2, PrimaryAddress: address.String()})
	k.SetPlayer(ctx, types.Player{Id: "1-1", Index: 1})
	k.SetGuild(ctx, types.Guild{Id: "0-1", Index: 1, Owner: "1-2"})
	coins := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 2000000))
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, address, coins))
	require.NoError(t, v22.MigrateProvisionVaults(ctx, keepers))
	var previous uint64
	numbers := map[string]uint64{}
	for i, id := range []string{"0-1", "1-1", "1-2"} {
		vault, err := types.VaultAddress(id)
		require.NoError(t, err)
		owner, found := k.GetVaultOwner(ctx, vault)
		require.True(t, found)
		require.Equal(t, id, owner)
		number := k.AccountKeeper().GetAccount(ctx, vault).GetAccountNumber()
		numbers[id] = number
		if i > 0 {
			require.Less(t, previous, number)
		}
		previous = number
		require.True(t, k.BankKeeper().SpendableCoins(ctx, vault).IsZero())
	}
	require.Equal(t, coins, k.BankKeeper().SpendableCoins(ctx, address))
	require.NoError(t, v22.MigrateProvisionVaults(ctx, keepers))
	for id, number := range numbers {
		vault, _ := types.VaultAddress(id)
		require.Equal(t, number, k.AccountKeeper().GetAccount(ctx, vault).GetAccountNumber())
	}
}
