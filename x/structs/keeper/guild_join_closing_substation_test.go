package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
	"testing"
)

func TestGuildAdmissionRejectsEntryStationThatStartedClosing(t *testing.T) {
	for _, mode := range []string{"direct", "invite", "request", "invite_override"} {
		t.Run(mode, func(t *testing.T) {
			k, ms, ctx := setupMsgServer(t)
			gs := testCreateGuild(k, ctx)
			address := sdk.AccAddress("closing_joiner_addr__")
			player := testAppendPlayer(k, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
			station := types.Substation{Id: "4-9000", Owner: gs.GuildOwner.Id}
			k.SetSubstation(ctx, station)
			guild := gs.Guild
			if mode != "invite_override" {
				guild.EntrySubstationId = station.Id
			}
			k.SetGuild(ctx, guild)
			var err error
			if mode == "invite" || mode == "invite_override" {
				override := ""
				if mode == "invite_override" {
					override = station.Id
				}
				_, err = ms.GuildMembershipInvite(ctx, &types.MsgGuildMembershipInvite{Creator: gs.GuildOwner.Creator, GuildId: guild.Id, PlayerId: player.Id, SubstationId: override})
				require.NoError(t, err)
			} else if mode == "request" {
				_, err = ms.GuildMembershipRequest(ctx, &types.MsgGuildMembershipRequest{Creator: player.Creator, GuildId: guild.Id, PlayerId: player.Id})
				require.NoError(t, err)
			}
			application, pending := k.GetGuildMembershipApplication(ctx, guild.Id, player.Id)
			station.DeletionPending = true
			k.SetSubstation(ctx, station)
			switch mode {
			case "direct":
				_, err = ms.GuildMembershipJoin(ctx, &types.MsgGuildMembershipJoin{Creator: player.Creator, GuildId: guild.Id, PlayerId: player.Id})
			case "invite", "invite_override":
				_, err = ms.GuildMembershipInviteApprove(ctx, &types.MsgGuildMembershipInviteApprove{Creator: player.Creator, GuildId: guild.Id, PlayerId: player.Id})
			case "request":
				_, err = ms.GuildMembershipRequestApprove(ctx, &types.MsgGuildMembershipRequestApprove{Creator: gs.GuildOwner.Creator, GuildId: guild.Id, PlayerId: player.Id})
			}
			require.ErrorContains(t, err, "substation_deleting")
			actual, found := k.GetPlayer(ctx, player.Id)
			require.True(t, found)
			require.Equal(t, player, actual, "failed admission must preserve membership, rank and connection")
			if pending {
				actual, found := k.GetGuildMembershipApplication(ctx, guild.Id, player.Id)
				require.True(t, found)
				require.Equal(t, application, actual, "failed acceptance must retain the proposal")
			}
			require.Empty(t, k.GetAllPlayerIdBySubstationIndex(ctx, station.Id))
		})
	}
}
