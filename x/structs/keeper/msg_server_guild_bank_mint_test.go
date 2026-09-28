package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildBankMint(t *testing.T) {
	for _, scenario := range []string{"success", "unregistered_signer", "no_mint_permission"} {
		t.Run(scenario, func(t *testing.T) {
			k, ms, goCtx := setupMsgServer(t)
			ctx := sdk.UnwrapSDKContext(goCtx)
			primary := sdk.AccAddress("mint-table-primary-01")
			player := testAppendPlayer(k, ctx, types.Player{Creator: primary.String(), PrimaryAddress: primary.String()})
			reactor := k.AppendReactor(ctx, types.Reactor{RawAddress: sdk.ValAddress(primary.Bytes()).Bytes()})
			guild := k.AppendGuild(ctx, "test-endpoint", "", reactor, player, "")
			player.GuildId = guild.Id
			k.SetPlayer(ctx, player)
			vaults, err := k.NewCurrentContext(ctx).EnsureVaults(guild.Id)
			require.NoError(t, err)
			fundAlpha(t, k, ctx, vaults[0], 1000)
			fundAlpha(t, k, ctx, primary, 777)
			msg := &types.MsgGuildBankMint{Creator: player.Creator, AmountAlpha: 100, AmountToken: 10}
			if scenario == "unregistered_signer" {
				msg.Creator = sdk.AccAddress("mint-table-unknown-01").String()
				msg.GuildId = guild.Id
			}
			if scenario == "no_mint_permission" {
				// Guild ownership cannot lift the signing key's own ceiling.
				testPermissionRemove(k, ctx, keeperlib.GetAddressPermissionIDBytes(player.Creator), types.PermGuildTokenMint)
			}
			indexBefore := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
			collateralBefore := collateralBalance(k, ctx, guild.Id)
			denom := "uguild." + guild.Id
			_, err = ms.GuildBankMint(ctx, msg)
			wantAlpha, wantTokens := int64(1000), int64(0)
			if scenario == "success" {
				require.NoError(t, err)
				wantAlpha, wantTokens = 900, 10
				require.True(t, collateralBefore.AddRaw(100).Equal(collateralBalance(k, ctx, guild.Id)))
			} else {
				require.Error(t, err)
				if scenario == "no_mint_permission" {
					requireDeniedOnSignerBit(t, err, msg.Creator, types.PermGuildTokenMint)
				}
				require.True(t, collateralBefore.Equal(collateralBalance(k, ctx, guild.Id)))
			}
			require.Equal(t, wantAlpha, k.BankKeeper().SpendableCoin(ctx, vaults[0], "ualpha").Amount.Int64())
			require.Equal(t, wantTokens, k.BankKeeper().SpendableCoin(ctx, vaults[0], denom).Amount.Int64())
			require.Equal(t, int64(777), k.BankKeeper().SpendableCoin(ctx, primary, "ualpha").Amount.Int64())
			require.True(t, k.BankKeeper().SpendableCoin(ctx, primary, denom).IsZero())
			require.Equal(t, indexBefore, k.GetPlayerIndexFromAddress(ctx, msg.Creator), "mint cannot implicitly register a caller")
		})
	}
}

func TestGuildBankMintRequiresTokenTransferOnSigningKey(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	primaryAcc := sdk.AccAddress("mintprimary123456789012345678901234")
	player := testAppendPlayer(k, ctx, types.Player{
		Creator:        primaryAcc.String(),
		PrimaryAddress: primaryAcc.String(),
	})
	reactor := k.AppendReactor(ctx, types.Reactor{RawAddress: sdk.ValAddress(primaryAcc.Bytes()).Bytes()})
	guild := k.AppendGuild(ctx, "test-endpoint", "", reactor, player, "")

	player.GuildId = guild.Id
	k.SetPlayer(ctx, player)

	weakAcc := sdk.AccAddress("mintweak123456789012345678901234567")
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, weakAcc.String(), player.Index))
	k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(weakAcc.String()), types.PermGuildTokenMint)

	alpha := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(100)))
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, alpha))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, primaryAcc, alpha))

	_, err := ms.GuildBankMint(ctx, &types.MsgGuildBankMint{
		Creator:     weakAcc.String(),
		GuildId:     guild.Id,
		AmountAlpha: 10,
		AmountToken: 10,
	})
	requireDeniedOnSignerBit(t, err, weakAcc.String(), types.PermTokenTransfer)
	require.Equal(t, math.NewInt(100), k.BankKeeper().SpendableCoin(ctx, primaryAcc, "ualpha").Amount)
}

func TestGuildMintUsesTreasuryAndKeepsCollateralSeparate(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	f.fundAlpha(f.buyer, 1000)
	msg := &types.MsgGuildBankMint{Creator: f.buyer.Creator, GuildId: f.guild.Id, AmountAlpha: 100, AmountToken: 10}
	// Owning personal alpha cannot fund an unfunded treasury implicitly.
	_, err := f.ms.GuildBankMint(f.ctx, msg)
	require.Error(t, err)
	vaults, err := f.k.NewCurrentContext(f.ctx).EnsureVaults(f.guild.Id)
	require.NoError(t, err)
	fundAlpha(t, f.k, f.ctx, vaults[0], 150)
	_, err = f.ms.GuildBankMint(f.ctx, msg)
	require.NoError(t, err)
	require.Equal(t, int64(50), f.k.BankKeeper().SpendableCoin(f.ctx, vaults[0], "ualpha").Amount.Int64())
	require.Equal(t, int64(100), collateralBalance(f.k, f.ctx, f.guild.Id).Int64())
	require.Equal(t, int64(10), rankVaultBalance(t, f, f.guild.Id))
	require.True(t, f.tokenBalance(f.buyer).IsZero())
	personal, err := sdk.AccAddressFromBech32(f.buyer.PrimaryAddress)
	require.NoError(t, err)
	require.Equal(t, int64(1000), f.k.BankKeeper().SpendableCoin(f.ctx, personal, "ualpha").Amount.Int64())
}

func TestGuildMintPermissionDoesNotAuthorizeTreasurySpend(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	vaults, err := f.k.NewCurrentContext(f.ctx).EnsureVaults(f.guild.Id)
	require.NoError(t, err)
	fundAlpha(t, f.k, f.ctx, vaults[0], 100)
	testPermissionAdd(f.k, f.ctx, keeperlib.GetObjectPermissionIDBytes(f.guild.Id, f.member.Id), types.PermGuildTokenMint)
	_, err = f.ms.GuildBankMint(f.ctx, &types.MsgGuildBankMint{Creator: f.member.Creator, GuildId: f.guild.Id, AmountAlpha: 100, AmountToken: 10})
	require.Error(t, err)
	require.Equal(t, int64(100), f.k.BankKeeper().SpendableCoin(f.ctx, vaults[0], "ualpha").Amount.Int64())
	require.Zero(t, rankVaultBalance(t, f, f.guild.Id))
}
