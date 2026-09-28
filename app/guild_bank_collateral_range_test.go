package app_test

import (
	"bytes"
	"math/big"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// A pool aggregates many uint64 transactions and also accepts ordinary bank
// deposits. Its SDK balance can exceed uint64 while an individual quote fits.
func TestGuildBankWideCollateralKeepsRepresentableQuotesLive(t *testing.T) {
	for _, scenario := range []string{"donation", "mint", "convert", "already_wide_convert", "fee", "max_quote", "quote_one_above", "quote_out_of_range"} {
		t.Run(scenario, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			address := sdk.AccAddress(bytes.Repeat([]byte{0x79}, 20))
			owner := appendAppPlayer(t, bApp.StructsKeeper, ctx, address)
			guild := types.CreateEmptyGuild()
			guild.Id, guild.Index, guild.Owner, guild.Creator = "0-983", 983, owner.Id, owner.Creator
			if scenario == "fee" {
				guild.BankConvertOutFee = math.LegacyMustNewDecFromStr("0.1")
			}
			bApp.StructsKeeper.SetGuild(ctx, guild)
			_, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(owner.Id, guild.Id)
			require.NoError(t, err)
			vault, err := types.VaultAddress(owner.Id)
			require.NoError(t, err)
			treasury, err := types.VaultAddress(guild.Id)
			require.NoError(t, err)
			pool := bApp.StructsKeeper.NewCurrentContext(ctx).GetGuild(guild.Id).GetBankCollateralPool()
			denom := "uguild." + guild.Id
			max := math.NewIntFromUint64(^uint64(0))
			collateral, supply := max.SubRaw(10), math.NewInt(1000)
			amountAlpha := math.NewInt(100000000000000000)
			if scenario == "already_wide_convert" || scenario == "fee" {
				collateral = max.AddRaw(100)
			}
			if scenario == "max_quote" || scenario == "quote_one_above" {
				collateral, supply = max.MulRaw(2), math.NewInt(2)
				if scenario == "quote_one_above" {
					collateral = collateral.AddRaw(2)
				}
			}
			if scenario == "quote_out_of_range" {
				// Leave space below the SDK's total-supply limit for the fixture's
				// other alpha. Multiplying this by max uint64 needs 320 bits.
				collateral = math.NewIntFromBigInt(new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(10001)))
				supply, amountAlpha = max, math.NewInt(1000)
			}
			fundRedelegationAccount(t, bApp, ctx, address, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 200)))
			fundRedelegationAccount(t, bApp, ctx, vault, sdk.NewCoins(sdk.NewCoin("ualpha", amountAlpha), sdk.NewCoin(denom, supply)))
			fundRedelegationAccount(t, bApp, ctx, treasury, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100)))
			fundRedelegationAccount(t, bApp, ctx, pool, sdk.NewCoins(sdk.NewCoin("ualpha", collateral)))
			alphaSupply := bApp.BankKeeper.GetSupply(ctx, "ualpha")
			ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
			switch scenario {
			case "donation":
				require.NoError(t, bApp.BankKeeper.SendCoins(ctx, address, pool, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 100))))
				collateral = collateral.AddRaw(100)
			case "mint":
				_, err = ms.GuildBankMint(ctx, &types.MsgGuildBankMint{Creator: owner.Creator, GuildId: guild.Id, AmountAlpha: 100, AmountToken: 1})
				require.NoError(t, err)
				collateral, supply = collateral.AddRaw(100), supply.AddRaw(1)
				require.Equal(t, math.OneInt(), bApp.BankKeeper.GetBalance(ctx, treasury, denom).Amount)
			case "convert", "already_wide_convert":
				// floor(10^17 * 1000 / ~2^64) = 5; no floating-point quote.
				_, err = ms.GuildBankConvert(ctx, &types.MsgGuildBankConvert{Creator: owner.Creator, GuildId: guild.Id, AmountAlpha: amountAlpha.Uint64(), MinAmountToken: 5})
				require.NoError(t, err)
				collateral, supply = collateral.Add(amountAlpha), supply.AddRaw(5)
				require.Equal(t, supply, bApp.BankKeeper.GetBalance(ctx, vault, denom).Amount)
			}
			require.True(t, collateral.GT(max), "exercise an aggregate balance outside uint64")
			require.Equal(t, collateral, bApp.BankKeeper.GetBalance(ctx, pool, "ualpha").Amount)
			beforeVault := bApp.BankKeeper.GetAllBalances(ctx, vault)
			beforePool := bApp.BankKeeper.GetAllBalances(ctx, pool)
			module := bApp.AccountKeeper.GetModuleAddress(types.ModuleName)
			beforeModule := bApp.BankKeeper.GetAllBalances(ctx, module)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			amountToken := math.OneInt()
			if scenario == "fee" {
				amountToken = math.NewInt(7)
			} else if scenario == "quote_out_of_range" {
				amountToken = supply
			}
			require.NotPanics(t, func() {
				_, err = ms.GuildBankRedeem(ctx, &types.MsgGuildBankRedeem{Creator: owner.Creator, AmountToken: sdk.NewCoin(denom, amountToken), MinAmountAlpha: 1})
			})
			if scenario == "quote_out_of_range" || scenario == "quote_one_above" {
				require.ErrorContains(t, err, "amountAlpha")
				require.ErrorContains(t, err, "out_of_range")
				require.Equal(t, beforeVault, bApp.BankKeeper.GetAllBalances(ctx, vault))
				require.Equal(t, beforePool, bApp.BankKeeper.GetAllBalances(ctx, pool))
				require.Equal(t, supply, bApp.BankKeeper.GetSupply(ctx, denom).Amount)
				for _, event := range ctx.EventManager().Events() {
					require.NotContains(t, []string{"burn", "transfer", "structs.structs.EventGuildBankRedeem"}, event.Type)
				}
			} else {
				require.NoError(t, err)
				gross := collateral.Mul(amountToken).Quo(supply)
				fee := math.ZeroInt()
				if scenario == "fee" {
					fee = gross.AddRaw(9).QuoRaw(10)
				}
				paid := gross.Sub(fee)
				require.Equal(t, beforeVault.AmountOf("ualpha").Add(paid), bApp.BankKeeper.GetBalance(ctx, vault, "ualpha").Amount)
				require.Equal(t, beforeVault.AmountOf(denom).Sub(amountToken), bApp.BankKeeper.GetBalance(ctx, vault, denom).Amount)
				require.Equal(t, collateral.Sub(paid), bApp.BankKeeper.GetBalance(ctx, pool, "ualpha").Amount)
				require.Equal(t, supply.Sub(amountToken), bApp.BankKeeper.GetSupply(ctx, denom).Amount)
				redemptions := 0
				for _, event := range ctx.EventManager().ABCIEvents() {
					if event.Type == "structs.structs.EventGuildBankRedeem" {
						decoded, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						require.Equal(t, &types.EventGuildBankRedeemDetail{GuildId: guild.Id, AmountAlpha: paid.Uint64(), AmountToken: amountToken.Uint64(), PlayerId: owner.Id, Fee: fee.Uint64()}, decoded.(*types.EventGuildBankRedeem).EventGuildBankRedeemDetail)
						redemptions++
					}
				}
				require.Equal(t, 1, redemptions)
			}
			require.Equal(t, beforeModule, bApp.BankKeeper.GetAllBalances(ctx, module))
			require.Equal(t, alphaSupply, bApp.BankKeeper.GetSupply(ctx, "ualpha"))
		})
	}
}
