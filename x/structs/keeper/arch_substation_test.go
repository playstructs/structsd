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

/* TestArch_HandlersResolveMessageSubstationIdsThroughAnExistenceCheck is the
 * substation form of the rule GetExistingPlayer exists for.
 *
 * cc.GetSubstation is a cache allocator. It builds a SubstationCache around any
 * string and reads nothing, so it says nothing about whether a substation is
 * there. That is correct for an id read off something already loaded from state
 * - a provider's substation, a guild's entry substation - and wrong for one a
 * transaction supplied.
 *
 * The permission check is not a backstop. Object permissions are keyed by the
 * raw id string, and every player holds PermAll on their own player id, so
 * ProviderCreate accepted a player id as a substation and passed
 * CanAllocateAsSourceBy against a substation that did not exist. Worse than a
 * nil: grid attribute ids are derived from the object id alone, so the provider
 * sold capacity backed by the player's own capacity and load counters.
 *
 * Any of the three establishes existence: GetExistingSubstation, which also
 * checks the id's type, or CheckSubstation / LoadSubstation on the cache.
 */
func TestArch_HandlersResolveMessageSubstationIdsThroughAnExistenceCheck(t *testing.T) {
	guards := map[string]bool{
		"GetExistingSubstation": true,
		"CheckSubstation":       true,
		"LoadSubstation":        true,
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

			var resolvesMessageId, guarded bool

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if guards[sel.Sel.Name] {
					guarded = true
					return true
				}
				if sel.Sel.Name != "GetSubstation" {
					return true
				}
				recv, ok := sel.X.(*ast.Ident)
				if !ok || recv.Name != "cc" {
					return true
				}
				// Only an id the message supplied. An id read off a loaded
				// object exists by construction.
				for _, arg := range call.Args {
					argSel, ok := arg.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					if argIdent, ok := argSel.X.(*ast.Ident); ok && argIdent.Name == "msg" {
						resolvesMessageId = true
					}
				}
				return true
			})

			if !resolvesMessageId {
				continue
			}

			handlersChecked++
			if !guarded {
				failures = append(failures, name+": "+fn.Name.Name+
					" resolves a message-supplied substation id but never establishes the substation exists;"+
					" use cc.GetExistingSubstation")
			}
		}
	}

	// If the scan stops matching handlers the guard passes while checking nothing.
	require.GreaterOrEqual(t, handlersChecked, 8,
		"expected to find the substation handlers; the source scan is probably broken")

	require.Empty(t, failures,
		"a substation id a transaction chose has to be resolved, not merely allocated:\n  - %s",
		strings.Join(failures, "\n  - "))
}
