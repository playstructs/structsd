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

	t.Run("kick clears a leftover membership application via the cache commit", func(t *testing.T) {
		appMemberAcc := sdk.AccAddress("kick_app_member_pad0")
		appMember := testAppendPlayer(k, ctx, types.Player{
			Creator:        appMemberAcc.String(),
			PrimaryAddress: appMemberAcc.String(),
			GuildId:        gs.Guild.Id,
			GuildRank:      50,
		})

		// Seed a stored application row for the member, the case the removed
		// direct clear used to handle: the kick must still leave no row behind.
		k.SetGuildMembershipApplication(ctx, types.GuildMembershipApplication{
			GuildId:            gs.Guild.Id,
			PlayerId:           appMember.Id,
			JoinType:           types.GuildJoinType_request,
			RegistrationStatus: types.RegistrationStatus_proposed,
		})
		_, found := k.GetGuildMembershipApplication(ctx, gs.Guild.Id, appMember.Id)
		require.True(t, found, "precondition: an application row exists before the kick")

		_, err := ms.GuildMembershipKick(wctx, &types.MsgGuildMembershipKick{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: appMember.Id,
		})
		require.NoError(t, err)

		_, found = k.GetGuildMembershipApplication(ctx, gs.Guild.Id, appMember.Id)
		require.False(t, found, "the kick must clear the application row through the cache commit")
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

	// A player id naming nobody is refused as missing, not as a non-member. The
	// distinction matters: "not a member" was reached by reading a guild id off a
	// zero-valued cache, so the kick got as far as comparing fields on a player
	// who does not exist.
	t.Run("target player not found", func(t *testing.T) {
		_, err := ms.GuildMembershipKick(wctx, &types.MsgGuildMembershipKick{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: "1-999",
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrObjectNotFound)
		require.Contains(t, err.Error(), "1-999")
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

/* TestGuildMembershipKickClearsDirectGuildPermissions is the regression on a
 * dismissal that did not dismiss.
 *
 * Kick cleared the player's GuildId and rank and nothing else. PermissionCheck
 * reads a direct object grant with no membership predicate - only the
 * rank-derived branch is gated on being in a guild - so whatever the guild had
 * granted the player directly survived the kick. PermAdmin on a guild is all
 * GuildUpdateOwnerId requires, and it demands no membership either, so a
 * dismissed administrator could take the guild.
 *
 * The ownership path is asserted alongside because that is the escalation that
 * makes this worth a release rather than a tidy-up.
 */
func TestGuildMembershipKickClearsDirectGuildPermissions(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	adminAcc := sdk.AccAddress("kick_admin_addr_pad1")
	admin := testAppendPlayer(k, ctx, types.Player{
		Creator:        adminAcc.String(),
		PrimaryAddress: adminAcc.String(),
		GuildId:        gs.Guild.Id,
		GuildRank:      2,
	})

	// The guild delegates administration, exactly as PermissionGrantOnObject would.
	adminPermId := keeperlib.GetObjectPermissionIDBytes(gs.Guild.Id, admin.Id)
	k.SetPermissionsByBytes(ctx, adminPermId, types.PermAdmin|types.PermUpdate|types.PermGuildMembership)

	_, err := ms.GuildMembershipKick(wctx, &types.MsgGuildMembershipKick{
		Creator:  gs.GuildOwner.Creator,
		GuildId:  gs.Guild.Id,
		PlayerId: admin.Id,
	})
	require.NoError(t, err)

	kicked, found := k.GetPlayer(ctx, admin.Id)
	require.True(t, found)
	require.Equal(t, "", kicked.GuildId, "the kick itself should have landed")

	require.Equal(t, types.Permissionless, k.GetPermissionsByBytes(ctx, adminPermId),
		"a kicked member must keep no direct grant on the guild that dismissed them")

	// The escalation the grant enabled: PermAdmin is the whole of what
	// GuildUpdateOwnerId asks for, and it never checks membership.
	_, err = ms.GuildUpdateOwnerId(wctx, &types.MsgGuildUpdateOwnerId{
		Creator: admin.Creator,
		GuildId: gs.Guild.Id,
		Owner:   admin.Id,
	})
	require.Error(t, err, "a dismissed administrator must not be able to seize the guild")

	guild, guildFound := k.GetGuild(ctx, gs.Guild.Id)
	require.True(t, guildFound)
	require.Equal(t, gs.GuildOwner.Id, guild.Owner, "ownership must be untouched")
}

// The owner's row is how ownership is stored, and guilds are property: an owner
// need not be a member. GetGuildMembershipKickCache refuses to kick them, so the
// clearing above can never reach that row - this pins the refusal that makes it
// safe.
func TestGuildMembershipKickCannotReachTheOwnersGrants(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)
	ownerPermId := keeperlib.GetObjectPermissionIDBytes(gs.Guild.Id, gs.GuildOwner.Id)
	require.NotEqual(t, types.Permissionless, k.GetPermissionsByBytes(ctx, ownerPermId))

	_, err := ms.GuildMembershipKick(wctx, &types.MsgGuildMembershipKick{
		Creator:  gs.GuildOwner.Creator,
		GuildId:  gs.Guild.Id,
		PlayerId: gs.GuildOwner.Id,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot_kick_owner")

	require.NotEqual(t, types.Permissionless, k.GetPermissionsByBytes(ctx, ownerPermId),
		"the owner's grants are their ownership and must survive")
}
