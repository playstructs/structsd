package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestGuildProviderDeletePaysConsumersBeforeVaultRemainder(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")
	// Exercise settlement of an existing guild-owned record through the handler.
	guild := types.Guild{Id: "0-900", Owner: f.provider.Owner}
	f.k.SetGuild(f.ctx, guild)
	f.provider.Owner = guild.Id
	_, err := f.k.SetProvider(f.ctx, f.provider)
	require.NoError(t, err)
	agreement, collateral := f.openAgreement(t, 100, 50)
	f.fund(t, f.collateralAcc, 7)
	f.fund(t, f.earningsAcc, 3)
	vault, err := types.VaultAddress(guild.Id)
	require.NoError(t, err)
	keyBefore := f.balance(f.ownerAcc)
	_, err = f.ms.ProviderDelete(f.ctx, &types.MsgProviderDelete{Creator: f.provider.Creator, ProviderId: f.provider.Id})
	require.NoError(t, err)
	require.Equal(t, collateral, f.balance(f.consumerAcc))
	require.Equal(t, math.NewInt(10), f.balance(vault))
	require.Equal(t, keyBefore, f.balance(f.ownerAcc))
	require.True(t, f.balance(f.collateralAcc).IsZero())
	require.True(t, f.balance(f.earningsAcc).IsZero())
	owner, found := f.k.GetVaultOwner(f.ctx, vault)
	require.True(t, found)
	require.Equal(t, guild.Id, owner)
	f.requireSettledOnce(t, agreement)
}

func TestProviderWithdrawalDefaultsToAssetOwnerVault(t *testing.T) {
	for _, guildOwned := range []bool{false, true} {
		name := "player"
		if guildOwned {
			name = "guild"
		}
		t.Run(name, func(t *testing.T) {
			f := setupTeardownFixture(t, 10, "0", "0")
			if guildOwned {
				guild := types.Guild{Id: "0-900", Owner: f.provider.Owner}
				f.k.SetGuild(f.ctx, guild)
				f.provider.Owner = guild.Id
				_, err := f.k.SetProvider(f.ctx, f.provider)
				require.NoError(t, err)
			}
			agreement, collateral := f.openAgreement(t, 100, 50)
			f.fund(t, f.earningsAcc, 3)
			f.ctx = sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(int64(agreement.StartBlock + 2))
			vault, err := types.VaultAddress(f.provider.Owner)
			require.NoError(t, err)
			keyBefore := f.balance(f.ownerAcc)
			msg := &types.MsgProviderWithdrawBalance{Creator: f.provider.Creator, ProviderId: f.provider.Id}
			testPermissionRemove(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermProviderWithdraw)
			_, err = f.ms.ProviderWithdrawBalance(f.ctx, msg)
			require.Error(t, err)
			require.True(t, f.balance(vault).IsZero())
			require.Equal(t, collateral, f.balance(f.collateralAcc))
			testPermissionAdd(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermProviderWithdraw)
			_, err = f.ms.ProviderWithdrawBalance(f.ctx, msg)
			require.NoError(t, err)
			require.Equal(t, math.NewInt(2003), f.balance(vault))
			require.Equal(t, collateral.SubRaw(2000), f.balance(f.collateralAcc), "unearned consumer collateral remains reserved")
			require.True(t, f.balance(f.earningsAcc).IsZero())
			require.Equal(t, keyBefore, f.balance(f.ownerAcc))
			_, err = f.ms.ProviderWithdrawBalance(f.ctx, msg)
			require.NoError(t, err)
			require.Equal(t, math.NewInt(2003), f.balance(vault), "same-block replay pays no additional revenue")
		})
	}
}
