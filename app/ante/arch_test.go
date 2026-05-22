package ante_test

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

// TestArch_NoUnannotatedCheckTxSkips is the architectural guard that prevents
// a regression of incident 2026-05. Every place in the app/ante package that
// short-circuits on ctx.IsCheckTx() or ctx.IsReCheckTx() MUST be accompanied
// by a // SKIP_RATIONALE: <reason> comment.
//
// The original ThrottleDecorator skipped CheckTx because "the transient store
// resets per block, so we can't reliably enforce these checks across
// CheckTx/DeliverTx without false rejections." That rationale turned out to
// be wrong — the SDK isolates CheckTx-state and DeliverTx-state transient
// stores. The comment existed, but it was wrong AND vague, and it was easy
// for reviewers to miss the implications. This test does not validate the
// rationale (humans must do that), but it forces every skip to be visible
// at code-review time. Combined with the invariant property test, this
// closes the loop.
//
// Files exempt from this check (legitimate CheckTx-only decorators that
// inherently short-circuit on non-CheckTx phases): CheckTxThrottleDecorator
// and ConditionalMempoolFeeDecorator. These are added to allowedFiles below
// with their own rationale comment in the source.
func TestArch_NoUnannotatedCheckTxSkips(t *testing.T) {
	const skipMarker = "SKIP_RATIONALE:"

	allowedFiles := map[string]string{
		// These files only run during CheckTx (mempool-only enforcement).
		// They use !ctx.IsCheckTx() to short-circuit, which is the inverse
		// pattern and intentionally CheckTx-only.
		"checktx_throttle.go":        "CheckTx-only by design (mempool flood defense)",
		"conditional_mempool_fee.go": "mempool min-fee policy is CheckTx-only by design",
	}

	wd, err := os.Getwd()
	require.NoError(t, err)
	pkgDir := wd

	entries, err := os.ReadDir(pkgDir)
	require.NoError(t, err)

	fset := token.NewFileSet()
	var failures []string

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		path := filepath.Join(pkgDir, name)
		src, err := os.ReadFile(path)
		require.NoError(t, err)

		file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		require.NoError(t, err)

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			fnName := sel.Sel.Name
			if fnName != "IsCheckTx" && fnName != "IsReCheckTx" {
				return true
			}

			pos := fset.Position(call.Pos())
			if reason, allowed := allowedFiles[name]; allowed {
				found := false
				for _, cg := range file.Comments {
					if cg.Pos() > call.End() {
						continue
					}
					if cg.End() < call.Pos()-token.Pos(800) {
						continue
					}
					for _, c := range cg.List {
						if strings.Contains(c.Text, skipMarker) {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				if !found {
					failures = append(failures, formatFail(name, pos.Line, fnName,
						"file is on the allowlist ("+reason+") but no `// SKIP_RATIONALE:` comment was found nearby"))
				}
				return true
			}

			found := false
			for _, cg := range file.Comments {
				if cg.End() < call.Pos()-token.Pos(800) {
					continue
				}
				if cg.Pos() > call.End()+token.Pos(200) {
					continue
				}
				for _, c := range cg.List {
					if strings.Contains(c.Text, skipMarker) {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				failures = append(failures, formatFail(name, pos.Line, fnName,
					"use of ctx."+fnName+"() requires an adjacent `// SKIP_RATIONALE: <reason>` comment, "+
						"OR the file must be added to allowedFiles in arch_test.go with justification"))
			}

			return true
		})
	}

	if len(failures) > 0 {
		t.Fatalf("architectural guard violated (%d):\n  - %s\n\n"+
			"This guard prevents a regression of incident 2026-05 where a CheckTx skip in "+
			"ThrottleDecorator silently admitted invalid txs into the mempool. Every "+
			"CheckTx/ReCheckTx short-circuit must be visible at review time.",
			len(failures), strings.Join(failures, "\n  - "))
	}
}

func formatFail(file string, line int, fnName, msg string) string {
	return file + ":" + itoa(line) + ": " + fnName + ": " + msg
}

// itoa avoids pulling strconv just for line numbers in a test error message.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
