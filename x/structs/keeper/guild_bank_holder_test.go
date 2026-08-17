package keeper_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestGuildBankDenomSendRestriction(t *testing.T) {
	k, _, ctx := setupMsgServer(t)

	registered := sdk.AccAddress("guildholder_registered")
	player := testAppendPlayer(k, ctx, types.Player{
		Creator:        registered.String(),
		PrimaryAddress: registered.String(),
	})
	require.NotEmpty(t, player.Id)

	secondary := sdk.AccAddress("guildholder_secondary")
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, secondary.String(), player.Index))
	unregistered := sdk.AccAddress("guildholder_unregistered")
	providerId := "10-77"
	collateral := keeperlib.GetProviderCollateralPoolLocation(providerId)
	earnings := keeperlib.GetProviderEarningsPoolLocation(providerId)
	k.IndexProviderPoolAddresses(ctx, providerId)

	guildCoins := sdk.NewCoins(sdk.NewCoin("uguild.0-1", math.NewInt(10)))
	alphaCoins := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(10)))

	assertAllowed := func(t *testing.T, to sdk.AccAddress, coins sdk.Coins) {
		t.Helper()
		got, err := k.GuildBankDenomSendRestriction(ctx, registered, to, coins)
		require.NoError(t, err)
		require.True(t, got.Equals(to))
	}
	assertDenied := func(t *testing.T, to sdk.AccAddress) {
		t.Helper()
		_, err := k.GuildBankDenomSendRestriction(ctx, registered, to, guildCoins)
		var destinationErr *types.GuildBankDestinationError
		require.ErrorAs(t, err, &destinationErr)
		require.Equal(t, to.String(), destinationErr.Address)
	}

	assertAllowed(t, registered, guildCoins)
	assertAllowed(t, secondary, guildCoins)
	assertAllowed(t, unregistered, alphaCoins)
	assertAllowed(t, k.AccountKeeper().GetModuleAddress(types.ModuleName), guildCoins)
	assertAllowed(t, collateral, guildCoins)
	assertAllowed(t, earnings, guildCoins)
	assertDenied(t, unregistered)

	got, err := k.GuildBankDenomSendRestriction(ctx, unregistered, registered, guildCoins)
	require.NoError(t, err)
	require.True(t, got.Equals(registered), "an unusual pre-upgrade source must be able to exit to a player")

	k.RemoveProviderPoolAddresses(ctx, providerId)
	assertDenied(t, collateral)
	assertDenied(t, earnings)
}

func TestGuildBankConfiscationProtectsOnlyProviderObligation(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	reactor := f.k.AppendReactor(f.ctx, types.Reactor{RawAddress: sdk.ValAddress(f.consumerAcc.Bytes()).Bytes()})
	guild := f.k.AppendGuild(f.ctx, "test-endpoint", "", reactor, f.consumer, "")
	f.consumer.GuildId = guild.Id
	f.k.SetPlayer(f.ctx, f.consumer)

	guildDenom := "uguild." + guild.Id
	f.provider.Rate = sdk.NewCoin(guildDenom, math.NewInt(10))
	_, err := f.k.SetProvider(f.ctx, f.provider)
	require.NoError(t, err)

	confiscate := func(guildId string, amount math.Int, address sdk.AccAddress) error {
		cc := f.k.NewCurrentContext(f.ctx)
		err := cc.GetGuild(guildId).BankConfiscateAndBurn(amount, address.String())
		if err == nil {
			cc.CommitAll()
		}
		return err
	}

	directDeposit := math.NewInt(19)
	fundGuildToken(t, f.k, f.ctx, f.collateralAcc, guildDenom, directDeposit)
	require.NoError(t, confiscate(guild.Id, directDeposit, f.collateralAcc))

	foreignGuildId := "0-foreign"
	foreignDenom := "uguild." + foreignGuildId
	foreignDeposit := math.NewInt(17)
	fundGuildToken(t, f.k, f.ctx, f.collateralAcc, foreignDenom, foreignDeposit)
	require.NoError(t, confiscate(foreignGuildId, foreignDeposit, f.collateralAcc))

	const capacity, duration = uint64(100), uint64(50)
	collateral := math.NewIntFromUint64(capacity).Mul(math.NewIntFromUint64(duration)).Mul(f.provider.Rate.Amount)
	fundGuildToken(t, f.k, f.ctx, f.consumerAcc, guildDenom, collateral)

	_, err = f.ms.AgreementOpen(f.ctx, &types.MsgAgreementOpen{
		Creator:    f.consumer.Creator,
		ProviderId: f.provider.Id,
		Capacity:   capacity,
		Duration:   duration,
	})
	require.NoError(t, err)

	err = confiscate(guild.Id, math.OneInt(), f.collateralAcc)
	var protectedErr *types.GuildBankConfiscationError
	require.ErrorAs(t, err, &protectedErr)
	require.Equal(t, "consumer_collateral", protectedErr.Reason)
	require.Equal(t, collateral, f.k.ProviderCollateralObligation(f.ctx, f.provider))

	cc := f.k.NewCurrentContext(f.ctx)
	err = cc.GetGuild(guild.Id).BankConfiscateAndBurn(math.OneInt(), strings.ToUpper(f.collateralAcc.String()))
	require.ErrorAs(t, err, &protectedErr)
	require.Equal(t, "consumer_collateral", protectedErr.Reason,
		"bech32 casing must not bypass provider collateral protection")

	excess := math.NewInt(37)
	fundGuildToken(t, f.k, f.ctx, f.collateralAcc, guildDenom, excess)
	require.NoError(t, confiscate(guild.Id, excess, f.collateralAcc))
	require.Equal(t, collateral, f.k.BankKeeper().SpendableCoin(f.ctx, f.collateralAcc, guildDenom).Amount)

	earnings := math.NewInt(23)
	fundGuildToken(t, f.k, f.ctx, f.earningsAcc, guildDenom, earnings)
	require.NoError(t, confiscate(guild.Id, earnings, f.earningsAcc))
	require.True(t, f.k.BankKeeper().SpendableCoin(f.ctx, f.earningsAcc, guildDenom).Amount.IsZero())

	legacyEscrow := sdk.AccAddress("legacy_ibc_escrow_addr")
	f.k.SetLegacyGuildBankEscrow(f.ctx, legacyEscrow.String())
	fundGuildToken(t, f.k, f.ctx, legacyEscrow, guildDenom, math.OneInt())
	err = confiscate(guild.Id, math.OneInt(), legacyEscrow)
	require.True(t, errors.As(err, &protectedErr))
	require.Equal(t, "ibc_escrow_backing", protectedErr.Reason)

	cc = f.k.NewCurrentContext(f.ctx)
	err = cc.GetGuild(guild.Id).BankConfiscateAndBurn(math.OneInt(), strings.ToUpper(legacyEscrow.String()))
	require.True(t, errors.As(err, &protectedErr))
	require.Equal(t, "ibc_escrow_backing", protectedErr.Reason,
		"bech32 casing must not bypass legacy escrow protection")
}

func TestProviderDeleteClearsGuildBankPoolRoles(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")

	_, _, found := f.k.GetProviderPoolAddress(f.ctx, f.collateralAcc.String())
	require.True(t, found)
	_, _, found = f.k.GetProviderPoolAddress(f.ctx, f.earningsAcc.String())
	require.True(t, found)

	cc := f.k.NewCurrentContext(f.ctx)
	require.NoError(t, cc.GetProvider(f.provider.Id).Delete())
	cc.CommitAll()

	_, _, found = f.k.GetProviderPoolAddress(f.ctx, f.collateralAcc.String())
	require.False(t, found)
	_, _, found = f.k.GetProviderPoolAddress(f.ctx, f.earningsAcc.String())
	require.False(t, found)
}

func fundGuildToken(t *testing.T, k keeperlib.Keeper, ctx context.Context, account sdk.AccAddress, denom string, amount math.Int) {
	t.Helper()
	coins := sdk.NewCoins(sdk.NewCoin(denom, amount))
	require.NoError(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, account, coins))
}
