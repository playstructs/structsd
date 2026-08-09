package keeper_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestArch_SignerLookupDiscipline guards the split between the two player-by-
// address lookups.
//
// CurrentContext caches PlayerCache by player id, so every address of a player
// resolves to one shared object. The acting identity therefore cannot live on
// that object: it lives on the context, is written once by GetSigningPlayer, and
// is what every Layer 1 permission check reads. That only holds if handlers keep
// the two lookups straight:
//
//	GetSigningPlayer(msg.Creator)  -- the authenticated signer, at most once
//	GetPlayerByAddress(<other>)    -- a subject named by the message body
//
// Passing a caller-supplied address to GetSigningPlayer would let a weak key
// nominate a stronger address as the actor. Passing msg.Creator to
// GetPlayerByAddress leaves the operation with no signer, which fails closed but
// only at runtime. This test catches both at build time instead.
func TestArch_SignerLookupDiscipline(t *testing.T) {
	const (
		signerFunc = "GetSigningPlayer"
		lookupFunc = "GetPlayerByAddress"
	)

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

		signerCalls := 0

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}

			arg := argSource(call.Args[0])
			pos := fset.Position(call.Pos())

			switch sel.Sel.Name {
			case signerFunc:
				signerCalls++
				if arg != "msg.Creator" {
					failures = append(failures, fmt.Sprintf(
						"%s:%d: %s(%s) -- only msg.Creator may establish the signer",
						name, pos.Line, signerFunc, arg))
				}
			case lookupFunc:
				if arg == "msg.Creator" {
					failures = append(failures, fmt.Sprintf(
						"%s:%d: %s(msg.Creator) -- use %s so the operation has an acting identity",
						name, pos.Line, lookupFunc, signerFunc))
				}
			}
			return true
		})

		if signerCalls > 1 {
			failures = append(failures, fmt.Sprintf(
				"%s: calls %s %d times -- an operation has exactly one signer",
				name, signerFunc, signerCalls))
		}
		handlersChecked++
	}

	require.Greater(t, handlersChecked, 80, "expected to scan the full handler set")
	require.Empty(t, failures, "signer lookup discipline violated:\n%s", strings.Join(failures, "\n"))
}

// argSource renders the simple expressions that appear as lookup arguments
// (msg.Creator, msg.FromAddress, address, ...) for use in failure messages and
// comparisons. Anything more complex renders as "<expr>" and is treated as not
// being msg.Creator, which is the conservative direction for the signer check
// and would be caught by the runtime fail-closed behaviour anyway.
func argSource(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		if base, ok := e.X.(*ast.Ident); ok {
			return base.Name + "." + e.Sel.Name
		}
	}
	return "<expr>"
}
