package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

func TestMsgGuildBankRedeem(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player and guild
	playerAcc := sdk.AccAddress("creator123456789012345678901234567890")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	// Create reactor for guild
	validatorAddress := sdk.ValAddress(playerAcc.Bytes())
	reactor := types.Reactor{
		RawAddress: validatorAddress.Bytes(),
	}
	// AppendReactor already calls SetReactorValidatorBytes internally
	reactor = k.AppendReactor(ctx, reactor)

	// Create guild
	guild := k.AppendGuild(ctx, "test-endpoint", "", reactor, player, "")
	player.GuildId = guild.Id
	k.SetPlayer(ctx, player)

	vaultAddress, err := types.VaultAddress(player.Id)
	require.NoError(t, err)
	// Set up balances and mint some tokens first
	alphaCoin := sdk.NewCoin("ualpha", math.NewInt(1000))
	k.BankKeeper().MintCoins(ctx, types.ModuleName, sdk.NewCoins(alphaCoin))
	k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, vaultAddress, sdk.NewCoins(alphaCoin))

	_, err = bootstrapGuildVaultTokens(t, k, ms, wctx, &types.MsgGuildBankMint{
		Creator:     player.Creator,
		AmountAlpha: 100,
		AmountToken: 10,
	})
	require.NoError(t, err)

	primaryFunds := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 777))
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, primaryFunds))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, playerAcc, primaryFunds))
	denom := "uguild." + guild.Id
	guildCache := k.NewCurrentContext(ctx).GetGuild(guild.Id)
	require.True(t, guildCache.GetBankConvertOutFee().IsZero())
	collateral := guildCache.GetBankCollateralPool()
	balance := func(address sdk.AccAddress, denom string) math.Int {
		return k.BankKeeper().SpendableCoin(ctx, address, denom).Amount
	}

	testCases := []struct {
		name      string
		input     *types.MsgGuildBankRedeem
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid bank redeem",
			input: &types.MsgGuildBankRedeem{
				Creator:        player.Creator,
				AmountToken:    sdk.NewCoin("uguild."+guild.Id, math.NewInt(5)),
				MinAmountAlpha: 1,
			},
			expErr: false,
		},
		{
			name: "invalid denom format",
			input: &types.MsgGuildBankRedeem{
				Creator:        player.Creator,
				AmountToken:    sdk.NewCoin("invalid-denom", math.NewInt(5)),
				MinAmountAlpha: 1,
			},
			expErr:    true,
			expErrMsg: "invalid_format",
		},
		{
			name: "guild not found",
			input: &types.MsgGuildBankRedeem{
				Creator:        player.Creator,
				AmountToken:    sdk.NewCoin("uguild.0-999999", math.NewInt(5)),
				MinAmountAlpha: 1,
			},
			expErr:    true,
			expErrMsg: "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			alphaBefore := balance(vaultAddress, "ualpha")
			tokensBefore := balance(vaultAddress, denom)
			collateralBefore := balance(collateral, "ualpha")
			supplyBefore := k.BankKeeper().GetSupply(ctx, denom).Amount
			resp, err := ms.GuildBankRedeem(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.NotNil(t, resp)
				require.True(t, alphaBefore.Equal(balance(vaultAddress, "ualpha")))
				require.True(t, tokensBefore.Equal(balance(vaultAddress, denom)))
				require.True(t, collateralBefore.Equal(balance(collateral, "ualpha")))
				require.True(t, supplyBefore.Equal(k.BankKeeper().GetSupply(ctx, denom).Amount))
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, alphaBefore.AddRaw(50).Equal(balance(vaultAddress, "ualpha")))
				require.True(t, tokensBefore.SubRaw(5).Equal(balance(vaultAddress, denom)))
				require.True(t, collateralBefore.SubRaw(50).Equal(balance(collateral, "ualpha")))
				require.True(t, supplyBefore.SubRaw(5).Equal(k.BankKeeper().GetSupply(ctx, denom).Amount))
			}
			require.Equal(t, int64(777), balance(playerAcc, "ualpha").Int64())
			require.True(t, balance(playerAcc, denom).IsZero())
		})
	}
}
