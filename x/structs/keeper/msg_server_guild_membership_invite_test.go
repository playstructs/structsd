package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildMembershipInvite(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	t.Run("valid invite", func(t *testing.T) {
		target := types.Player{
			Creator:        sdk.AccAddress("inv_valid_target_pad").String(),
			PrimaryAddress: sdk.AccAddress("inv_valid_target_pad").String(),
		}
		target = testAppendPlayer(k, ctx, target)

		resp, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: target.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.GuildMembershipApplication)
		require.Equal(t, types.RegistrationStatus_proposed, resp.GuildMembershipApplication.RegistrationStatus)
		require.Equal(t, types.GuildJoinType_invite, resp.GuildMembershipApplication.JoinType)
	})

	t.Run("guild not found", func(t *testing.T) {
		target := types.Player{
			Creator:        sdk.AccAddress("inv_gnf_target_pad01").String(),
			PrimaryAddress: sdk.AccAddress("inv_gnf_target_pad01").String(),
		}
		target = testAppendPlayer(k, ctx, target)

		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  "0-999",
			PlayerId: target.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("target already a member", func(t *testing.T) {
		member := types.Player{
			Creator:        sdk.AccAddress("inv_already_memb_pad").String(),
			PrimaryAddress: sdk.AccAddress("inv_already_memb_pad").String(),
			GuildId:        gs.Guild.Id,
		}
		member = testAppendPlayer(k, ctx, member)

		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: member.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "already a member")
	})

	t.Run("invites closed", func(t *testing.T) {
		target := types.Player{
			Creator:        sdk.AccAddress("inv_closed_target_pa").String(),
			PrimaryAddress: sdk.AccAddress("inv_closed_target_pa").String(),
		}
		target = testAppendPlayer(k, ctx, target)

		guildObj, _ := k.GetGuild(ctx, gs.Guild.Id)
		guildObj.JoinInfusionMinimumBypassByInvite = types.GuildJoinBypassLevel_closed
		k.SetGuild(ctx, guildObj)
		defer func() {
			guildObj.JoinInfusionMinimumBypassByInvite = types.GuildJoinBypassLevel_member
			k.SetGuild(ctx, guildObj)
		}()

		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: target.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not currently allowing")
	})

	t.Run("non-member tries to invite at member bypass", func(t *testing.T) {
		outsider := types.Player{
			Creator:        sdk.AccAddress("inv_outsider_pad_pad").String(),
			PrimaryAddress: sdk.AccAddress("inv_outsider_pad_pad").String(),
		}
		outsider = testAppendPlayer(k, ctx, outsider)

		target := types.Player{
			Creator:        sdk.AccAddress("inv_nm_target_padpad").String(),
			PrimaryAddress: sdk.AccAddress("inv_nm_target_padpad").String(),
		}
		target = testAppendPlayer(k, ctx, target)

		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  outsider.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: target.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not a member")
	})

	t.Run("unregistered creator", func(t *testing.T) {
		unregAddr := sdk.AccAddress("inv_unreg_creator_pa").String()
		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  unregAddr,
			GuildId:  gs.Guild.Id,
			PlayerId: "1-0",
		})
		require.Error(t, err)
	})

	t.Run("invalid substation override", func(t *testing.T) {
		target := types.Player{
			Creator:        sdk.AccAddress("inv_sub_override_pad").String(),
			PrimaryAddress: sdk.AccAddress("inv_sub_override_pad").String(),
		}
		target = testAppendPlayer(k, ctx, target)

		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:      gs.GuildOwner.Creator,
			GuildId:      gs.Guild.Id,
			PlayerId:     target.Id,
			SubstationId: "4-999",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("cross-guild invite succeeds", func(t *testing.T) {
		gsB := testCreateGuild(k, ctx)

		playerInA := types.Player{
			Creator:        sdk.AccAddress("inv_crossguild_pad01").String(),
			PrimaryAddress: sdk.AccAddress("inv_crossguild_pad01").String(),
			GuildId:        gs.Guild.Id,
		}
		playerInA = testAppendPlayer(k, ctx, playerInA)

		resp, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  gsB.GuildOwner.Creator,
			GuildId:  gsB.Guild.Id,
			PlayerId: playerInA.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.GuildMembershipApplication)
		require.Equal(t, types.RegistrationStatus_proposed, resp.GuildMembershipApplication.RegistrationStatus)
		require.Equal(t, types.GuildJoinType_invite, resp.GuildMembershipApplication.JoinType)
	})

	t.Run("permissioned bypass without permission", func(t *testing.T) {
		member := types.Player{
			Creator:        sdk.AccAddress("inv_perm_bypass_memb").String(),
			PrimaryAddress: sdk.AccAddress("inv_perm_bypass_memb").String(),
			GuildId:        gs.Guild.Id,
			GuildRank:      50,
		}
		member = testAppendPlayer(k, ctx, member)

		target := types.Player{
			Creator:        sdk.AccAddress("inv_perm_bypass_targ").String(),
			PrimaryAddress: sdk.AccAddress("inv_perm_bypass_targ").String(),
		}
		target = testAppendPlayer(k, ctx, target)

		guildObj, _ := k.GetGuild(ctx, gs.Guild.Id)
		guildObj.JoinInfusionMinimumBypassByInvite = types.GuildJoinBypassLevel_permissioned
		k.SetGuild(ctx, guildObj)
		defer func() {
			guildObj.JoinInfusionMinimumBypassByInvite = types.GuildJoinBypassLevel_member
			k.SetGuild(ctx, guildObj)
		}()

		memberPermId := keeperlib.GetObjectPermissionIDBytes(gs.Guild.Id, member.Id)
		testPermissionRemove(k, ctx, memberPermId, types.PermGuildMembership)

		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  member.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: target.Id,
		})
		require.Error(t, err)
	})
}
