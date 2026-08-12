package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

/* Guild membership is two-sided: a player files a request and the guild approves
 * it, or the guild issues an invite and the player accepts it. The stored
 * application is the only evidence the first leg happened, so a loader that
 * synthesizes one on a store miss hands each approval path the very consent it is
 * about to verify.
 *
 * These tests pin the player-side leg for the paths where the guild acts, since
 * that is the leg synthesis erased.
 */

// testRegisterGuildlessPlayer registers a player who belongs to no guild.
func testRegisterGuildlessPlayer(k keeperlib.Keeper, ctx context.Context, seed string) types.Player {
	acc := sdk.AccAddress(seed)
	return testAppendPlayer(k, ctx, types.Player{
		Creator:        acc.String(),
		PrimaryAddress: acc.String(),
	})
}

// testGuildMemberWithSubstation registers an ordinary member of gs's guild who
// owns a substation of their own. The substation is what SetSubstationIdOverride
// needs to accept them as a destination, since it checks rights there and not on
// the guild; the membership is what CanInviteMembers checks at bypass level
// member. Deliberately given no permission on the guild object, so bypass level
// permissioned refuses them.
func testGuildMemberWithSubstation(k keeperlib.Keeper, ctx context.Context, gs testGuildSetup, seed string) (types.Player, types.Substation) {
	acc := sdk.AccAddress(seed)
	member := testAppendPlayer(k, ctx, types.Player{
		Creator:        acc.String(),
		PrimaryAddress: acc.String(),
		GuildId:        gs.Guild.Id,
		GuildRank:      5,
	})

	alloc, _ := testAppendAllocation(k, ctx, types.Allocation{
		SourceObjectId: member.Id,
		Controller:     member.Id,
		Type:           types.AllocationType_static,
	}, 100)

	substation, _, _ := testAppendSubstation(k, ctx, alloc, member)
	testPermissionAdd(k, ctx, keeperlib.GetObjectPermissionIDBytes(substation.Id, member.Id), types.PermAll)

	return member, substation
}

func TestGuildMembershipForceJoinIsRefused(t *testing.T) {
	// The headline exploit. The guild owner has every guild-side permission
	// there is and the guild is recruiting, so VerifyRequestAsGuild passes; the
	// only thing standing between them and the victim's player record is the
	// absence of a request the victim filed.
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)
	gs := testCreateGuild(k, ctx)

	victim := testRegisterGuildlessPlayer(k, ctx, "consent_forcejoin_vic1")
	before, found := k.GetPlayer(ctx, victim.Id)
	require.True(t, found)

	_, err := ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{
		Creator:  gs.GuildOwner.Creator,
		GuildId:  gs.Guild.Id,
		PlayerId: victim.Id,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrGuildMembershipApplication)

	after, found := k.GetPlayer(ctx, victim.Id)
	require.True(t, found)
	require.Equal(t, before.GuildId, after.GuildId, "victim was moved into the guild without requesting membership")
	require.Equal(t, before.GuildRank, after.GuildRank, "victim's guild rank was overwritten")
	require.Equal(t, before.SubstationId, after.SubstationId, "victim's substation connection was changed")
}

func TestGuildMembershipForceJoinDoesNotEvictFromExistingGuild(t *testing.T) {
	// The same exploit against a player who already belongs somewhere, which is
	// the damaging version: MigrateGuild overwrites GuildId with no check on the
	// guild being left, so a member in good standing lands in the attacker's
	// guild at its entry rank.
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)
	home := testCreateGuild(k, ctx)
	attacker := testCreateGuild(k, ctx)

	acc := sdk.AccAddress("consent_evict_victim1")
	victim := testAppendPlayer(k, ctx, types.Player{
		Creator:        acc.String(),
		PrimaryAddress: acc.String(),
		GuildId:        home.Guild.Id,
		GuildRank:      4,
	})

	_, err := ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{
		Creator:  attacker.GuildOwner.Creator,
		GuildId:  attacker.Guild.Id,
		PlayerId: victim.Id,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrGuildMembershipApplication)

	after, _ := k.GetPlayer(ctx, victim.Id)
	require.Equal(t, home.Guild.Id, after.GuildId)
	require.Equal(t, uint64(4), after.GuildRank)
}

// TestGuildMembershipRefusesPlayerIdsThatNameNobody covers the id that is not a
// victim at all, which the loader used to accept as readily as a real one.
//
// resolveGuildMembershipApplication guarded its lookup on CurrentContext.GetPlayer,
// which never returns an error, so the check was dead and a phantom id read as a
// guildless player: GetGuildId on a cache that failed to load returns "", which is
// not the guild, so the already-a-member check waved it through as well. What
// followed was a write against a zero-valued player, and since PlayerCache.Commit
// keys the write by Player.Id rather than by the id asked for, the store was handed
// an empty key and panicked — a recovered panic and a failed transaction rather
// than the corruption first reported, but only by luck of where the write landed.
//
// The refusal now happens up front, on both the loader that opens an application
// and the kick loader.
func TestGuildMembershipRefusesPlayerIdsThatNameNobody(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)
	gs := testCreateGuild(k, ctx)

	phantom := "1-999"
	_, found := k.GetPlayer(ctx, phantom)
	require.False(t, found, "the fixture depends on this id naming nobody")

	t.Run("invite", func(t *testing.T) {
		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: phantom,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrObjectNotFound)
		require.Contains(t, err.Error(), phantom)
	})

	t.Run("request approve", func(t *testing.T) {
		_, err := ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: phantom,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrObjectNotFound)
	})

	t.Run("kick", func(t *testing.T) {
		_, err := ms.GuildMembershipKick(wctx, &types.MsgGuildMembershipKick{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: phantom,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrObjectNotFound)
	})

	// Nothing was filed under the phantom id on the way to any of those refusals.
	_, found = k.GetGuildMembershipApplication(ctx, gs.Guild.Id, phantom)
	require.False(t, found, "a refused membership path must not leave an application behind")
}

func TestGuildMembershipTerminalTransitionsRequireAnApplication(t *testing.T) {
	// Every path that consumes an application, not just the one that migrates a
	// player. Deny and revoke were less dramatic — they set a status that Commit
	// turns into a delete, so the write was a no-op — but each reported success
	// for an application that did not exist, which is a lie a client acts on.
	cases := []struct {
		name string
		// addressSeed must be distinct per case and 20 bytes, matching the
		// padded literals the other keeper tests use.
		addressSeed string
		// callerIsGuild picks which side the handler verifies, per the split in
		// the six msg_server_guild_membership_*_{approve,deny,revoke}.go files.
		callerIsGuild bool
		send          func(ms types.MsgServer, wctx sdk.Context, creator string, guildId string, playerId string) error
	}{
		{
			name:          "request approve",
			addressSeed:   "consent_none_reqappr1",
			callerIsGuild: true,
			send: func(ms types.MsgServer, wctx sdk.Context, creator, guildId, playerId string) error {
				_, err := ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{Creator: creator, GuildId: guildId, PlayerId: playerId})
				return err
			},
		},
		{
			name:          "request deny",
			addressSeed:   "consent_none_reqdeny1",
			callerIsGuild: true,
			send: func(ms types.MsgServer, wctx sdk.Context, creator, guildId, playerId string) error {
				_, err := ms.GuildMembershipRequestDeny(wctx, &types.MsgGuildMembershipRequestDeny{Creator: creator, GuildId: guildId, PlayerId: playerId})
				return err
			},
		},
		{
			name:          "request revoke",
			addressSeed:   "consent_none_reqrevo1",
			callerIsGuild: false,
			send: func(ms types.MsgServer, wctx sdk.Context, creator, guildId, playerId string) error {
				_, err := ms.GuildMembershipRequestRevoke(wctx, &types.MsgGuildMembershipRequestRevoke{Creator: creator, GuildId: guildId, PlayerId: playerId})
				return err
			},
		},
		{
			name:          "invite approve",
			addressSeed:   "consent_none_invappr1",
			callerIsGuild: false,
			send: func(ms types.MsgServer, wctx sdk.Context, creator, guildId, playerId string) error {
				_, err := ms.GuildMembershipInviteApprove(wctx, &types.MsgGuildMembershipInviteApprove{Creator: creator, GuildId: guildId, PlayerId: playerId})
				return err
			},
		},
		{
			name:          "invite deny",
			addressSeed:   "consent_none_invdeny1",
			callerIsGuild: false,
			send: func(ms types.MsgServer, wctx sdk.Context, creator, guildId, playerId string) error {
				_, err := ms.GuildMembershipInviteDeny(wctx, &types.MsgGuildMembershipInviteDeny{Creator: creator, GuildId: guildId, PlayerId: playerId})
				return err
			},
		},
		{
			name:          "invite revoke",
			addressSeed:   "consent_none_invrevo1",
			callerIsGuild: true,
			send: func(ms types.MsgServer, wctx sdk.Context, creator, guildId, playerId string) error {
				_, err := ms.GuildMembershipInviteRevoke(wctx, &types.MsgGuildMembershipInviteRevoke{Creator: creator, GuildId: guildId, PlayerId: playerId})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, ms, ctx := setupMsgServer(t)
			wctx := sdk.UnwrapSDKContext(ctx)
			gs := testCreateGuild(k, ctx)

			target := testRegisterGuildlessPlayer(k, ctx, tc.addressSeed)

			creator := target.Creator
			if tc.callerIsGuild {
				creator = gs.GuildOwner.Creator
			}

			err := tc.send(ms, wctx, creator, gs.Guild.Id, target.Id)
			require.Error(t, err)
			require.ErrorIs(t, err, types.ErrGuildMembershipApplication)

			_, found := k.GetGuildMembershipApplication(ctx, gs.Guild.Id, target.Id)
			require.False(t, found, "a refused transition left an application behind")

			after, _ := k.GetPlayer(ctx, target.Id)
			require.Equal(t, "", after.GuildId)
		})
	}
}

func TestGuildMembershipApproveRefusesNonProposedApplication(t *testing.T) {
	// requirePending's second half. No public path can write a non-proposed row
	// today — Commit persists proposed and clears the other three — so this
	// seeds one directly, which is also how a genesis file could carry one.
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)
	gs := testCreateGuild(k, ctx)

	target := testRegisterGuildlessPlayer(k, ctx, "consent_nonprop_targ1")

	k.SetGuildMembershipApplication(ctx, types.GuildMembershipApplication{
		GuildId:            gs.Guild.Id,
		PlayerId:           target.Id,
		Proposer:           target.Id,
		JoinType:           types.GuildJoinType_request,
		RegistrationStatus: types.RegistrationStatus_denied,
	})

	_, err := ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{
		Creator:  gs.GuildOwner.Creator,
		GuildId:  gs.Guild.Id,
		PlayerId: target.Id,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrGuildMembershipApplication)
	require.Contains(t, err.Error(), "no longer pending")

	after, _ := k.GetPlayer(ctx, target.Id)
	require.Equal(t, "", after.GuildId)
}

func TestGuildMembershipInviteCannotBeRetargetedByOutsider(t *testing.T) {
	// The mirror defect on the creation side. GuildMembershipInvite has no Verify
	// call of its own — the constructor's CanInviteMembers is its entire
	// authorization — and that check used to run only when synthesizing. So an
	// invite already on file was unguarded: an outsider named it and carried on
	// into SetSubstationIdOverride, which validates rights on the destination
	// substation and says nothing about the guild, pointing the invite at a
	// substation they own. The invitee connected there on accept, handing the
	// attacker their power connection.
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)
	gs := testCreateGuild(k, ctx)
	attacker := testCreateGuild(k, ctx)

	invitee := testRegisterGuildlessPlayer(k, ctx, "consent_retarget_inv1")

	_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
		Creator:  gs.GuildOwner.Creator,
		GuildId:  gs.Guild.Id,
		PlayerId: invitee.Id,
	})
	require.NoError(t, err)

	original, found := k.GetGuildMembershipApplication(ctx, gs.Guild.Id, invitee.Id)
	require.True(t, found)

	// The attacker owns their own guild, so CanManageConnectionsBy passes on the
	// substation they are redirecting to. Only the guild-side check stops them.
	// testCreateGuild leaves byInvite at member, so CanInviteMembers refuses with
	// not_member rather than ErrPermission — pin that so a regression that
	// fails later (in SetSubstationIdOverride) cannot still pass this test.
	_, err = ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
		Creator:      attacker.GuildOwner.Creator,
		GuildId:      gs.Guild.Id,
		PlayerId:     invitee.Id,
		SubstationId: attacker.Substation.Id,
	})
	require.ErrorIs(t, err, types.ErrGuildMembership)
	require.Contains(t, err.Error(), "not a member")
	require.Contains(t, err.Error(), gs.Guild.Id)

	current, found := k.GetGuildMembershipApplication(ctx, gs.Guild.Id, invitee.Id)
	require.True(t, found)
	require.Equal(t, original.SubstationId, current.SubstationId, "outsider redirected another guild's invite")
	require.NotEqual(t, attacker.Substation.Id, current.SubstationId)
}

func TestGuildMembershipInviteAmendmentFollowsInviteAuthority(t *testing.T) {
	// Amending an invite takes the same authority as issuing one, deliberately.
	// CanInviteMembers is the entire boundary and there is no separate, stricter
	// gate for changing a record somebody else wrote, so at bypass level member —
	// where every member may invite — every member may also retarget a
	// colleague's invite. That is accepted policy rather than the bug next door:
	// TestGuildMembershipInviteCannotBeRetargetedByOutsider covers the case that
	// was a vulnerability, somebody with no invite authority over the guild at all.
	//
	// A guild that wants a narrower set of amenders has a lever, and the second
	// case is it. Under permissioned, CanInviteMembers demands
	// PermGuildMembership on the guild object, which an ordinary member does not
	// hold: PermissionCheck short-circuits for the owner and otherwise requires an
	// explicit object permission.

	t.Run("a fellow member may amend at bypass level member", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx) // testCreateGuild sets byInvite to member

		member, memberSubstation := testGuildMemberWithSubstation(k, ctx, gs, "policy_memb_amend_01")
		invitee := testRegisterGuildlessPlayer(k, ctx, "policy_memb_invitee1")

		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: invitee.Id,
		})
		require.NoError(t, err)

		_, err = ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:      member.Creator,
			GuildId:      gs.Guild.Id,
			PlayerId:     invitee.Id,
			SubstationId: memberSubstation.Id,
		})
		require.NoError(t, err, "at bypass level member every member may invite, so every member may amend")

		app, found := k.GetGuildMembershipApplication(ctx, gs.Guild.Id, invitee.Id)
		require.True(t, found)
		require.Equal(t, memberSubstation.Id, app.SubstationId)
		require.NotEqual(t, gs.Substation.Id, app.SubstationId,
			"the amendment should have moved the destination off the guild's own substation")
	})

	t.Run("a fellow member may not amend at bypass level permissioned", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		guild, found := k.GetGuild(ctx, gs.Guild.Id)
		require.True(t, found)
		guild.JoinInfusionMinimumBypassByInvite = types.GuildJoinBypassLevel_permissioned
		k.SetGuild(ctx, guild)

		member, memberSubstation := testGuildMemberWithSubstation(k, ctx, gs, "policy_perm_amend_01")
		invitee := testRegisterGuildlessPlayer(k, ctx, "policy_perm_invitee1")

		// The owner still invites under permissioned, via PermissionCheck's
		// owner short-circuit rather than an explicit grant.
		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: invitee.Id,
		})
		require.NoError(t, err)

		_, err = ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:      member.Creator,
			GuildId:      gs.Guild.Id,
			PlayerId:     invitee.Id,
			SubstationId: memberSubstation.Id,
		})
		require.Error(t, err, "permissioned is the lever a guild has against members amending invites")
		// Pin the reason, not just the refusal. The member owns the substation
		// they named, so CanManageConnectionsBy would have passed; this has to be
		// the guild-side check failing, and PermissionCheck names the object it
		// refused on.
		require.ErrorIs(t, err, types.ErrPermission)
		require.Contains(t, err.Error(), gs.Guild.Id)

		app, found := k.GetGuildMembershipApplication(ctx, gs.Guild.Id, invitee.Id)
		require.True(t, found)
		require.NotEqual(t, memberSubstation.Id, app.SubstationId)

		// And the level narrows who may amend rather than stopping amendment
		// outright, which is what makes the refusal above meaningful.
		_, err = ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:      gs.GuildOwner.Creator,
			GuildId:      gs.Guild.Id,
			PlayerId:     invitee.Id,
			SubstationId: gs.Substation.Id,
		})
		require.NoError(t, err)

		app, found = k.GetGuildMembershipApplication(ctx, gs.Guild.Id, invitee.Id)
		require.True(t, found)
		require.Equal(t, gs.Substation.Id, app.SubstationId)
	})
}

func TestGuildMembershipConsentedFlowsStillWork(t *testing.T) {
	// Positive controls. The guards are worthless if they also close the two
	// legitimate routes into a guild, and the request path in particular now
	// depends on the request handler's write being visible to the approve
	// handler that follows it.
	t.Run("request then approve", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		joiner := testRegisterGuildlessPlayer(k, ctx, "consent_flow_request1")

		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  joiner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: joiner.Id,
		})
		require.NoError(t, err)

		_, err = ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: joiner.Id,
		})
		require.NoError(t, err)

		after, _ := k.GetPlayer(ctx, joiner.Id)
		require.Equal(t, gs.Guild.Id, after.GuildId)

		_, found := k.GetGuildMembershipApplication(ctx, gs.Guild.Id, joiner.Id)
		require.False(t, found, "approved application should be cleared")
	})

	t.Run("invite then accept", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		joiner := testRegisterGuildlessPlayer(k, ctx, "consent_flow_invite01")

		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: joiner.Id,
		})
		require.NoError(t, err)

		_, err = ms.GuildMembershipInviteApprove(wctx, &types.MsgGuildMembershipInviteApprove{
			Creator:  joiner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: joiner.Id,
		})
		require.NoError(t, err)

		after, _ := k.GetPlayer(ctx, joiner.Id)
		require.Equal(t, gs.Guild.Id, after.GuildId)
	})

	t.Run("guild owner re-invites and changes the substation", func(t *testing.T) {
		// Hoisting CanInviteMembers out of the synthesis branch made it run on
		// the second invite too. Somebody who may invite must still be able to
		// amend an invite they already sent.
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		invitee := testRegisterGuildlessPlayer(k, ctx, "consent_reinvite_inv1")

		_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:  gs.GuildOwner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: invitee.Id,
		})
		require.NoError(t, err)

		_, err = ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
			Creator:      gs.GuildOwner.Creator,
			GuildId:      gs.Guild.Id,
			PlayerId:     invitee.Id,
			SubstationId: gs.Substation.Id,
		})
		require.NoError(t, err)

		app, found := k.GetGuildMembershipApplication(ctx, gs.Guild.Id, invitee.Id)
		require.True(t, found)
		require.Equal(t, gs.Substation.Id, app.SubstationId)
	})

	t.Run("request then revoke by the player", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		joiner := testRegisterGuildlessPlayer(k, ctx, "consent_flow_revoke01")

		_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
			Creator:  joiner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: joiner.Id,
		})
		require.NoError(t, err)

		_, err = ms.GuildMembershipRequestRevoke(wctx, &types.MsgGuildMembershipRequestRevoke{
			Creator:  joiner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: joiner.Id,
		})
		require.NoError(t, err)

		_, found := k.GetGuildMembershipApplication(ctx, gs.Guild.Id, joiner.Id)
		require.False(t, found)

		after, _ := k.GetPlayer(ctx, joiner.Id)
		require.Equal(t, "", after.GuildId)
	})

	t.Run("direct join still needs no stored application", func(t *testing.T) {
		// GuildMembershipJoin creates and consumes its application in one
		// handler, which is why DirectJoin is deliberately outside requirePending.
		k, ms, ctx := setupMsgServer(t)
		wctx := sdk.UnwrapSDKContext(ctx)
		gs := testCreateGuild(k, ctx)

		joiner := testRegisterGuildlessPlayer(k, ctx, "consent_flow_direct01")

		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:  joiner.Creator,
			GuildId:  gs.Guild.Id,
			PlayerId: joiner.Id,
		})
		require.NoError(t, err)

		after, _ := k.GetPlayer(ctx, joiner.Id)
		require.Equal(t, gs.Guild.Id, after.GuildId)
	})
}
