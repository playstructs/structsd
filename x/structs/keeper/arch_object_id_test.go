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

/* TestArch_HandlersResolveMessageObjectIdsThroughAnExistenceCheck generalizes
 * the rule GetExistingPlayer exists for to the other objects a message can name.
 *
 * These context getters are cache allocators. They build a cache around any
 * string and read nothing, so they say nothing about whether the object is
 * there. That is correct for an id taken off something already loaded from state
 * - a provider's substation, an agreement's provider - and wrong for one a
 * transaction supplied.
 *
 * The permission check is not a backstop. Object permissions are keyed by the
 * raw id string with no type namespacing, and registration grants every player
 * PermAll on their own player id, so submitting that id collides with a record
 * that genuinely exists and the check passes against an object that does not.
 * ProviderCreate sold capacity backed by the player's own grid counters that
 * way; the provider update handlers reached a commit on a zero-valued record.
 *
 * That commit happens to panic in the KV store on an empty id, which reads like
 * a refusal and is not one: the same shape wrote real state in
 * AllocationTransfer, where the write was keyed by something non-empty.
 *
 * Any of the three establishes existence: the GetExisting* resolver, which also
 * checks the id's type namespace, or Check* / Load* on the cache.
 */
func TestArch_HandlersResolveMessageObjectIdsThroughAnExistenceCheck(t *testing.T) {
	// Each allocator, with the calls that turn it into a resolution.
	kinds := []struct {
		takers   []string
		guards   map[string]bool
		resolver string
	}{
		{
			// Either name counts as naming the object; only the guards decide
			// whether it was resolved. Counting both keeps the floor below
			// meaningful, since a handler that adopts the resolver stops
			// mentioning the allocator at all.
			takers: []string{"GetSubstation", "GetExistingSubstation"},
			guards: map[string]bool{
				"GetExistingSubstation": true,
				"CheckSubstation":       true,
				"LoadSubstation":        true,
			},
			resolver: "cc.GetExistingSubstation",
		},
		{
			takers: []string{"GetProvider", "GetExistingProvider"},
			guards: map[string]bool{
				"GetExistingProvider": true,
				"CheckProvider":       true,
				"LoadProvider":        true,
			},
			resolver: "cc.GetExistingProvider",
		},
	}

	wd, err := os.Getwd()
	require.NoError(t, err)

	entries, err := os.ReadDir(wd)
	require.NoError(t, err)

	fset := token.NewFileSet()
	var failures []string
	handlersChecked := 0

	for _, kind := range kinds {
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
					// Not exclusive: the resolver both names the id and
					// guards it, so this must not short-circuit.
					if kind.guards[sel.Sel.Name] {
						guarded = true
					}

					named := false
					for _, taker := range kind.takers {
						if sel.Sel.Name == taker {
							named = true
							break
						}
					}
					if !named {
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
						" names a message-supplied object id"+
						" but never establishes the object exists; use "+kind.resolver)
				}
			}
		}
	}

	// If the scan stops matching handlers the guard passes while checking nothing.
	require.GreaterOrEqual(t, handlersChecked, 16,
		"expected to find the substation and provider handlers; the source scan is probably broken")

	require.Empty(t, failures,
		"an object id a transaction chose has to be resolved, not merely allocated:\n  - %s",
		strings.Join(failures, "\n  - "))
}
