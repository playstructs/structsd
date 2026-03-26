package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

func TestMsgGuildMembershipRequestApprove(t *testing.T) {
	t.Run("valid request approve", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		targetAcc := sdk.AccAddress("appr_valid_addr_pad1")
		target := types.Player{
			Creator:        targetAcc.String(),
			PrimaryAddress: targetAcc.String(),
		}
		target = testAppendPlayer(k, ctx, target)

		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  target.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: target.Id,
		})
		require.NoError(t, err)

		resp, err := ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: target.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.GuildMembershipApplication)

		player, _ := k.GetPlayer(ctx, target.Id)
		require.Equal(t, gs.Guild.Id, player.GuildId)
	})

	t.Run("cross-guild request approve migrates player", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gsA := testCreateGuild(k, ctx)
		gsB := testCreateGuild(k, ctx)

		playerAcc := sdk.AccAddress("appr_crossguild_pad1")
		player := types.Player{
			Creator:        playerAcc.String(),
			PrimaryAddress: playerAcc.String(),
			GuildId:        gsA.Guild.Id,
		}
		player = testAppendPlayer(k, ctx, player)

		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  player.Creator,
			GuildId:  gsB.Guild.Id,
			PlayerId: player.Id,
		})
		require.NoError(t, err)

		resp, err := ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{
			Creator:  gsB.GuildOwner.Creator,
			GuildId:  gsB.Guild.Id,
			PlayerId: player.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.GuildMembershipApplication)

		p, _ := k.GetPlayer(ctx, player.Id)
		require.Equal(t, gsB.Guild.Id, p.GuildId)
	})

	t.Run("no pending request", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		targetAcc := sdk.AccAddress("appr_noreq_addr_pad1")
		target := types.Player{
			Creator:        targetAcc.String(),
			PrimaryAddress: targetAcc.String(),
		}
		target = testAppendPlayer(k, ctx, target)

		_, err := ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  "0-999",
			PlayerId: target.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("join type mismatch", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		targetAcc := sdk.AccAddress("appr_mismatch_addr_p")
		target := types.Player{
			Creator:        targetAcc.String(),
			PrimaryAddress: targetAcc.String(),
		}
		target = testAppendPlayer(k, ctx, target)

		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: target.Id,
		})
		require.NoError(t, err)

		_, err = ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: target.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "join_type_mismatch")
	})

	t.Run("approver not in guild", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		targetAcc := sdk.AccAddress("appr_target_addr_pad")
		target := types.Player{
			Creator:        targetAcc.String(),
			PrimaryAddress: targetAcc.String(),
		}
		target = testAppendPlayer(k, ctx, target)

		outsiderAcc := sdk.AccAddress("appr_outsider_addr_p")
		outsider := types.Player{
			Creator:        outsiderAcc.String(),
			PrimaryAddress: outsiderAcc.String(),
		}
		outsider = testAppendPlayer(k, ctx, outsider)

		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  target.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: target.Id,
		})
		require.NoError(t, err)

		_, err = ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{
			Creator:  outsider.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: target.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not")
		require.Contains(t, err.Error(), "member")
	})

	t.Run("unregistered creator", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		targetAcc := sdk.AccAddress("appr_target_addr_pa2")
		target := types.Player{
			Creator:        targetAcc.String(),
			PrimaryAddress: targetAcc.String(),
		}
		target = testAppendPlayer(k, ctx, target)

		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  target.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: target.Id,
		})
		require.NoError(t, err)

		unregAddr := sdk.AccAddress("appr_unreg_creator_p").String()
		_, err = ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{
			Creator:  unregAddr,
			GuildId:  gs.Guild.Id,
			PlayerId: target.Id,
		})
		require.Error(t, err)
	})
}
