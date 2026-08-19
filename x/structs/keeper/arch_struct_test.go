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

// TestArch_StructHandlersRejectDestroyed guards the rubble window.
//
// Destroying a struct does not remove it. DestroyAndCommit adds the Destroyed
// flag, leaves Built set, and deliberately leaves the struct in its planet or
// fleet slot; only StructSweepDestroyed, StructSweepDelay blocks later, clears
// the slot and deletes the object. For those blocks the struct is fully readable
// and every handler can still load it by id.
//
// That makes "is this struct destroyed" a question every struct handler has to
// ask, and the cost of forgetting is not a failed transaction. StructActivate
// forgot: a destroyed struct still read as built and offline, so it passed every
// activation check and came back online, and GoOnline re-added the owner's load,
// the planet's shield and its defensive cannon or interceptor count. The sweep
// then deleted the struct without taking it offline again, so those contributions
// outlived the object with nothing left to reverse them.
//
// Handlers may ask directly with IsDestroyed(), or inherit the check from
// ReadinessCheck or ActivationReadinessCheck. Some are only incidentally safe
// today because they also require the struct to be online and a destroyed struct
// is offline; that is not good enough to rely on, since it evaporates the moment
// anything can bring a destroyed struct back online, which is precisely the bug
// this suite exists for.
func TestArch_StructHandlersRejectDestroyed(t *testing.T) {
	// Calls that establish the struct is not destroyed, whether directly or
	// through a shared readiness check that performs the test.
	guards := map[string]bool{
		"IsDestroyed":              true,
		"ReadinessCheck":           true,
		"ActivationReadinessCheck": true,
	}

	// Handlers that load a struct but legitimately need no guard, each with the
	// reason. Keep this list short and justified: an entry here is a handler that
	// may act on rubble.
	allowed := map[string]string{
		// Deactivation of a destroyed struct is a no-op: destruction already took
		// it offline, and GoOffline is guarded on IsOnline. Rejecting it would
		// also strand a player mid-cleanup for no benefit.
		"StructDeactivate":      "GoOffline is idempotent, so deactivating rubble changes nothing",
		"StructDeactivateBatch": "GoOffline is idempotent, so deactivating rubble changes nothing",
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

			var loadsStruct, guarded bool
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if guards[sel.Sel.Name] {
					guarded = true
					return true
				}
				// cc.GetStruct(id) pulls a struct out of the store by id. Not
				// StructCache.GetStruct(), which is the zero-argument getter for the
				// already-loaded record and says nothing about where it came from.
				if sel.Sel.Name != "GetStruct" {
					return true
				}
				recv, ok := sel.X.(*ast.Ident)
				if !ok || recv.Name != "cc" {
					return true
				}
				loadsStruct = true
				return true
			})

			if !loadsStruct {
				continue
			}
			if _, exempt := allowed[fn.Name.Name]; exempt {
				continue
			}

			handlersChecked++
			if !guarded {
				failures = append(failures, name+": "+fn.Name.Name+
					" loads a struct but never establishes it is not destroyed;"+
					" add an IsDestroyed() check or go through a readiness check")
			}
		}
	}

	// If the scan stops matching handlers the guard passes while checking nothing.
	require.GreaterOrEqual(t, handlersChecked, 10,
		"expected to find the struct handlers; the source scan is probably broken")

	require.Empty(t, failures,
		"every handler acting on a struct must reject one that is destroyed:\n  - %s",
		strings.Join(failures, "\n  - "))
}
