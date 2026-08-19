package keeper_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestArch_MembershipTransitionsRequirePendingApplication keeps the two
// membership application loaders apart.
//
// Guild membership is two-sided: a player files a request and the guild approves
// it, or the guild issues an invite and the player accepts it. Each leg has its
// own check, and the stored application is the only evidence the first leg ever
// happened. A single loader that synthesized a proposed application on a store
// miss therefore handed every approval path exactly the consent it was about to
// verify. On GuildMembershipRequestApprove that was a force-join: any member of a
// recruiting guild could name any player and have ApproveRequest overwrite their
// GuildId, reset their GuildRank and move their substation connection, with the
// victim never involved.
//
// So the loader is split. GetOrCreateGuildMembershipApplicationCache belongs to
// the handlers that open an application; GetPendingGuildMembershipApplicationCache
// belongs to everything that acts on one somebody else filed, and refuses when
// the store has nothing. This test fails a handler that pairs the creating loader
// with a terminal transition.
//
// GuildMembershipApplicationCache.requirePending is the runtime backstop and
// would catch such a handler in production. This test is what catches it in
// review, with the whole reason attached, which is the difference between a rule
// people follow and a rule people rediscover.
func TestArch_MembershipTransitionsRequirePendingApplication(t *testing.T) {
	// The transitions that consume an application somebody else filed. Each is
	// guarded by requirePending, so reaching one from a synthesized record is a
	// contradiction rather than a vulnerability today — but it is still a handler
	// that believes it is approving something real.
	terminal := map[string]bool{
		"ApproveRequest": true,
		"DenyRequest":    true,
		"RevokeRequest":  true,
		"ApproveInvite":  true,
		"DenyInvite":     true,
		"RevokeInvite":   true,
	}

	// Handlers that legitimately create and consume an application in one go,
	// each with the reason. An entry here is a handler acting on a record it
	// invented, so it must carry its own proof of consent.
	allowed := map[string]string{
		// GuildMembershipJoin creates the application and calls DirectJoin on it
		// immediately; there is never a stored row. Consent is not in question
		// because VerifyDirectJoin demands PermGuildMembership on the joining
		// player themselves, and DirectJoin is deliberately outside
		// requirePending for that reason.
		"GuildMembershipJoin": "creates and consumes in one handler; VerifyDirectJoin checks the joining player's own permission",
		// A kick is initiated by the guild and there is nothing for the member to
		// have filed. It goes through GetGuildMembershipKickCache, not either of
		// the loaders this test watches, and authorizes itself with
		// CanKickMembers plus a rank comparison plus refusing the owner.
		"GuildMembershipKick": "a kick has no application to find; CanKickMembers and the rank comparison are its authorization",
	}

	wd, err := os.Getwd()
	require.NoError(t, err)

	entries, err := os.ReadDir(wd)
	require.NoError(t, err)

	fset := token.NewFileSet()
	var failures []string
	pendingLoaders := 0
	creatingLoaders := 0

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "msg_server_guild_membership") || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(wd, name), nil, 0)
		require.NoError(t, err)

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Body == nil {
				continue
			}

			var creates, pends bool
			var transitions []string

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				switch sel.Sel.Name {
				case "GetOrCreateGuildMembershipApplicationCache":
					creates = true
				case "GetPendingGuildMembershipApplicationCache":
					pends = true
				default:
					if terminal[sel.Sel.Name] {
						transitions = append(transitions, sel.Sel.Name)
					}
				}
				return true
			})

			if pends {
				pendingLoaders++
			}
			if creates {
				creatingLoaders++
			}

			if !creates || len(transitions) == 0 {
				continue
			}
			if _, exempt := allowed[fn.Name.Name]; exempt {
				continue
			}

			failures = append(failures, name+": "+fn.Name.Name+
				" builds its application with GetOrCreateGuildMembershipApplicationCache and then calls "+
				strings.Join(transitions, ", ")+
				"; a path that consumes an application must load it with GetPendingGuildMembershipApplicationCache"+
				" so a missing one is refused rather than invented")
		}
	}

	// The substantive check goes first. A handler switched to the wrong loader
	// also drops the pending-loader count below its floor, and require aborts on
	// the first failure, so asserting the counts first would report "the source
	// scan is probably broken" for a violation the next line names precisely.
	require.Empty(t, failures,
		"a membership transition may not act on an application it synthesized:\n  - %s",
		strings.Join(failures, "\n  - "))

	// If the scan stops matching handlers the guard passes while checking
	// nothing. Six consumption handlers take the pending loader; request, invite
	// and join take the creating one.
	require.GreaterOrEqual(t, pendingLoaders, 6,
		"expected the six membership consumption handlers; the source scan is probably broken")
	require.GreaterOrEqual(t, creatingLoaders, 3,
		"expected the membership creation handlers; the source scan is probably broken")
}
