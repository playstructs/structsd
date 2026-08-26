package types_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// transcendentalMathCalls are the standard library math functions whose results
// are not required to be identical across implementations. IEEE 754 pins the
// basic operations and Sqrt to a correctly rounded result; it says nothing about
// these, and Go ships assembly versions on some architectures and the portable
// algorithm on others.
var transcendentalMathCalls = map[string]string{
	"Log":   "logarithm",
	"Log10": "logarithm",
	"Log2":  "logarithm",
	"Log1p": "logarithm",
	"Exp":   "exponential",
	"Exp2":  "exponential",
	"Expm1": "exponential",
	"Pow":   "power",
	"Pow10": "power",
	"Cbrt":  "root",
	"Hypot": "root",
	"Sin":   "trigonometric",
	"Cos":   "trigonometric",
	"Tan":   "trigonometric",
	"Asin":  "trigonometric",
	"Acos":  "trigonometric",
	"Atan":  "trigonometric",
	"Atan2": "trigonometric",
	"Sinh":  "trigonometric",
	"Cosh":  "trigonometric",
	"Tanh":  "trigonometric",
	"Gamma": "special function",
	"Erf":   "special function",
}

/* TestArch_NoTranscendentalFloatInConsensusPaths is the float sibling of
 * TestArch_NoToolchainUnicodeInConsensusPaths, and it exists for the same reason:
 * consensus code may not ask the toolchain a question the toolchain is allowed
 * to answer differently.
 *
 * Proof-of-work difficulty used to be
 *
 *     64 - int(math.Log10(age)/math.Log10(range)*63)
 *
 * and math.Log10 is implemented in assembly on amd64 and s390x and in portable
 * Go elsewhere. Those implementations disagree in the last bit, and the int()
 * truncation turns a last-bit disagreement into a whole leading hexadecimal
 * zero: the same proof, for the same block and the same planet, is accepted by
 * one validator and rejected by another. That is an app hash split rather than a
 * wrong answer, and no test of the function's output on one machine can catch
 * it. The amd64/arm64 agreement that held at the time was a coincidence of Go's
 * implementation, not a guarantee - a Go release or a new architecture backend
 * would have broken it with no change to this repository at all.
 *
 * CalculateDifficulty is integer-only now. This keeps the next one out.
 *
 * Two ways past it for a legitimate use: an adjacent // DETERMINISM_OK: <reason>
 * comment, or an entry in allowedFiles for a file wholly outside the state
 * machine.
 */
func TestArch_NoTranscendentalFloatInConsensusPaths(t *testing.T) {
	// Files exempt, each with the reason a differing float result cannot reach
	// state. Keep this short.
	allowedFiles := map[string]string{
		// The CLI mines proofs locally and prints progress. It submits the
		// result to the chain, which re-derives the requirement itself, so a
		// disagreement here costs the miner a rejected transaction rather than
		// the network a split.
		"x/structs/module/autocli.go": "CLI wiring, never executed during block processing",
	}

	root := repoRootForFloatScan(t)

	scanDirs := []string{
		filepath.Join(root, "x", "structs"),
		filepath.Join(root, "app"),
	}

	fset := token.NewFileSet()
	var failures []string
	filesScanned := 0

	for _, dir := range scanDirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			// Generated code is not ours to edit, and the gogo preamble names
			// math.Inf purely to keep the import alive.
			if strings.HasSuffix(path, ".pb.go") || strings.HasSuffix(path, ".pb.gw.go") ||
				strings.HasSuffix(path, ".pulsar.go") {
				return nil
			}

			rel, relErr := filepath.Rel(root, path)
			require.NoError(t, relErr)
			rel = filepath.ToSlash(rel)

			if _, exempt := allowedFiles[rel]; exempt {
				return nil
			}

			src, readErr := os.ReadFile(path)
			require.NoError(t, readErr)

			file, parseErr := parser.ParseFile(fset, path, src, parser.ParseComments)
			require.NoError(t, parseErr)
			filesScanned++

			// Find what the standard library's math package is called here, if
			// it is imported at all. cosmossdk.io/math is imported as `math`
			// throughout this repository and is a different package entirely -
			// matching on the identifier alone would flag every LegacyDec in the
			// module.
			stdMathName := ""
			for _, imp := range file.Imports {
				path, unquoteErr := strconv.Unquote(imp.Path.Value)
				if unquoteErr != nil || path != "math" {
					continue
				}
				stdMathName = "math"
				if imp.Name != nil {
					stdMathName = imp.Name.Name
				}
			}
			if stdMathName == "" || stdMathName == "_" {
				return nil
			}

			lines := strings.Split(string(src), "\n")
			annotated := func(line int) bool {
				for i := line - 3; i < line && i < len(lines); i++ {
					if i >= 0 && strings.Contains(lines[i], "DETERMINISM_OK:") {
						return true
					}
				}
				return false
			}

			ast.Inspect(file, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok || pkg.Name != stdMathName {
					return true
				}
				kind, bad := transcendentalMathCalls[sel.Sel.Name]
				if !bad {
					return true
				}
				pos := fset.Position(sel.Pos())
				if annotated(pos.Line) {
					return true
				}
				failures = append(failures, rel+":"+strconv.Itoa(pos.Line)+": math."+sel.Sel.Name+
					" is a "+kind+" the toolchain may implement differently per architecture;"+
					" compute it in integers or fixed point")
				return true
			})

			return nil
		})
		require.NoError(t, err)
	}

	// If the walk stops finding files the guard passes while checking nothing.
	require.Greater(t, filesScanned, 100,
		"expected to scan the module sources; the walk is probably broken")

	require.Empty(t, failures,
		"consensus code may not ask the toolchain for a transcendental float:\n  - %s",
		strings.Join(failures, "\n  - "))
}

func repoRootForFloatScan(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "walked to the filesystem root without finding go.mod")
		dir = parent
	}
}
