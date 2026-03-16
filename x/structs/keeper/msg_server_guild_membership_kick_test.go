package keeper_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildMembershipKick(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	memberAcc := sdk.AccAddress("kick_member_addr_pad")
	member := types.Player{
		Creator:        memberAcc.String(),
		PrimaryAddress: memberAcc.String(),
		GuildId:        gs.Guild.Id,
		GuildRank:      50,
	}
	member = testAppendPlayer(k, ctx, member)

	t.Run("valid kick", func(t *testing.T) {
		resp, err := ms.GuildMembershipKick(wctx, &types.MsgGuildMembershipKick{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: member.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		p, found := k.GetPlayer(ctx, member.Id)
		require.True(t, found)
		require.Equal(t, "", p.GuildId)
	})

	t.Run("target not a member", func(t *testing.T) {
		nonMemberAcc := sdk.AccAddress("nonmember_kickpad_01")
		nonMember := types.Player{
			Creator:        nonMemberAcc.String(),
			PrimaryAddress: nonMemberAcc.String(),
		}
		nonMember = testAppendPlayer(k, ctx, nonMember)

		_, err := ms.GuildMembershipKick(wctx, &types.MsgGuildMembershipKick{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: nonMember.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not a member")
	})

	t.Run("cannot kick owner", func(t *testing.T) {
		kickerAcc := sdk.AccAddress("kicker_rank1_addr_pad")
		kicker := types.Player{
			Creator:        kickerAcc.String(),
			PrimaryAddress: kickerAcc.String(),
			GuildId:        gs.Guild.Id,
			GuildRank:      1,
		}
		kicker = testAppendPlayer(k, ctx, kicker)

		permId := keeperlib.GetObjectPermissionIDBytes(gs.Guild.Id, kicker.Id)
		testPermissionAdd(k, ctx, permId, types.PermGuildMembership)

		_, err := ms.GuildMembershipKick(wctx, &types.MsgGuildMembershipKick{
			Creator:  kicker.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: gs.GuildOwner.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot_kick_owner")
	})

	t.Run("lower rank cannot kick higher rank", func(t *testing.T) {
		memberAAcc := sdk.AccAddress("member_a_rank50_pad0")
		memberA := types.Player{
			Creator:        memberAAcc.String(),
			PrimaryAddress: memberAAcc.String(),
			GuildId:        gs.Guild.Id,
			GuildRank:      50,
		}
		memberA = testAppendPlayer(k, ctx, memberA)

		memberBAcc := sdk.AccAddress("member_b_rank10_pad0")
		memberB := types.Player{
			Creator:        memberBAcc.String(),
			PrimaryAddress: memberBAcc.String(),
			GuildId:        gs.Guild.Id,
			GuildRank:      10,
		}
		memberB = testAppendPlayer(k, ctx, memberB)

		permId := keeperlib.GetObjectPermissionIDBytes(gs.Guild.Id, memberA.Id)
		testPermissionAdd(k, ctx, permId, types.PermGuildMembership)

		_, err := ms.GuildMembershipKick(wctx, &types.MsgGuildMembershipKick{
			Creator:  memberA.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: memberB.Id,
		})
		require.Error(t, err)
		errStr := err.Error()
		require.True(t, strings.Contains(errStr, "permission") || strings.Contains(errStr, "administrate"),
			"expected error containing 'permission' or 'administrate', got: %s", errStr)
	})

	t.Run("target player not found", func(t *testing.T) {
		_, err := ms.GuildMembershipKick(wctx, &types.MsgGuildMembershipKick{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: "1-999",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not a member")
	})

	t.Run("unregistered creator", func(t *testing.T) {
		unregAcc := sdk.AccAddress("unreg_kick_addr_pad0")
		_, err := ms.GuildMembershipKick(wctx, &types.MsgGuildMembershipKick{
			Creator:  unregAcc.String(),
			GuildId:  gs.Guild.Id,
			PlayerId: member.Id,
		})
		require.Error(t, err)
	})
}
