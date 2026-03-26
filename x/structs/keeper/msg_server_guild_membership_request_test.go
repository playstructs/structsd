package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

func TestMsgGuildMembershipRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		requesterAcc := sdk.AccAddress("req_valid_addr_pad01")
		requester := types.Player{
			Creator:        requesterAcc.String(),
			PrimaryAddress: requesterAcc.String(),
		}
		requester = testAppendPlayer(k, ctx, requester)

		resp, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  requester.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.GuildMembershipApplication)
		require.Equal(t, types.RegistrationStatus_proposed, resp.GuildMembershipApplication.RegistrationStatus)
		require.Equal(t, types.GuildJoinType_request, resp.GuildMembershipApplication.JoinType)
	})

	t.Run("guild not found", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		_ = testCreateGuild(k, ctx)

		requesterAcc := sdk.AccAddress("req_gnf_addr_pad0001")
		requester := types.Player{
			Creator:        requesterAcc.String(),
			PrimaryAddress: requesterAcc.String(),
		}
		requester = testAppendPlayer(k, ctx, requester)

		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  requester.Creator,
			GuildId:  "0-999",
			PlayerId: requester.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("already a member", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		memberAcc := sdk.AccAddress("req_member_addr_pad1")
		member := types.Player{
			Creator:        memberAcc.String(),
			PrimaryAddress: memberAcc.String(),
			GuildId:        gs.Guild.Id,
		}
		member = testAppendPlayer(k, ctx, member)

		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  member.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: member.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "already")
		require.Contains(t, err.Error(), "member")
	})

	t.Run("requests closed", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		requesterAcc := sdk.AccAddress("req_closed_addr_pad1")
		requester := types.Player{
			Creator:        requesterAcc.String(),
			PrimaryAddress: requesterAcc.String(),
		}
		requester = testAppendPlayer(k, ctx, requester)

		guildObj, _ := k.GetGuild(ctx, gs.Guild.Id)
		guildObj.JoinInfusionMinimumBypassByRequest = types.GuildJoinBypassLevel_closed
		k.SetGuild(ctx, guildObj)

		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  requester.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not")
		require.Contains(t, err.Error(), "allowing")

		guildObj.JoinInfusionMinimumBypassByRequest = types.GuildJoinBypassLevel_member
		k.SetGuild(ctx, guildObj)
	})

	t.Run("unregistered creator", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		unregAddr := sdk.AccAddress("unreg_creator_addr_0").String()
		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator: unregAddr,
			GuildId: gs.Guild.Id,
		})
		require.Error(t, err)
	})

	t.Run("cross-guild request succeeds", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gsA := testCreateGuild(k, ctx)
		gsB := testCreateGuild(k, ctx)

		playerAcc := sdk.AccAddress("req_crossguild_pad01")
		player := types.Player{
			Creator:        playerAcc.String(),
			PrimaryAddress: playerAcc.String(),
			GuildId:        gsA.Guild.Id,
		}
		player = testAppendPlayer(k, ctx, player)

		resp, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  player.Creator,
			GuildId:  gsB.Guild.Id,
			PlayerId: player.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.GuildMembershipApplication)
		require.Equal(t, types.RegistrationStatus_proposed, resp.GuildMembershipApplication.RegistrationStatus)
		require.Equal(t, types.GuildJoinType_request, resp.GuildMembershipApplication.JoinType)
	})
}
