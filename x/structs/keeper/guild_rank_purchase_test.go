package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func configureRankSales(t *testing.T, f *nonMemberOwnerFixture, admission bool) {
	t.Helper()
	_, err := f.ms.GuildUpdateRankPurchase(f.ctx, &types.MsgGuildUpdateRankPurchase{
		Creator: f.buyer.Creator, GuildId: f.guild.Id, BasePrice: 10,
		StepIncrease: 2, MinimumRank: 2, JoinByRankPurchaseEnabled: admission,
	})
	require.NoError(t, err)
}

func fundRankVault(t *testing.T, f *nonMemberOwnerFixture, player types.Player, amount int64) {
	t.Helper()
	addresses, err := f.k.NewCurrentContext(f.ctx).EnsureVaults(player.Id, f.guild.Id)
	require.NoError(t, err)
	coins := sdk.NewCoins(sdk.NewInt64Coin("uguild."+f.guild.Id, amount))
	require.NoError(t, f.k.BankKeeper().MintCoins(f.ctx, types.ModuleName, coins))
	require.NoError(t, f.k.BankKeeper().SendCoinsFromModuleToAccount(f.ctx, types.ModuleName, addresses[0], coins))
}

func rankVaultBalance(t *testing.T, f *nonMemberOwnerFixture, id string) int64 {
	t.Helper()
	address, err := types.VaultAddress(id)
	require.NoError(t, err)
	return f.k.BankKeeper().SpendableCoin(f.ctx, address, "uguild."+f.guild.Id).Amount.Int64()
}

func rankPurchaseEvents(t *testing.T, f *nonMemberOwnerFixture) []*types.EventGuildRankPurchase {
	t.Helper()
	events := []*types.EventGuildRankPurchase{}
	for _, event := range f.ctx.EventManager().ABCIEvents() {
		if event.Type != "structs.structs.EventGuildRankPurchase" {
			continue
		}
		decoded, err := sdk.ParseTypedEvent(event)
		require.NoError(t, err)
		events = append(events, decoded.(*types.EventGuildRankPurchase))
	}
	return events
}

func TestRankPurchaseChargesOnlyClimbedRanksToGuildVault(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	configureRankSales(t, f, false)
	fundRankVault(t, f, f.member, 2000)
	price, err := types.CalculateGuildRankPurchasePrice(f.guild.EntryRank, 5, 3, 10, 2)
	require.NoError(t, err)
	response, err := f.ms.GuildRankPurchase(f.ctx, &types.MsgGuildRankPurchase{
		Creator: f.member.Creator, GuildId: f.guild.Id, TargetRank: 3, ExpectedRank: 5, MaxPrice: price,
	})
	require.NoError(t, err)
	require.Equal(t, price, response.Price)
	require.Equal(t, int64(price), rankVaultBalance(t, f, f.guild.Id))
	require.Equal(t, 2000-int64(price), rankVaultBalance(t, f, f.member.Id))
	player, found := f.k.GetPlayer(f.ctx, f.member.Id)
	require.True(t, found)
	require.Equal(t, uint64(3), player.GuildRank)
	require.Equal(t, f.guild.Id, player.GuildId)
	require.True(t, f.tokenBalance(f.member).IsZero())
	// Replaying the purchase cannot charge twice.
	_, err = f.ms.GuildRankPurchase(f.ctx, &types.MsgGuildRankPurchase{
		Creator: f.member.Creator, GuildId: f.guild.Id, TargetRank: 3, ExpectedRank: 5, MaxPrice: price,
	})
	require.Error(t, err)
	require.Equal(t, int64(price), rankVaultBalance(t, f, f.guild.Id))
	require.Equal(t, []*types.EventGuildRankPurchase{{GuildId: f.guild.Id, PlayerId: f.member.Id, PreviousRank: 5, Rank: 3, Price: price, Denom: "uguild." + f.guild.Id}}, rankPurchaseEvents(t, f), "rejected replay must not duplicate the paid event")
}

func TestGuildCanSellRankOneWithoutImplicitOwnershipTransfer(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	_, err := f.ms.GuildUpdateRankPurchase(f.ctx, &types.MsgGuildUpdateRankPurchase{
		Creator: f.buyer.Creator, GuildId: f.guild.Id, BasePrice: 1, MinimumRank: 1,
	})
	require.NoError(t, err)
	fundRankVault(t, f, f.member, 10)
	_, err = f.ms.GuildRankPurchase(f.ctx, &types.MsgGuildRankPurchase{
		Creator: f.member.Creator, GuildId: f.guild.Id, TargetRank: 1, ExpectedRank: 5, MaxPrice: 4,
	})
	require.NoError(t, err)
	player, found := f.k.GetPlayer(f.ctx, f.member.Id)
	require.True(t, found)
	require.Equal(t, uint64(1), player.GuildRank)
	guild, found := f.k.GetGuild(f.ctx, f.guild.Id)
	require.True(t, found)
	require.Equal(t, f.buyer.Id, guild.Owner)
	require.Equal(t, int64(4), rankVaultBalance(t, f, f.guild.Id))
	// Rank affects existing rank checks and configured grants. It does not
	// synthesize the separate ownership-transfer permission on this guild.
	_, err = f.ms.GuildUpdateOwnerId(f.ctx, &types.MsgGuildUpdateOwnerId{
		Creator: f.member.Creator, GuildId: f.guild.Id, Owner: f.member.Id,
	})
	require.Error(t, err)
	guild, found = f.k.GetGuild(f.ctx, f.guild.Id)
	require.True(t, found)
	require.Equal(t, f.buyer.Id, guild.Owner)
}

func TestRankPurchaseFailuresLeaveBalanceAndRankUnchanged(t *testing.T) {
	for _, scenario := range []string{"disabled", "price_cap", "stale_rank", "same_rank", "outside_range", "insufficient", "missing_spend_bit", "missing_membership_bit", "nonmember"} {
		t.Run(scenario, func(t *testing.T) {
			f := newNonMemberOwnerFixture(t)
			if scenario != "disabled" {
				configureRankSales(t, f, false)
			}
			fundRankVault(t, f, f.member, 2000)
			msg := &types.MsgGuildRankPurchase{Creator: f.member.Creator, GuildId: f.guild.Id, TargetRank: 3, ExpectedRank: 5, MaxPrice: 2000}
			switch scenario {
			case "price_cap":
				msg.MaxPrice = 0
			case "stale_rank":
				msg.ExpectedRank = 6
			case "same_rank":
				msg.TargetRank = 5
			case "outside_range":
				msg.TargetRank = 1
			case "insufficient":
				msg.TargetRank = 2
				cc := f.k.NewCurrentContext(f.ctx)
				require.NoError(t, cc.GetGuild(f.guild.Id).SetRankPurchase(10000, 0, 2, false))
				cc.CommitAll()
			case "missing_spend_bit", "missing_membership_bit":
				cc := f.k.NewCurrentContext(f.ctx)
				cc.ClearPermissions(keeperlib.GetAddressPermissionIDBytes(f.member.Creator))
				bit := types.PermGuildMembership
				if scenario == "missing_membership_bit" {
					bit = types.PermTokenTransfer
				}
				cc.SetPermissions(keeperlib.GetAddressPermissionIDBytes(f.member.Creator), bit)
				cc.CommitAll()
			case "nonmember":
				msg.Creator = f.buyer.Creator
			}
			_, err := f.ms.GuildRankPurchase(f.ctx, msg)
			require.Error(t, err)
			require.Equal(t, int64(2000), rankVaultBalance(t, f, f.member.Id))
			require.Zero(t, rankVaultBalance(t, f, f.guild.Id))
			player, found := f.k.GetPlayer(f.ctx, f.member.Id)
			require.True(t, found)
			require.Equal(t, uint64(5), player.GuildRank)
			require.Empty(t, rankPurchaseEvents(t, f), "rejected purchase emits no successful purchase event")
		})
	}
}

func TestRankPurchaseAdmissionIsExplicitAndSeparate(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	configureRankSales(t, f, false)
	fundRankVault(t, f, f.buyer, 2000)
	target := f.guild.EntryRank - 2
	msg := &types.MsgGuildMembershipJoinByRankPurchase{Creator: f.buyer.Creator, GuildId: f.guild.Id, TargetRank: target, MaxPrice: 22}
	_, err := f.ms.GuildMembershipJoinByRankPurchase(f.ctx, msg)
	require.Error(t, err)
	require.Zero(t, rankVaultBalance(t, f, f.guild.Id))
	configureRankSales(t, f, true)
	msg.ExpectedGuildId = "0-999"
	_, err = f.ms.GuildMembershipJoinByRankPurchase(f.ctx, msg)
	require.Error(t, err)
	msg.ExpectedGuildId = ""
	response, err := f.ms.GuildMembershipJoinByRankPurchase(f.ctx, msg)
	require.NoError(t, err)
	require.Equal(t, uint64(22), response.Price)
	player, found := f.k.GetPlayer(f.ctx, f.buyer.Id)
	require.True(t, found)
	require.Equal(t, f.guild.Id, player.GuildId)
	require.Equal(t, target, player.GuildRank)
	require.Equal(t, f.guild.EntrySubstationId, player.SubstationId)
	require.Equal(t, int64(22), rankVaultBalance(t, f, f.guild.Id))
	_, err = f.ms.GuildMembershipJoinByRankPurchase(f.ctx, msg)
	require.Error(t, err)
	require.Equal(t, int64(22), rankVaultBalance(t, f, f.guild.Id))
	require.Equal(t, []*types.EventGuildRankPurchase{{GuildId: f.guild.Id, PlayerId: f.buyer.Id, PreviousRank: f.buyer.GuildRank, Rank: target, Price: 22, Denom: "uguild." + f.guild.Id, Joined: true}}, rankPurchaseEvents(t, f))
}

func TestRankSalesConfigurationRequiresGuildAuthority(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	_, err := f.ms.GuildUpdateRankPurchase(f.ctx, &types.MsgGuildUpdateRankPurchase{Creator: f.member.Creator, GuildId: f.guild.Id, MinimumRank: 2})
	require.Error(t, err)
	for _, minimum := range []uint64{0, f.guild.EntryRank + 1} {
		_, err = f.ms.GuildUpdateRankPurchase(f.ctx, &types.MsgGuildUpdateRankPurchase{Creator: f.buyer.Creator, GuildId: f.guild.Id, MinimumRank: minimum, JoinByRankPurchaseEnabled: true})
		require.Error(t, err)
	}
}

func TestRankPurchaseEntryAdmissionCanBeFree(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	configureRankSales(t, f, true)
	response, err := f.ms.GuildMembershipJoinByRankPurchase(f.ctx, &types.MsgGuildMembershipJoinByRankPurchase{
		Creator: f.buyer.Creator, GuildId: f.guild.Id, TargetRank: f.guild.EntryRank,
	})
	require.NoError(t, err)
	require.Zero(t, response.Price)
	player, found := f.k.GetPlayer(f.ctx, f.buyer.Id)
	require.True(t, found)
	require.Equal(t, f.guild.Id, player.GuildId)
	require.Equal(t, f.guild.EntryRank, player.GuildRank)
	require.Equal(t, []*types.EventGuildRankPurchase{{GuildId: f.guild.Id, PlayerId: f.buyer.Id, PreviousRank: f.buyer.GuildRank, Rank: f.guild.EntryRank, Price: 0, Denom: "uguild." + f.guild.Id, Joined: true}}, rankPurchaseEvents(t, f), "free admission is still an observable rank and membership change")
}

func TestRankPurchaseAdmissionPreflightsSubstationBeforePayment(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	configureRankSales(t, f, true)
	fundRankVault(t, f, f.buyer, 2000)
	guild, found := f.k.GetGuild(f.ctx, f.guild.Id)
	require.True(t, found)
	guild.EntrySubstationId = "3-999999999"
	f.k.SetGuild(f.ctx, guild)
	_, err := f.ms.GuildMembershipJoinByRankPurchase(f.ctx, &types.MsgGuildMembershipJoinByRankPurchase{
		Creator: f.buyer.Creator, GuildId: f.guild.Id, TargetRank: f.guild.EntryRank - 2, MaxPrice: 22,
	})
	require.Error(t, err)
	require.Equal(t, int64(2000), rankVaultBalance(t, f, f.buyer.Id))
	require.Zero(t, rankVaultBalance(t, f, f.guild.Id))
	player, found := f.k.GetPlayer(f.ctx, f.buyer.Id)
	require.True(t, found)
	require.Empty(t, player.GuildId)
}

func TestRankPurchaseCannotLeaveOwnedGuild(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	configureRankSales(t, f, true)
	fundRankVault(t, f, f.buyer, 2000)
	// The buyer owns the destination already; owning a second current guild
	// must still prevent departure through the new admission route.
	current := f.guild
	current.Id = "0-99999"
	current.Owner = f.buyer.Id
	f.k.SetGuild(f.ctx, current)
	player, found := f.k.GetPlayer(f.ctx, f.buyer.Id)
	require.True(t, found)
	player.GuildId = current.Id
	f.k.SetPlayer(f.ctx, player)
	_, err := f.ms.GuildMembershipJoinByRankPurchase(f.ctx, &types.MsgGuildMembershipJoinByRankPurchase{
		Creator: f.buyer.Creator, GuildId: f.guild.Id, TargetRank: f.guild.EntryRank - 2, MaxPrice: 22, ExpectedGuildId: current.Id,
	})
	require.Error(t, err)
	require.Equal(t, int64(2000), rankVaultBalance(t, f, f.buyer.Id))
	require.Zero(t, rankVaultBalance(t, f, f.guild.Id))
	player, found = f.k.GetPlayer(f.ctx, f.buyer.Id)
	require.True(t, found)
	require.Equal(t, current.Id, player.GuildId)
}
