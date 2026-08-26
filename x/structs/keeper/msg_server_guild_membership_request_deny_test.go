package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildMembershipRequestDeny(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	requesterAcc := sdk.AccAddress("req_deny_addr_pad_01")
	requester := types.Player{
		Creator:        requesterAcc.String(),
		PrimaryAddress: requesterAcc.String(),
	}
	requester = testAppendPlayer(k, ctx, requester)

	requesterPermId := keeperlib.GetAddressPermissionIDBytes(requester.Creator)
	testPermissionAdd(k, ctx, requesterPermId, types.PermGuildMembership)

	t.Run("valid request deny", func(t *testing.T) {
		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  requester.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.NoError(t, err)

		resp, err := ms.GuildMembershipRequestDeny(wctx, &types.MsgGuildMembershipRequestDeny{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.GuildMembershipApplication)
		require.Equal(t, types.RegistrationStatus_denied, resp.GuildMembershipApplication.RegistrationStatus)
	})

	t.Run("no pending request", func(t *testing.T) {
		// This case was skipped with the bug written out as the reason — "handler
		// creates and denies; no 'not found' error path". That was the loader
		// synthesizing the request it was about to deny, which on the approve
		// path was a force-join.
		strangerAcc := sdk.AccAddress("deny_no_request_pad1")
		stranger := types.Player{
			Creator:        strangerAcc.String(),
			PrimaryAddress: strangerAcc.String(),
		}
		stranger = testAppendPlayer(k, ctx, stranger)

		_, err := ms.GuildMembershipRequestDeny(wctx, &types.MsgGuildMembershipRequestDeny{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: stranger.Id,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrGuildMembershipApplication)
		require.Contains(t, err.Error(), "no application on file")
	})

	t.Run("denier not in guild", func(t *testing.T) {
		outsiderAcc := sdk.AccAddress("outsider_deny_pad_01")
		outsider := types.Player{
			Creator:        outsiderAcc.String(),
			PrimaryAddress: outsiderAcc.String(),
		}
		outsider = testAppendPlayer(k, ctx, outsider)

		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  requester.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.NoError(t, err)

		_, err = ms.GuildMembershipRequestDeny(wctx, &types.MsgGuildMembershipRequestDeny{
			Creator:  outsider.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not a member")
	})

	t.Run("unregistered creator", func(t *testing.T) {
		unregAcc := sdk.AccAddress("unreg_deny_addr_pad_0")
		_, err := ms.GuildMembershipRequestDeny(wctx, &types.MsgGuildMembershipRequestDeny{
			Creator:  unregAcc.String(),
			GuildId:  gs.Guild.Id,
			PlayerId: requester.Id,
		})
		require.Error(t, err)
	})
}

/* TestGuildBypassMemberTierKeepsTheSigningKeyCeiling is the regression on a
 * guild's join policy quietly un-scoping its members' delegated keys.
 *
 * PermissionCheck has two independent layers: the key that signed must hold the
 * permission, and the player must have standing on the object. The `member`
 * bypass tier is a statement about the second - any member may act, no grant on
 * the guild needed - but it was implemented by skipping PermissionCheck
 * altogether, which dropped the first as well.
 *
 * The result was that the same restricted key got two different answers
 * depending on a setting that has nothing to do with it: refused while the guild
 * sat at `permissioned`, accepted the moment an admin opened it to `member`.
 * Both request and invite tiers shared the helper, so both are covered here.
 */
func TestGuildBypassMemberTierKeepsTheSigningKeyCeiling(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// testCreateGuild opens both tiers to `member`.
	gs := testCreateGuild(k, ctx)
	require.Equal(t, types.GuildJoinBypassLevel_member, gs.Guild.JoinInfusionMinimumBypassByRequest)

	memberAcc := sdk.AccAddress("bypass_member_pad001")
	member := testAppendPlayer(k, ctx, types.Player{
		Creator:        memberAcc.String(),
		PrimaryAddress: memberAcc.String(),
		GuildId:        gs.Guild.Id,
		GuildRank:      5,
	})

	// A delegated key of that member, scoped to play and nothing else.
	limitedAcc := sdk.AccAddress("bypass_limited_pad01")
	_ = k.SetPlayerIndexForAddress(ctx, limitedAcc.String(), member.Index)
	k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(limitedAcc.String()), types.PermPlay)

	newRequest := func(t *testing.T, seed string) types.Player {
		t.Helper()
		acc := sdk.AccAddress(seed)
		applicant := testAppendPlayer(k, ctx, types.Player{
			Creator:        acc.String(),
			PrimaryAddress: acc.String(),
		})
		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  applicant.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: applicant.Id,
		})
		require.NoError(t, err)
		return applicant
	}

	t.Run("a restricted key cannot deny a pending request", func(t *testing.T) {
		applicant := newRequest(t, "bypass_applicant_p01")

		_, err := ms.GuildMembershipRequestDeny(wctx, &types.MsgGuildMembershipRequestDeny{
			Creator:  limitedAcc.String(),
			GuildId:  gs.Guild.Id,
			PlayerId: applicant.Id,
		})
		require.Error(t, err, "member tier waives the object grant, not the key ceiling")

		// The request has to survive, or the denial landed anyway.
		stored, found := k.GetGuildMembershipApplication(ctx, gs.Guild.Id, applicant.Id)
		require.True(t, found, "the pending request must still be there")
		require.Equal(t, types.RegistrationStatus_proposed, stored.RegistrationStatus)
	})

	t.Run("a restricted key cannot approve a pending request", func(t *testing.T) {
		applicant := newRequest(t, "bypass_applicant_p02")

		_, err := ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{
			Creator:  limitedAcc.String(),
			GuildId:  gs.Guild.Id,
			PlayerId: applicant.Id,
		})
		require.Error(t, err)

		after, found := k.GetPlayer(ctx, applicant.Id)
		require.True(t, found)
		require.Empty(t, after.GuildId, "and must not have joined them")
	})

	t.Run("a restricted key cannot invite", func(t *testing.T) {
		outsiderAcc := sdk.AccAddress("bypass_outsider_p001")
		outsider := testAppendPlayer(k, ctx, types.Player{
			Creator:        outsiderAcc.String(),
			PrimaryAddress: outsiderAcc.String(),
		})

		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  limitedAcc.String(),
			GuildId:  gs.Guild.Id,
			PlayerId: outsider.Id,
		})
		require.Error(t, err, "the invite tier shares the helper and the same ceiling")
	})

	// The tier still does what it is for: a member acting with a key that
	// carries the bit needs no grant on the guild object.
	t.Run("a member with the bit still needs no object grant", func(t *testing.T) {
		applicant := newRequest(t, "bypass_applicant_p03")

		require.Equal(t, types.Permissionless,
			k.GetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(gs.Guild.Id, member.Id)),
			"this member holds nothing on the guild object; standing is all they have")

		_, err := ms.GuildMembershipRequestDeny(wctx, &types.MsgGuildMembershipRequestDeny{
			Creator:  member.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: applicant.Id,
		})
		require.NoError(t, err)
	})
}
