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

// TestArch_StakingMutationsRequireAnAssetBit is the general form of the bug
// GuildMembershipJoin carried: a handler that moves somebody's stake while
// asking only for permission to do something else.
//
// The staking keeper's mutating calls take the delegator address as an argument
// and authorize nothing themselves - no signature over that address is required,
// and none of these messages name it as a signer. Whether the caller may move
// that stake is therefore entirely up to the handler, and the answer has to be
// an asset bit: PermTokenMigrate for a redelegation, PermTokenDefuse for an
// unbonding, matching what ReactorBeginMigration and ReactorDefuse demand for
// the identical calls.
//
// GuildMembershipJoin demanded PermGuildMembership and nothing else, and then
// redelegated on the infusion address's behalf as a side effect of the join. An
// access check is not a spend check: membership permission says who may join the
// guild, not whose stake may move to its validator. Because PermissionCheck's
// Layer 1 tests the signing key's own bits before any owner shortcut, one of
// these calls is what stops both a restricted associated address moving its own
// player's stake and an outsider moving somebody else's.
//
// The ante's PermissionMap is not a substitute. A join only migrates anything
// when an infusion's reactor sits outside the destination guild, so the bit
// cannot be demanded of every join without breaking the ordinary one - which is
// exactly the case the rule covers: fixed part in the map, policy part here.
func TestArch_StakingMutationsRequireAnAssetBit(t *testing.T) {
	// Staking keeper calls that move a delegator's bonded stake.
	mutations := map[string]string{
		"BeginRedelegation":         "redelegates the delegator's stake to another validator",
		"Undelegate":                "unbonds the delegator's stake",
		"CancelUnbondingDelegation": "returns the delegator's unbonding stake to a validator",
	}

	// Calls that establish the caller holds an asset permission over the player
	// whose stake is moving.
	assetGuards := map[string]bool{
		"CanMigrateTokensBy": true,
		"CanDefuseTokensBy":  true,
		"CanInfuseTokensBy":  true,
	}

	wd, err := os.Getwd()
	require.NoError(t, err)

	entries, err := os.ReadDir(wd)
	require.NoError(t, err)

	fset := token.NewFileSet()
	var failures []string
	handlersChecked := 0

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "msg_server_") || !strings.HasSuffix(name, ".go") ||
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

			var mutation string
			var guarded bool

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if assetGuards[sel.Sel.Name] {
					guarded = true
					return true
				}
				if why, isMutation := mutations[sel.Sel.Name]; isMutation {
					// Only the staking keeper's own methods count. A local
					// helper sharing the name is not this.
					if recv, ok := sel.X.(*ast.SelectorExpr); ok && recv.Sel.Name == "stakingKeeper" {
						mutation = sel.Sel.Name + " " + why
					}
				}
				return true
			})

			if mutation == "" {
				continue
			}

			handlersChecked++
			if !guarded {
				failures = append(failures, name+": "+fn.Name.Name+" "+mutation+
					" but never asks for an asset permission;"+
					" add a CanMigrateTokensBy/CanDefuseTokensBy check over the player whose stake moves")
			}
		}
	}

	// If the scan stops matching handlers the guard passes while checking nothing.
	require.GreaterOrEqual(t, handlersChecked, 3,
		"expected to find the staking handlers; the source scan is probably broken")

	require.Empty(t, failures,
		"a handler that moves a delegator's stake must demand an asset bit for it:\n  - %s",
		strings.Join(failures, "\n  - "))
}
