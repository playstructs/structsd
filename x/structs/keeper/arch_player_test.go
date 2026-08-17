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

// TestArch_HandlersResolveMessagePlayerIdsThroughGetExistingPlayer keeps a
// transaction-supplied player id from reaching a cache that cannot say whether
// that player is real.
//
// CurrentContext.GetPlayer is a cache allocator, like every other context getter:
// it never reads the store, so it cannot report existence and has no error return
// to check. It used to have one that was always nil, and twenty-eight callers
// guarded on it. Those guards read exactly like existence checks and were dead
// code, which is how AllocationTransfer came to hand an allocation to any string a
// message named: the guard it relied on returned a NewPlayerRequiredError from a
// branch the compiler could prove unreachable.
//
// A cache for a player who is not there is worse than a nil pointer, because it
// behaves. It loads as a zero value, and since a failed load leaves PlayerLoaded
// false, every getter reloads and silently reverts whatever the handler wrote.
// What survives is anything written elsewhere under the id the message chose, and
// a commit against Player.Id "", which the KV store panics on.
//
// So handlers resolve message-supplied ids with GetExistingPlayer, which loads and
// returns ObjectNotFound. GetPlayer remains correct for an id read off an object
// already loaded from state — an owner, a controller, a substation's connection
// list — because that id exists by construction.
func TestArch_HandlersResolveMessagePlayerIdsThroughGetExistingPlayer(t *testing.T) {
	// Deliberately empty. An entry here is a handler that mutates or authorizes
	// against a player it has not established exists, and needs the reason
	// written down next to it.
	allowed := map[string]string{}

	wd, err := os.Getwd()
	require.NoError(t, err)

	entries, err := os.ReadDir(wd)
	require.NoError(t, err)

	fset := token.NewFileSet()
	var failures []string
	checkedLoaders := 0

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

			// Locals carrying a message field, so that routing msg.PlayerId
			// through a variable — which is what a loop over a repeated field
			// does — is not a way around the rule.
			tainted := map[string]bool{}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch s := n.(type) {
				case *ast.AssignStmt:
					for i, rhs := range s.Rhs {
						if i < len(s.Lhs) && rootsAtMsg(rhs) {
							if id, ok := s.Lhs[i].(*ast.Ident); ok {
								tainted[id.Name] = true
							}
						}
					}
				case *ast.RangeStmt:
					if rootsAtMsg(s.X) {
						if id, ok := s.Value.(*ast.Ident); ok {
							tainted[id.Name] = true
						}
					}
				}
				return true
			})

			resolvesMessagePlayer := false
			passesMessageController := false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				if sel.Sel.Name == "GetExistingPlayer" {
					checkedLoaders++
					if len(call.Args) == 1 {
						arg := call.Args[0]
						id, isIdent := arg.(*ast.Ident)
						if rootsAtMsg(arg) || isIdent && tainted[id.Name] {
							resolvesMessagePlayer = true
						}
					}
					return true
				}

				if sel.Sel.Name == "NewAllocation" && len(call.Args) > 4 {
					controller := call.Args[4]
					id, isIdent := controller.(*ast.Ident)
					if rootsAtMsg(controller) || isIdent && tainted[id.Name] {
						passesMessageController = true
					}
				}

				// One argument is what distinguishes the CurrentContext getter
				// from Keeper.GetPlayer(ctx, id), which reports found, and from
				// the no-argument GetPlayer on the various caches.
				if sel.Sel.Name != "GetPlayer" || len(call.Args) != 1 {
					return true
				}
				if _, exempt := allowed[fn.Name.Name]; exempt {
					return true
				}

				arg := call.Args[0]
				id, isIdent := arg.(*ast.Ident)
				switch {
				case rootsAtMsg(arg):
					failures = append(failures, name+": "+fn.Name.Name+
						" passes a message field straight to cc.GetPlayer")
				case isIdent && tainted[id.Name]:
					failures = append(failures, name+": "+fn.Name.Name+
						" passes "+id.Name+", which holds a message field, to cc.GetPlayer")
				}
				return true
			})

			if passesMessageController && !resolvesMessagePlayer {
				failures = append(failures, name+": "+fn.Name.Name+
					" passes a message-supplied controller to cc.NewAllocation without GetExistingPlayer")
			}
		}
	}

	// The substantive check first: a violation also drags the loader count below
	// its floor, and require aborts on the first failure, so asserting the count
	// ahead of this would blame the scan for something the next line names.
	require.Empty(t, failures,
		"a player id chosen by a transaction must be resolved with cc.GetExistingPlayer, which fails when that player does not exist:\n  - %s",
		strings.Join(failures, "\n  - "))

	// Without a floor the scan could stop matching handlers and pass while
	// checking nothing. Twelve handlers take a player id from a message today.
	require.GreaterOrEqual(t, checkedLoaders, 10,
		"expected the handlers that resolve a message-supplied player id; the source scan is probably broken")
}

// rootsAtMsg reports whether an expression reads a field of the handler's msg,
// including through further field access or an index, as msg.PlayerId[0] does.
func rootsAtMsg(expr ast.Expr) bool {
	for {
		switch e := expr.(type) {
		case *ast.SelectorExpr:
			if id, ok := e.X.(*ast.Ident); ok {
				return id.Name == "msg"
			}
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.StarExpr:
			expr = e.X
		case *ast.ParenExpr:
			expr = e.X
		default:
			return false
		}
	}
}
