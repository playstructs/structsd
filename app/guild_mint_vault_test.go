package app_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestGuildMintRealBankUsesGuildVault(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	ownerAddress := sdk.AccAddress(bytes.Repeat([]byte{0x65}, 20))
	owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, ownerAddress)
	guild := types.CreateEmptyGuild()
	guild.Id = "0-912"
	guild.Index = 912
	guild.Owner = owner.Id
	guild.Creator = owner.Creator
	bApp.StructsKeeper.SetGuild(ctx, guild)
	vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(guild.Id)
	require.NoError(t, err)
	alpha := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, types.ModuleName, alpha))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, vaults[0], alpha))
	_, err = ms.GuildBankMint(ctx, &types.MsgGuildBankMint{Creator: owner.Creator, GuildId: guild.Id, AmountAlpha: 60, AmountToken: 10})
	require.NoError(t, err)
	require.Equal(t, int64(40), bApp.BankKeeper.GetBalance(ctx, vaults[0], "ualpha").Amount.Int64())
	require.Equal(t, int64(10), bApp.BankKeeper.GetBalance(ctx, vaults[0], "uguild."+guild.Id).Amount.Int64())
	pool := bApp.StructsKeeper.NewCurrentContext(ctx).GetGuild(guild.Id).GetBankCollateralPool()
	require.Equal(t, int64(60), bApp.BankKeeper.GetBalance(ctx, pool, "ualpha").Amount.Int64())
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, ownerAddress).IsZero())
	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, vaults[0].String()))
}
