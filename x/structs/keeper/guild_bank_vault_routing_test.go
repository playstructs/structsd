package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestGuildConversionAndRedemptionDoNotSpendSigningAccount(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	f.fundAlpha(f.buyer, 1000)
	// Seed a valid 1:1 bank with a holder on a legacy signing account. These
	// balances must not be an implicit fallback for an empty protocol vault.
	_, err := bootstrapGuildTokens(t, f.k, f.ms, f.ctx, &types.MsgGuildBankMint{
		Creator: f.buyer.Creator, GuildId: f.guild.Id, AmountAlpha: 100, AmountToken: 100,
	})
	require.NoError(t, err)
	_, err = f.ms.GuildBankConvert(f.ctx, &types.MsgGuildBankConvert{
		Creator: f.buyer.Creator, GuildId: f.guild.Id, AmountAlpha: 10, MinAmountToken: 1,
	})
	// MockBankKeeper.HasBalance always returns true, so refusal comes from
	// its actual balance-moving method rather than the affordability precheck.
	require.ErrorContains(t, err, "insufficient funds")
	_, err = f.ms.GuildBankRedeem(f.ctx, &types.MsgGuildBankRedeem{
		Creator: f.buyer.Creator, AmountToken: sdk.NewInt64Coin("uguild."+f.guild.Id, 10), MinAmountAlpha: 1,
	})
	require.ErrorContains(t, err, "insufficient funds")
	require.Equal(t, int64(100), f.tokenBalance(f.buyer).Int64())
	primary, err := sdk.AccAddressFromBech32(f.buyer.PrimaryAddress)
	require.NoError(t, err)
	require.Equal(t, int64(900), f.k.BankKeeper().SpendableCoin(f.ctx, primary, "ualpha").Amount.Int64())
	require.Equal(t, int64(100), collateralBalance(f.k, f.ctx, f.guild.Id).Int64())
}
