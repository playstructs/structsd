package keeper_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Regression suite for the guild join bypass level.
//
// guildJoinBypassLevel declares three values, but proto3 enums are open: the
// generated decoder shifts bytes into an int32 and never consults the enum. A
// player holding only PermGuildJoinConstraintsUpdate could therefore store 500,
// and the three switches that read the level had no default, so an unmatched
// value left err nil and every membership gate opened. An outsider could then
// submit and approve their own membership request, which is PermGuildMembership
// authority obtained from a bit that is not PermGuildMembership.
//
// The fix has two halves and both are covered here: the readers deny an
// undeclared level, which also covers records already on disk, and the writers
// refuse to store one.

// undeclaredBypassLevel is outside the declared enum. This is the value the
// pre-v0.21.0 handler tests asserted was persisted successfully.
const undeclaredBypassLevel = types.GuildJoinBypassLevel(500)

// testPoisonBypassLevels writes both bypass fields straight to the store,
// simulating a record written before the update handlers validated their input.
// It deliberately bypasses GuildCache, which now refuses these values.
func testPoisonBypassLevels(t *testing.T, k keeperlib.Keeper, ctx context.Context, guildId string, level types.GuildJoinBypassLevel) {
	t.Helper()

	guildObj, found := k.GetGuild(ctx, guildId)
	require.True(t, found, "guild %s should exist", guildId)
	guildObj.JoinInfusionMinimumBypassByRequest = level
	guildObj.JoinInfusionMinimumBypassByInvite = level
	k.SetGuild(ctx, guildObj)
}

// TestGuildBypassLevel_UndeclaredLevelBlocksSelfApproval is the exploit itself.
// The request is filed while the guild holds a legitimate level, because the
// attack only needs the approval leg to fail open; the level is then poisoned
// and the outsider tries to wave their own request through.
func TestGuildBypassLevel_UndeclaredLevelBlocksSelfApproval(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)
	gs := testCreateGuild(k, ctx)

	outsiderAcc := sdk.AccAddress("bypass_outsider_addr")
	outsider := types.Player{
		Creator:        outsiderAcc.String(),
		PrimaryAddress: outsiderAcc.String(),
	}
	outsider = testAppendPlayer(k, ctx, outsider)

	// testCreateGuild leaves both levels at member, so the request is filed
	// legitimately before anything is poisoned.
	_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
		Creator:  outsider.Creator,
		GuildId:  gs.Guild.Id,
		PlayerId: outsider.Id,
	})
	require.NoError(t, err, "filing the request should succeed under the member level")

	testPoisonBypassLevels(t, k, wctx, gs.Guild.Id, undeclaredBypassLevel)

	_, err = ms.GuildMembershipRequestApprove(wctx, &types.MsgGuildMembershipRequestApprove{
		Creator:  outsider.Creator,
		GuildId:  gs.Guild.Id,
		PlayerId: outsider.Id,
	})
	require.Error(t, err, "an undeclared bypass level must not authorize self-approval")
	require.Contains(t, err.Error(), "invalid_bypass_level")

	// The whole point: ApproveRequest calls MigrateGuild, so a fail-open switch
	// here is unauthorized membership, not just a confusing error.
	stored, found := k.GetPlayer(wctx, outsider.Id)
	require.True(t, found)
	require.Empty(t, stored.GuildId, "outsider must not have been migrated into the guild")
}

// TestGuildBypassLevel_UndeclaredLevelBlocksRequest covers the other reader on
// the request path. Under the old code an undeclared level behaved like an open
// guild here, because only closed set an error.
func TestGuildBypassLevel_UndeclaredLevelBlocksRequest(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)
	gs := testCreateGuild(k, ctx)

	requesterAcc := sdk.AccAddress("bypass_requester_adr")
	requester := types.Player{
		Creator:        requesterAcc.String(),
		PrimaryAddress: requesterAcc.String(),
	}
	requester = testAppendPlayer(k, ctx, requester)

	testPoisonBypassLevels(t, k, wctx, gs.Guild.Id, undeclaredBypassLevel)

	_, err := ms.GuildMembershipRequest(wctx, &types.MsgGuildMembershipRequest{
		Creator:  requester.Creator,
		GuildId:  gs.Guild.Id,
		PlayerId: requester.Id,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid_bypass_level")
}

// TestGuildBypassLevel_UndeclaredLevelBlocksInvite covers the invite reader,
// where an undeclared level let any player with a registered address invite
// anyone into a guild they have no standing in.
func TestGuildBypassLevel_UndeclaredLevelBlocksInvite(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)
	gs := testCreateGuild(k, ctx)

	outsiderAcc := sdk.AccAddress("bypass_inviter_addr0")
	outsider := types.Player{
		Creator:        outsiderAcc.String(),
		PrimaryAddress: outsiderAcc.String(),
	}
	outsider = testAppendPlayer(k, ctx, outsider)

	targetAcc := sdk.AccAddress("bypass_invitee_addr0")
	target := types.Player{
		Creator:        targetAcc.String(),
		PrimaryAddress: targetAcc.String(),
	}
	target = testAppendPlayer(k, ctx, target)

	testPoisonBypassLevels(t, k, wctx, gs.Guild.Id, undeclaredBypassLevel)

	_, err := ms.GuildMembershipInvite(wctx, &types.MsgGuildMembershipInvite{
		Creator:  outsider.Creator,
		GuildId:  gs.Guild.Id,
		PlayerId: target.Id,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid_bypass_level")
}

// TestGuildBypassLevel_DeclaredLevelsKeepTheirPolicy pins what the three
// declared levels do, so the new default branches are shown to deny only the
// undeclared case rather than tightening real policy.
func TestGuildBypassLevel_DeclaredLevelsKeepTheirPolicy(t *testing.T) {
	testCases := []struct {
		name  string
		level types.GuildJoinBypassLevel

		requestAllowed        bool
		ownerApproveAllowed   bool
		strangerApproveAllows bool
		ownerInviteAllowed    bool
		strangerInviteAllowed bool
	}{
		{
			name:  "closed",
			level: types.GuildJoinBypassLevel_closed,
		},
		{
			name:  "permissioned",
			level: types.GuildJoinBypassLevel_permissioned,
			// Anyone may ask; only PermGuildMembership on the guild approves.
			requestAllowed:      true,
			ownerApproveAllowed: true,
			ownerInviteAllowed:  true,
		},
		{
			name:  "member",
			level: types.GuildJoinBypassLevel_member,
			// Anyone may ask; any member approves, and the stranger is not one.
			requestAllowed:      true,
			ownerApproveAllowed: true,
			ownerInviteAllowed:  true,
		},
		{
			name:  "undeclared",
			level: undeclaredBypassLevel,
			// Everything denied, including for the guild owner. A level nobody
			// wrote policy for grants nothing to anybody.
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k, _, ctx := setupMsgServer(t)
			wctx := sdk.UnwrapSDKContext(ctx)
			gs := testCreateGuild(k, ctx)

			strangerAcc := sdk.AccAddress("bypass_stranger_ad" + tc.name[:2])
			stranger := types.Player{
				Creator:        strangerAcc.String(),
				PrimaryAddress: strangerAcc.String(),
			}
			stranger = testAppendPlayer(k, ctx, stranger)

			testPoisonBypassLevels(t, k, wctx, gs.Guild.Id, tc.level)

			// Both actors are loaded through GetSigningPlayer, because
			// PermissionCheck reads the signing address off the CurrentContext
			// and a stranger's primary address holds PermAll — which is exactly
			// why the address-level check cannot be what stops this.
			ownerCC := k.NewCurrentContext(wctx)
			owner, err := ownerCC.GetSigningPlayer(gs.GuildOwner.Creator)
			require.NoError(t, err)
			ownerGuild := ownerCC.GetGuild(gs.Guild.Id)
			require.True(t, ownerGuild.LoadGuild())

			requireAllowed(t, tc.requestAllowed, ownerGuild.CanRequestMembership(), "CanRequestMembership")
			requireAllowed(t, tc.ownerApproveAllowed, ownerGuild.CanApproveMembershipRequest(owner), "owner CanApproveMembershipRequest")
			requireAllowed(t, tc.ownerInviteAllowed, ownerGuild.CanInviteMembers(owner), "owner CanInviteMembers")

			strangerCC := k.NewCurrentContext(wctx)
			strangerPlayer, err := strangerCC.GetSigningPlayer(stranger.Creator)
			require.NoError(t, err)
			strangerGuild := strangerCC.GetGuild(gs.Guild.Id)
			require.True(t, strangerGuild.LoadGuild())

			requireAllowed(t, tc.strangerApproveAllows, strangerGuild.CanApproveMembershipRequest(strangerPlayer), "stranger CanApproveMembershipRequest")
			requireAllowed(t, tc.strangerInviteAllowed, strangerGuild.CanInviteMembers(strangerPlayer), "stranger CanInviteMembers")
		})
	}
}

func requireAllowed(t *testing.T, allowed bool, err error, what string) {
	t.Helper()

	if allowed {
		require.NoError(t, err, "%s should be allowed", what)
		return
	}
	require.Error(t, err, "%s should be denied", what)
}

// TestGuildBypassLevel_SetterRejectsUndeclaredLevel covers the write side at the
// cache, which is the choke point every transaction goes through. Rejecting
// before the assignment is what makes the handler's ordering uninteresting.
func TestGuildBypassLevel_SetterRejectsUndeclaredLevel(t *testing.T) {
	k, _, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)
	gs := testCreateGuild(k, ctx)

	cc := k.NewCurrentContext(wctx)
	guild := cc.GetGuild(gs.Guild.Id)
	require.True(t, guild.LoadGuild())

	before := guild.GetGuild().JoinInfusionMinimumBypassByRequest

	require.ErrorIs(t, guild.SetJoinInfusionMinimumBypassByRequest(undeclaredBypassLevel), types.ErrInvalidGuildJoinBypassLevel)
	require.ErrorIs(t, guild.SetJoinInfusionMinimumBypassByInvite(undeclaredBypassLevel), types.ErrInvalidGuildJoinBypassLevel)
	require.Equal(t, before, guild.GetGuild().JoinInfusionMinimumBypassByRequest, "a rejected level must not reach the cache")

	require.NoError(t, guild.SetJoinInfusionMinimumBypassByRequest(types.GuildJoinBypassLevel_permissioned))
	require.NoError(t, guild.SetJoinInfusionMinimumBypassByInvite(types.GuildJoinBypassLevel_closed))
	cc.CommitAll()

	stored, found := k.GetGuild(wctx, gs.Guild.Id)
	require.True(t, found)
	require.Equal(t, types.GuildJoinBypassLevel_permissioned, stored.JoinInfusionMinimumBypassByRequest)
	require.Equal(t, types.GuildJoinBypassLevel_closed, stored.JoinInfusionMinimumBypassByInvite)
}

// TestArch_GuildBypassSwitchesHaveDefault keeps the readers fail-closed.
//
// Scoped to this one enum on purpose. Other enums in the keeper (Ambit,
// ObjectType) are switched on for dispatch rather than authorization, and they
// carry their own defaultless-switch backlog; this is the enum where falling
// through a switch is a permission grant.
func TestArch_GuildBypassSwitchesHaveDefault(t *testing.T) {
	const path = "guild_cache.go"

	source, err := os.ReadFile(path)
	require.NoError(t, err, "reading %s", path)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
	require.NoError(t, err, "parsing %s", path)

	var found int

	ast.Inspect(file, func(node ast.Node) bool {
		switchStmt, ok := node.(*ast.SwitchStmt)
		if !ok || switchStmt.Body == nil {
			return true
		}

		var switchesOnBypassLevel, hasDefault bool

		for _, statement := range switchStmt.Body.List {
			clause, isClause := statement.(*ast.CaseClause)
			if !isClause {
				continue
			}
			if clause.List == nil {
				hasDefault = true
				continue
			}
			for _, expression := range clause.List {
				selector, isSelector := expression.(*ast.SelectorExpr)
				if !isSelector {
					continue
				}
				pkg, isIdent := selector.X.(*ast.Ident)
				if isIdent && pkg.Name == "types" && strings.HasPrefix(selector.Sel.Name, "GuildJoinBypassLevel_") {
					switchesOnBypassLevel = true
				}
			}
		}

		if switchesOnBypassLevel {
			found++
			require.True(t, hasDefault,
				"the switch on a guild join bypass level at %s has no default clause. "+
					"proto3 enums are open, so an undeclared value falls through and leaves the "+
					"error nil, which is a membership grant. Add a default that denies.",
				fset.Position(switchStmt.Pos()))
		}

		return true
	})

	require.Equal(t, 3, found,
		"expected the three bypass-level switches in %s (CanInviteMembers, "+
			"CanApproveMembershipRequest, CanRequestMembership); found %d. "+
			"If a reader was added or removed, update this count deliberately.",
		path, found)
}
