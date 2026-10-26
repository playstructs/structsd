package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
)

func TestGuildCreationPaymentBurnsAndCollateralizesExactlyOnce(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	vaults, err := f.k.NewCurrentContext(f.ctx).EnsureVaults(f.buyer.Id)
	require.NoError(t, err)
	fundAlpha(t, f.k, f.ctx, vaults[0], 3000000)
	supplyBefore := f.k.BankKeeper().GetSupply(f.ctx, "ualpha").Amount
	cc := f.k.NewCurrentContext(f.ctx)
	founder, err := cc.GetSigningPlayer(f.buyer.Creator)
	require.NoError(t, err)
	guild := cc.GetGuild(f.guild.Id)
	require.NoError(t, guild.FundCreation(founder))
	cc.CommitAll()
	require.Equal(t, int64(1000000), f.k.BankKeeper().SpendableCoin(f.ctx, vaults[0], "ualpha").Amount.Int64())
	require.Equal(t, int64(1000000), supplyBefore.Sub(f.k.BankKeeper().GetSupply(f.ctx, "ualpha").Amount).Int64())
	require.Equal(t, int64(1000000), collateralBalance(f.k, f.ctx, f.guild.Id).Int64())
	require.Equal(t, int64(1000000), rankVaultBalance(t, f, f.guild.Id))
	require.Equal(t, int64(1000000), f.k.BankKeeper().GetSupply(f.ctx, "uguild."+f.guild.Id).Amount.Int64())
	require.True(t, f.tokenBalance(f.buyer).IsZero())
	require.Error(t, guild.FundCreation(founder))
	require.Equal(t, int64(1000000), f.k.BankKeeper().SpendableCoin(f.ctx, vaults[0], "ualpha").Amount.Int64())
}

func TestGuildCreationPaymentPreflightsBeforeBurn(t *testing.T) {
	for _, scenario := range []string{"insufficient", "weak_key", "different_signer", "wrong_owner"} {
		t.Run(scenario, func(t *testing.T) {
			f := newNonMemberOwnerFixture(t)
			vaults, err := f.k.NewCurrentContext(f.ctx).EnsureVaults(f.buyer.Id)
			require.NoError(t, err)
			amount := int64(2000000)
			if scenario == "insufficient" {
				amount--
			}
			fundAlpha(t, f.k, f.ctx, vaults[0], amount)
			if scenario == "weak_key" {
				cc := f.k.NewCurrentContext(f.ctx)
				cc.ClearPermissions(keeperlib.GetAddressPermissionIDBytes(f.buyer.Creator))
				cc.CommitAll()
			}
			cc := f.k.NewCurrentContext(f.ctx)
			signer := f.buyer.Creator
			if scenario == "different_signer" {
				signer = f.member.Creator
			}
			_, err = cc.GetSigningPlayer(signer)
			require.NoError(t, err)
			founder, err := cc.GetExistingPlayer(f.buyer.Id)
			require.NoError(t, err)
			if scenario == "wrong_owner" {
				founder, err = cc.GetExistingPlayer(f.member.Id)
				require.NoError(t, err)
			}
			before := f.k.BankKeeper().GetSupply(f.ctx, "ualpha")
			require.Error(t, cc.GetGuild(f.guild.Id).FundCreation(founder))
			require.Equal(t, before, f.k.BankKeeper().GetSupply(f.ctx, "ualpha"))
			require.Equal(t, amount, f.k.BankKeeper().SpendableCoin(f.ctx, vaults[0], "ualpha").Amount.Int64())
			require.True(t, f.k.BankKeeper().GetSupply(f.ctx, "uguild."+f.guild.Id).IsZero())
		})
	}
}

func TestGuildCreationAbsorbsPredictableCollateralDonation(t *testing.T) {
 f := newNonMemberOwnerFixture(t)
 vaults, err := f.k.NewCurrentContext(f.ctx).EnsureVaults(f.buyer.Id)
 require.NoError(t, err)
 fundAlpha(t, f.k, f.ctx, vaults[0], 2000000)
 cc := f.k.NewCurrentContext(f.ctx)
 founder, err := cc.GetSigningPlayer(f.buyer.Creator)
 require.NoError(t, err)
 guild := cc.GetGuild(f.guild.Id)
 fundAlpha(t, f.k, f.ctx, guild.GetBankCollateralPool(), 1)
 require.NoError(t, guild.FundCreation(founder))
 cc.CommitAll()
 require.Equal(t, int64(1000001), collateralBalance(f.k, f.ctx, f.guild.Id).Int64())
 require.Equal(t, int64(1000000), rankVaultBalance(t, f, f.guild.Id))
 require.Zero(t, f.k.BankKeeper().SpendableCoin(f.ctx, vaults[0], "ualpha").Amount.Int64())
}
