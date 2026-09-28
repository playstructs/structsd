package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestPaidGuildCreate(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	vaults, err := f.k.NewCurrentContext(f.ctx).EnsureVaults(f.buyer.Id)
	require.NoError(t, err)
	fundAlpha(t, f.k, f.ctx, vaults[0], 3000000)
	// The fixture reactor belongs to another player. Founding must neither
	// require permission on it nor consume its old one-time entitlement.
	reactorBefore, found := f.k.GetReactor(f.ctx, f.guild.PrimaryReactorId)
	require.True(t, found)
	response, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{Creator: f.buyer.Creator, ReactorId: reactorBefore.Id, Endpoint: "paid-guild"})
	require.NoError(t, err)
	guild, found := f.k.GetGuild(f.ctx, response.GuildId)
	require.True(t, found)
	require.Equal(t, f.buyer.Id, guild.Owner)
	require.Equal(t, reactorBefore.Id, guild.PrimaryReactorId)
	require.Empty(t, guild.CharterSolverId)
	require.Equal(t, "paid-guild", guild.Endpoint)
	player, found := f.k.GetPlayer(f.ctx, f.buyer.Id)
	require.True(t, found)
	require.Equal(t, guild.Id, player.GuildId)
	require.Equal(t, uint64(1), player.GuildRank)
	require.Equal(t, int64(1000000), f.k.BankKeeper().SpendableCoin(f.ctx, vaults[0], "ualpha").Amount.Int64())
	require.Equal(t, int64(1000000), collateralBalance(f.k, f.ctx, guild.Id).Int64())
	address, err := types.VaultAddress(guild.Id)
	require.NoError(t, err)
	require.Equal(t, int64(1000000), f.k.BankKeeper().SpendableCoin(f.ctx, address, "uguild."+guild.Id).Amount.Int64())
	after, found := f.k.GetReactor(f.ctx, reactorBefore.Id)
	require.True(t, found)
	require.Equal(t, reactorBefore, after)
	purchased, found := f.k.GetGuild(f.ctx, f.guild.Id)
	require.True(t, found)
	require.Equal(t, f.buyer.Id, purchased.Owner, "founding preserves ownership of a guild the founder never joined")
}

// Paid founding retains the old charter's membership and connection semantics.
// Use the connection cache to seed both the player and its scoped index/count.
func TestPaidGuildCreatePreservesMembershipConnectionRules(t *testing.T) {
	for _, scenario := range []string{"old_entry", "personal", "new_entry"} {
		t.Run(scenario, func(t *testing.T) {
			f := newNonMemberOwnerFixture(t)
			oldEntry, _, err := testAppendSubstation(f.k, f.ctx, types.Allocation{}, f.buyer)
			require.NoError(t, err)
			personal, _, err := testAppendSubstation(f.k, f.ctx, types.Allocation{}, f.member)
			require.NoError(t, err)
			oldGuild, found := f.k.GetGuild(f.ctx, f.guild.Id)
			require.True(t, found)
			oldGuild.EntrySubstationId = oldEntry.Id
			f.k.SetGuild(f.ctx, oldGuild)
			initialStation, expectedStation, newEntry := oldEntry.Id, "", ""
			if scenario == "personal" {
				initialStation, expectedStation = personal.Id, personal.Id
			} else if scenario == "new_entry" {
				newEntry, expectedStation = personal.Id, personal.Id
			}
			cc := f.k.NewCurrentContext(f.ctx)
			cc.GetPlayer(f.member.Id).MigrateSubstation(initialStation)
			cc.CommitAll()
			vaults, err := f.k.NewCurrentContext(f.ctx).EnsureVaults(f.member.Id)
			require.NoError(t, err)
			fundAlpha(t, f.k, f.ctx, vaults[0], 2000000)

			response, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
				Creator: f.member.Creator, ReactorId: oldGuild.PrimaryReactorId, EntrySubstationId: newEntry,
			})
			require.NoError(t, err)
			player, found := f.k.GetPlayer(f.ctx, f.member.Id)
			require.True(t, found)
			require.Equal(t, response.GuildId, player.GuildId)
			require.Equal(t, uint64(1), player.GuildRank)
			require.Equal(t, expectedStation, player.SubstationId)
			guild, found := f.k.GetGuild(f.ctx, response.GuildId)
			require.True(t, found)
			require.Equal(t, f.member.Id, guild.Owner)
			require.Equal(t, newEntry, guild.EntrySubstationId)
			unchangedGuild, found := f.k.GetGuild(f.ctx, oldGuild.Id)
			require.True(t, found)
			require.Equal(t, oldGuild, unchangedGuild)
			cc = f.k.NewCurrentContext(f.ctx)
			for _, station := range []string{oldEntry.Id, personal.Id} {
				expectedCount := 0
				if station == expectedStation {
					expectedCount = 1
				}
				require.Len(t, cc.GetAllPlayerBySubstation(station), expectedCount)
				require.Equal(t, uint64(expectedCount), cc.GetSubstation(station).GetConnectionCount())
			}
		})
	}
}

func TestPaidGuildCreateRejectsBeforeAllocatingGuild(t *testing.T) {
	for _, scenario := range []string{"proof", "nonce", "consent_address", "consent_key", "consent_signature", "other_founder", "missing_reactor", "missing_substation", "uncontrolled_substation", "unfunded", "weak_key", "owns_current_guild"} {
		t.Run(scenario, func(t *testing.T) {
			f := newNonMemberOwnerFixture(t)
			vaults, err := f.k.NewCurrentContext(f.ctx).EnsureVaults(f.buyer.Id)
			require.NoError(t, err)
			if scenario != "unfunded" {
				fundAlpha(t, f.k, f.ctx, vaults[0], 2000000)
			}
			msg := &types.MsgGuildCreate{Creator: f.buyer.Creator, ReactorId: f.guild.PrimaryReactorId}
			switch scenario {
			case "proof":
				msg.Proof = "retired"
			case "nonce":
				msg.Nonce = "retired"
			case "consent_address":
				msg.Address = f.member.Creator
			case "consent_key":
				msg.ProofPubKey = "retired"
			case "consent_signature":
				msg.ProofSignature = "retired"
			case "other_founder":
				msg.FounderPlayerId = f.member.Id
			case "missing_reactor":
				msg.ReactorId = "4-999999999"
			case "missing_substation":
				msg.EntrySubstationId = "5-999999999"
			case "uncontrolled_substation":
				station := types.Substation{Id: "5-99999", Owner: f.member.Id, Creator: f.member.Creator}
				f.k.SetSubstation(f.ctx, station)
				msg.EntrySubstationId = station.Id
			case "weak_key":
				cc := f.k.NewCurrentContext(f.ctx)
				cc.ClearPermissions(keeperlib.GetAddressPermissionIDBytes(f.buyer.Creator))
				cc.SetPermissions(keeperlib.GetAddressPermissionIDBytes(f.buyer.Creator), types.PermPlay)
				cc.CommitAll()
			case "owns_current_guild":
				player := f.buyer
				player.GuildId = f.guild.Id
				f.k.SetPlayer(f.ctx, player)
			}
			count := f.k.GetGuildCount(f.ctx)
			balance := f.k.BankKeeper().SpendableCoin(f.ctx, vaults[0], "ualpha")
			_, err = f.ms.GuildCreate(f.ctx, msg)
			require.Error(t, err)
			require.Equal(t, count, f.k.GetGuildCount(f.ctx))
			require.Equal(t, balance, f.k.BankKeeper().SpendableCoin(f.ctx, vaults[0], "ualpha"))
		})
	}
}
